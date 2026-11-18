package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"gorm.io/gorm"
	"minierp/internal/model"
	"minierp/internal/repository"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
)

type AIState struct {
	once  sync.Once
	key   []byte
	err   error
	Slots chan struct{}
	pools sync.Map
}

func (s *Service) aiCipher() (cipher.AEAD, error) {
	s.ai.once.Do(func() {
		s.ai.Slots = make(chan struct{}, s.Config.ProxyConcurrency())
		if s.Config.DataDir == "" {
			s.ai.err = errors.New("AI key data directory is missing")
			return
		}
		p := filepath.Join(s.Config.DataDir, ".ai-master-key")
		s.ai.key, s.ai.err = os.ReadFile(p)
		if os.IsNotExist(s.ai.err) {
			key := make([]byte, 32)
			if _, s.ai.err = rand.Read(key); s.ai.err != nil {
				return
			}
			var f *os.File
			f, s.ai.err = os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
			if s.ai.err == nil {
				_, s.ai.err = f.Write(key)
				if err := f.Close(); s.ai.err == nil {
					s.ai.err = err
				}
				s.ai.key = key
			} else if os.IsExist(s.ai.err) {
				s.ai.key, s.ai.err = os.ReadFile(p)
			}
		}
		if s.ai.err == nil && len(s.ai.key) != 32 {
			s.ai.err = errors.New("AI master key is invalid")
		}
	})
	if s.ai.err != nil {
		return nil, s.ai.err
	}
	block, err := aes.NewCipher(s.ai.key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

type AIResourceInput struct {
	Name    string `json:"name"`
	BaseURL string `json:"baseURL"`
	Enabled bool   `json:"enabled"`
}

func parseAIBaseURL(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimRight(strings.TrimSpace(raw), "/"))
	if len(raw) > 2048 || err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, Fail(400, "validation", "请填写有效的 HTTP(S) API 基础地址，地址不能包含凭证或查询参数")
	}
	return u, nil
}

func validAIModelID(value string) bool {
	return value != "" && len(value) <= 256 && strings.IndexFunc(value, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) < 0
}

