package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"minioa/internal/model"
)

func TestUsageBufferDoesNotWaitForWriterAndDrains(t *testing.T) {
	s := newService(t)
	s.Config.DataDir = t.TempDir()
	u := bootstrap(t, s)
	key, _, err := s.CreateAIKey(u.User.ID, AIKeyInput{Name: "usage"})
	if err != nil {
		t.Fatal(err)
	}
	r, err := providerTestResource(s, AIResourceInput{Name: "provider", BaseURL: "https://example.com/v1", Enabled: true}, []AIProviderKeyInput{{Name: "one", APIKey: "dummy", Enabled: true}})
	if err != nil {
		t.Fatal(err)
	}
	s.StartUsageWriter()
	t.Cleanup(s.StopUsageWriter)
	pool, _ := s.DB.DB()
	conn, err := pool.Conn(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	// Owning the only write connection must not block request-side counters.
	done := make(chan error, 1)
	go func() {
		for range 100 {
			if err := s.RecordAIRequest(key.ID); err != nil {
				done <- err
				return
			}
			if err := s.recordUsage(true, r.Keys[0].ID, time.Now()); err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		conn.Close()
		t.Fatal("usage recording waited for a DB write")
	}
	conn.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	if err := s.FlushUsage(ctx); err != nil {
		t.Fatal(err)
	}
	var internal model.AIKey
	var provider model.AIProviderKey
	if err := s.DB.First(&internal, key.ID).Error; err != nil || internal.Requests != 100 || internal.LastUsedAt == nil {
		t.Fatalf("internal usage not persisted: count=%d err=%v", internal.Requests, err)
	}
	if err := s.DB.First(&provider, r.Keys[0].ID).Error; err != nil || provider.Requests != 100 || provider.LastUsedAt == nil {
		t.Fatal("provider usage not persisted")
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 20 {
				_ = s.RecordAIRequest(key.ID)
			}
		}()
	}
	wg.Wait()
	s.StopUsageWriter()
	if err := s.DB.First(&internal, key.ID).Error; err != nil || internal.Requests != 260 {
		t.Fatalf("shutdown lost concurrent usage: %d %v", internal.Requests, err)
	}
}

func TestUsageFailedFlushRetainsCounters(t *testing.T) {
	s := newService(t)
	u := bootstrap(t, s)
	key, _, err := s.CreateAIKey(u.User.ID, AIKeyInput{Name: "usage"})
	if err != nil {
		t.Fatal(err)
	}
	s.StartUsageWriter()
	t.Cleanup(s.StopUsageWriter)
	if err := s.RecordAIRequest(key.ID); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := s.FlushUsage(ctx); err == nil {
		t.Fatal("canceled flush succeeded")
	}
	if err := s.FlushUsage(t.Context()); err != nil {
		t.Fatal(err)
	}
	var found model.AIKey
	if err := s.DB.First(&found, key.ID).Error; err != nil || found.Requests != 1 {
		t.Fatal("failed flush lost or duplicated a counter")
	}
}
