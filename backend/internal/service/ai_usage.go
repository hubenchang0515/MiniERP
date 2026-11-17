package service

import (
	"context"
	"log"
	"sync"
	"time"

	"gorm.io/gorm"
	"minierp/internal/model"
)

type usageID struct {
	provider bool
	id       uint
}
type usageDelta struct {
	count uint64
	last  time.Time
}
type usageWriter struct {
	mu      sync.Mutex
	flushMu sync.Mutex
	pending map[usageID]usageDelta
	running bool
	stopped bool
	stop    chan struct{}
	done    chan struct{}
}

// StartUsageWriter moves non-billing counters off request paths. Changes are
// flushed once per second and drained on graceful shutdown.
func (s *Service) StartUsageWriter() {
	u := &s.usage
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.running || u.stopped {
		return
	}
	u.pending = map[usageID]usageDelta{}
	u.stop = make(chan struct{})
	u.done = make(chan struct{})
	u.running = true
	go func() {
		defer close(u.done)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				if err := s.FlushUsage(ctx); err != nil {
					log.Printf("AI usage flush delayed: %v", err)
				}
				cancel()
			case <-u.stop:
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				if err := s.FlushUsage(ctx); err != nil {
					log.Printf("AI usage shutdown flush failed: %v", err)
				}
				cancel()
				return
			}
		}
	}()
}

func (s *Service) StopUsageWriter() {
	u := &s.usage
	u.mu.Lock()
	if !u.running {
		u.mu.Unlock()
		return
	}
	if !u.stopped {
		u.stopped = true
		close(u.stop)
	}
	done := u.done
	u.mu.Unlock()
	<-done
}

func (s *Service) recordUsage(provider bool, id uint, at time.Time) error {
	u := &s.usage
	u.mu.Lock()
	if u.running && !u.stopped {
		key := usageID{provider, id}
		delta := u.pending[key]
		delta.count++
		if at.After(delta.last) {
			delta.last = at
		}
		u.pending[key] = delta
		u.mu.Unlock()
		return nil
	}
	u.mu.Unlock()
	return applyUsage(s.DB, usageID{provider, id}, usageDelta{1, at})
}

func applyUsage(db *gorm.DB, key usageID, delta usageDelta) error {
	var record any = &model.AIKey{}
	if key.provider {
		record = &model.AIProviderKey{}
	}
	return db.Model(record).Where("id = ?", key.id).Updates(map[string]any{
		"requests":     gorm.Expr("requests + ?", delta.count),
		"last_used_at": gorm.Expr("CASE WHEN last_used_at IS NULL OR last_used_at < ? THEN ? ELSE last_used_at END", delta.last, delta.last),
	}).Error
}

func (s *Service) FlushUsage(ctx context.Context) error {
	u := &s.usage
	u.flushMu.Lock()
	defer u.flushMu.Unlock()
	u.mu.Lock()
	batch := u.pending
	u.pending = map[usageID]usageDelta{}
	u.mu.Unlock()
	if len(batch) == 0 {
		return nil
	}
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for key, delta := range batch {
			if err := applyUsage(tx, key, delta); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		u.mu.Lock()
		for key, delta := range batch {
			current := u.pending[key]
			current.count += delta.count
			if delta.last.After(current.last) {
				current.last = delta.last
			}
			u.pending[key] = current
		}
		u.mu.Unlock()
	}
	return err
}