func (s *Service) SaveAIResource(id uint, in AIResourceInput) (model.AIResource, error) {
	var r model.AIResource
	in.Name = strings.TrimSpace(in.Name)
	in.BaseURL = strings.TrimRight(strings.TrimSpace(in.BaseURL), "/")
	if len(in.Name) == 0 || utf8.RuneCountInString(in.Name) > 128 {
		return r, Fail(400, "validation", "请输入供应商名称（最多 128 个字符）")
	}
	_, err := parseAIBaseURL(in.BaseURL)
	if err != nil {
		return r, err
	}
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		if id != 0 {
			if err := tx.First(&r, id).Error; err != nil {
				return err
			}
		}
		r.Name = in.Name
		r.BaseURL = in.BaseURL
		r.Enabled = in.Enabled
		if err := tx.Omit("Keys").Save(&r).Error; err != nil {
			return err
		}
		return tx.Order("id").Where("resource_id = ?", r.ID).Find(&r.Keys).Error
	})
	return r, Normalize(err)
}
func newAISecret() (string, string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	raw := "mierp_" + base64.RawURLEncoding.EncodeToString(b)
	return raw, hashAISecret(raw), nil
}
func hashAISecret(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

type AIKeyInput struct {
	Name string `json:"name"`
}

const MaxActiveAIKeys = 10

func ActiveAIKeyCount(db *gorm.DB, userID uint) (int64, error) {
	var count int64
	err := db.Model(&model.AIKey{}).Where("user_id = ? AND enabled = ?", userID, true).Count(&count).Error
	return count, err
}

func (s *Service) CreateAIKey(userID uint, in AIKeyInput) (model.AIKey, string, error) {
	var k model.AIKey
	in.Name = strings.TrimSpace(in.Name)
	if userID == 0 {
		return k, "", Fail(400, "validation", "用户不存在")
	}
	if in.Name == "" || utf8.RuneCountInString(in.Name) > 128 {
		return k, "", Fail(400, "validation", "请输入密钥名称（最多 128 个字符）")
	}
	raw, hash, err := newAISecret()
	if err != nil {
		return k, "", err
	}
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		u, err := repository.User(tx, userID)
		if err != nil {
			return Fail(400, "validation", "用户不存在")
		}
		if !u.Enabled || !Can(u, "ai:use") {
			return Fail(403, "forbidden", "当前账号没有使用 AI 的权限")
		}
		count, err := ActiveAIKeyCount(tx, userID)
		if err != nil {
			return err
		}
		if count >= MaxActiveAIKeys {
			return Fail(409, "ai_key_limit", "最多可持有 10 个有效密钥，请先撤销一个再创建")
		}
		k = model.AIKey{Name: in.Name, UserID: u.ID, TokenHash: hash, Prefix: raw[:16] + "…", Enabled: true}
		return tx.Create(&k).Error
	})
	if err != nil {
		return model.AIKey{}, "", Normalize(err)
	}
	return k, raw, nil
}
func (s *Service) ChangeAIKey(id, owner uint, rotate bool) (string, error) {
	if rotate && owner == 0 {
		return "", Fail(403, "forbidden", "只能重新生成本人的密钥")
	}
	raw, hash, err := newAISecret()
	if err != nil {
		return "", err
	}
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		var k model.AIKey
		query := tx
		if owner != 0 {
			query = query.Where("user_id = ?", owner)
		}
		if err := query.First(&k, id).Error; err != nil {
			return err
		}
		if !k.Enabled {
			return Fail(409, "revoked", "密钥已撤销，不能重新生成")
		}
		if rotate {
			u, err := repository.User(tx, k.UserID)
			if err != nil {
				return err
			}
			if !u.Enabled || !Can(u, "ai:use") {
				return Fail(403, "forbidden", "当前账号没有使用 AI 的权限")
			}
			return tx.Model(&k).Updates(map[string]any{"token_hash": hash, "prefix": raw[:16] + "…"}).Error
		}
		return tx.Model(&k).Update("enabled", false).Error
	})
	if !rotate {
		raw = ""
	}
	return raw, Normalize(err)
}
func (s *Service) ResolveAIKey(raw string) (model.AIKey, error) {
	var k model.AIKey
	if !strings.HasPrefix(raw, "mierp_") || len(raw) > 128 {
		return k, Fail(401, "invalid_key", "密钥无效或已撤销")
	}
	var err error
	k, err = repository.AIKeyAccess(s.ReadDB(), hashAISecret(raw))
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return k, err
	}
	if err != nil || !k.User.Enabled {
		return k, Fail(401, "invalid_key", "密钥或用户无效或已停用")
	}
	if !Can(k.User, "ai:use") {
		return k, Fail(403, "forbidden", "所属用户没有使用 AI 的权限")
	}
	return k, nil
}

func (s *Service) decryptProviderKey(key model.AIProviderKey) (string, error) {
	aead, err := s.aiCipher()
	if err != nil {
		return "", err
	}
	if len(key.Secret) < aead.NonceSize() {
		return "", errors.New("invalid AI ciphertext")
	}
	secret, err := aead.Open(nil, key.Secret[:aead.NonceSize()], key.Secret[aead.NonceSize():], nil)
	return string(secret), err
}
func (s *Service) RecordAIRequest(id uint) error {
	return s.recordUsage(false, id, time.Now())
}

// Never retain request/response bodies or plaintext credentials in usage records.
func (s *Service) AcquireAIProxy() (func(), error) {
	if _, err := s.aiCipher(); err != nil {
		return nil, err
	}
	select {
	case s.ai.Slots <- struct{}{}:
		var once sync.Once
		return func() { once.Do(func() { <-s.ai.Slots }) }, nil
	default:
		return nil, &Error{Status: 429, Code: "busy", Message: "AI 请求繁忙，请稍后重试", RetryAfter: time.Second}
	}
}
