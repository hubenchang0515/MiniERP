package service

import (
	"crypto/rand"
	"gorm.io/gorm"
	"minierp/internal/model"
	"strings"
	"time"
	"unicode/utf8"
)

const ProviderKeyAvailable = "available"
const ProviderKeyQuota = "quota_exhausted"
const ProviderKeyRateLimit = "rate_limited"
const ProviderKeyInvalid = "invalid"

type AIProviderKeyInput struct {
	ResourceID uint   `json:"resourceId"`
	Name       string `json:"name"`
	APIKey     string `json:"apiKey"`
	Enabled    bool   `json:"enabled"`
}

func (s *Service) SaveAIProviderKey(id uint, in AIProviderKeyInput) (model.AIProviderKey, error) {
	var key model.AIProviderKey
	in.Name = strings.TrimSpace(in.Name)
	if in.ResourceID == 0 || in.Name == "" || utf8.RuneCountInString(in.Name) > 128 || len(in.APIKey) > 8192 || strings.ContainsAny(in.APIKey, "\r\n") {
		return key, Fail(400, "validation", "请选择供应商，填写密钥名称（最多 128 字）及有效的 API KEY")
	}
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		var resource model.AIResource
		if err := tx.First(&resource, in.ResourceID).Error; err != nil {
			return Normalize(err)
		}
		if id != 0 {
			if err := tx.First(&key, id).Error; err != nil {
				return err
			}
			if key.ResourceID != in.ResourceID {
				return Fail(400, "validation", "不能更换密钥所属供应商，请为目标供应商新增密钥")
			}
			key.Revision++
		} else {
			var total int64
			if err := tx.Model(&model.AIProviderKey{}).Where("resource_id = ?", in.ResourceID).Count(&total).Error; err != nil {
				return err
			}
			if total >= 100 {
				return Fail(409, "limit", "每个供应商最多配置 100 个密钥")
			}
			key = model.AIProviderKey{ResourceID: in.ResourceID, State: ProviderKeyAvailable, Revision: 1}
		}
		key.Name, key.Enabled = in.Name, in.Enabled
		if in.APIKey != "" {
			aead, err := s.aiCipher()
			if err != nil {
				return err
			}
			nonce := make([]byte, aead.NonceSize())
			if _, err := rand.Read(nonce); err != nil {
				return err
			}
			key.Secret = aead.Seal(nonce, nonce, []byte(in.APIKey), nil)
			key.State, key.RetryAt = ProviderKeyAvailable, nil
		}
		if len(key.Secret) == 0 {
			return Fail(400, "validation", "新增供应商密钥必须填写 API KEY")
		}
		return tx.Save(&key).Error
	})
	return key, Normalize(err)
}

type ProviderKeyLease struct {
	Key      model.AIProviderKey
	Secret   string
	Probe    bool
	service  *Service
	released bool
}

// Selection and probe admission are serialized; network requests never hold this lock.
func (s *Service) AcquireProviderKey(resourceID uint, excluded map[uint]bool) (*ProviderKeyLease, error) {
	s.ai.poolMu.Lock()
	defer s.ai.poolMu.Unlock()
	if s.ai.cursors == nil {
		s.ai.cursors = map[uint]uint{}
		s.ai.inFlight = map[uint]int{}
	}
	var keys []model.AIProviderKey
	if err := s.DB.Where("resource_id = ? AND enabled = ?", resourceID, true).Order("id").Find(&keys).Error; err != nil {
		return nil, err
	}
	now := time.Now()
	var candidates []model.AIProviderKey
	for _, k := range keys {
		if excluded[k.ID] || k.State == ProviderKeyInvalid {
			continue
		}
		if k.State != ProviderKeyAvailable && (k.RetryAt == nil || k.RetryAt.After(now) || s.ai.inFlight[k.ID] > 0) {
			continue
		}
		candidates = append(candidates, k)
	}
	if len(candidates) == 0 {
		return nil, Fail(503, "no_available_key", "供应商暂无可用密钥，请稍后重试")
	}
	start := 0
	for i, k := range candidates {
		if k.ID > s.ai.cursors[resourceID] {
			start = i
			break
		}
	}
	selected := candidates[start]
	for i := 1; i < len(candidates); i++ {
		k := candidates[(start+i)%len(candidates)]
		if s.ai.inFlight[k.ID] < s.ai.inFlight[selected.ID] {
			selected = k
		}
	}
	secret, err := s.decryptProviderKey(selected)
	if err != nil {
		return nil, err
	}
	if err := s.DB.Model(&model.AIProviderKey{}).Where("id = ?", selected.ID).Updates(map[string]any{"requests": gorm.Expr("requests + 1"), "last_used_at": now}).Error; err != nil {
		return nil, err
	}
	s.ai.cursors[resourceID] = selected.ID
	s.ai.inFlight[selected.ID]++
	return &ProviderKeyLease{Key: selected, Secret: secret, Probe: selected.State != ProviderKeyAvailable, service: s}, nil
}

// A revision check prevents late responses from changing a replaced key or newer health state.
func (l *ProviderKeyLease) Finish(state string, retryAt *time.Time, success bool) error {
	s := l.service
	s.ai.poolMu.Lock()
	defer s.ai.poolMu.Unlock()
	if l.released {
		return nil
	}
	l.released = true
	s.ai.inFlight[l.Key.ID]--
	if s.ai.inFlight[l.Key.ID] == 0 {
		delete(s.ai.inFlight, l.Key.ID)
	}
	if state == "" && !(success && l.Probe) {
		return nil
	}
	if state == "" {
		state = ProviderKeyAvailable
	}
	return s.DB.Model(&model.AIProviderKey{}).Where("id = ? AND revision = ?", l.Key.ID, l.Key.Revision).Updates(map[string]any{"state": state, "retry_at": retryAt, "revision": gorm.Expr("revision + 1")}).Error
}
