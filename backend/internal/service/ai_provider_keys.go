package service

import (
	"crypto/rand"
	"gorm.io/gorm"
	"minioa/internal/model"
	"strings"
	"sync"
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
	pool     *providerPool
	released bool
}

type providerPool struct {
	mu       sync.Mutex
	cursor   uint
	inFlight map[uint]int
}

func (s *Service) providerPool(id uint) *providerPool {
	if pool, ok := s.ai.pools.Load(id); ok {
		return pool.(*providerPool)
	}
	pool, _ := s.ai.pools.LoadOrStore(id, &providerPool{inFlight: map[uint]int{}})
	return pool.(*providerPool)
}

// Selection and probe admission are serialized; network requests never hold this lock.
func (s *Service) AcquireProviderKey(resourceID uint, excluded map[uint]bool) (*ProviderKeyLease, error) {
	pool := s.providerPool(resourceID)
	pool.mu.Lock()
	defer pool.mu.Unlock()
	var keys []model.AIProviderKey
	if err := s.ReadDB().Where("resource_id = ? AND enabled = ?", resourceID, true).Order("id").Find(&keys).Error; err != nil {
		return nil, err
	}
	now := time.Now()
	var candidates []model.AIProviderKey
	var earliestRetry time.Time
	for _, k := range keys {
		if excluded[k.ID] || k.State == ProviderKeyInvalid {
			continue
		}
		if k.State != ProviderKeyAvailable && (k.RetryAt == nil || k.RetryAt.After(now) || pool.inFlight[k.ID] > 0) {
			if k.RetryAt != nil && (earliestRetry.IsZero() || k.RetryAt.Before(earliestRetry)) {
				earliestRetry = *k.RetryAt
			}
			continue
		}
		candidates = append(candidates, k)
	}
	if len(candidates) == 0 {
		var delay time.Duration
		if !earliestRetry.IsZero() {
			delay = time.Second
			if earliestRetry.After(now) {
				delay = earliestRetry.Sub(now)
			}
		}
		return nil, &Error{Status: 503, Code: "no_available_key", Message: "供应商暂无可用密钥，请稍后重试", RetryAfter: delay}
	}
	start := 0
	for i, k := range candidates {
		if k.ID > pool.cursor {
			start = i
			break
		}
	}
	selected := candidates[start]
	for i := 1; i < len(candidates); i++ {
		k := candidates[(start+i)%len(candidates)]
		if pool.inFlight[k.ID] < pool.inFlight[selected.ID] {
			selected = k
		}
	}
	secret, err := s.decryptProviderKey(selected)
	if err != nil {
		return nil, err
	}
	if err := s.recordUsage(true, selected.ID, now); err != nil {
		return nil, err
	}
	pool.cursor = selected.ID
	pool.inFlight[selected.ID]++
	return &ProviderKeyLease{Key: selected, Secret: secret, Probe: selected.State != ProviderKeyAvailable, service: s, pool: pool}, nil
}

// A revision check prevents late responses from changing a replaced key or newer health state.
func (l *ProviderKeyLease) Finish(state string, retryAt *time.Time, success bool) error {
	s := l.service
	l.pool.mu.Lock()
	defer l.pool.mu.Unlock()
	if l.released {
		return nil
	}
	l.released = true
	l.pool.inFlight[l.Key.ID]--
	if l.pool.inFlight[l.Key.ID] == 0 {
		delete(l.pool.inFlight, l.Key.ID)
	}
	if state == "" && !(success && l.Probe) {
		return nil
	}
	if state == "" {
		state = ProviderKeyAvailable
	}
	return s.DB.Model(&model.AIProviderKey{}).Where("id = ? AND revision = ?", l.Key.ID, l.Key.Revision).Updates(map[string]any{"state": state, "retry_at": retryAt, "revision": gorm.Expr("revision + 1")}).Error
}
