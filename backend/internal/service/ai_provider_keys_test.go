package service

import (
	"minierp/internal/model"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestProviderPoolBalanceProbeAndStaleFeedback(t *testing.T) {
	s := newService(t)
	s.Config.DataDir = t.TempDir()
	r, err := providerTestResource(s, AIResourceInput{Name: "pool", BaseURL: "https://example.com/v1", Enabled: true}, []AIProviderKeyInput{{Name: "one", APIKey: "first-secret", Enabled: true}, {Name: "two", APIKey: "second-secret", Enabled: true}})
	if err != nil {
		t.Fatal(err)
	}
	var held []*ProviderKeyLease
	counts := map[uint]int{}
	for range 8 {
		lease, err := s.AcquireProviderKey(r.ID, nil)
		if err != nil {
			t.Fatal(err)
		}
		held = append(held, lease)
		counts[lease.Key.ID]++
	}
	if counts[r.Keys[0].ID] != 4 || counts[r.Keys[1].ID] != 4 {
		t.Fatalf("in-flight work not balanced: %v", counts)
	}
	next := time.Now().Add(time.Hour)
	if err := held[0].Finish(ProviderKeyQuota, &next, false); err != nil {
		t.Fatal(err)
	}
	for _, lease := range held[1:] {
		if err := lease.Finish("", nil, true); err != nil {
			t.Fatal(err)
		}
	}
	var paused model.AIProviderKey
	if err := s.DB.First(&paused, r.Keys[0].ID).Error; err != nil || paused.State != ProviderKeyQuota {
		t.Fatal("late successful response cleared exhausted state")
	}
	// Reopened services see persisted health; there is no in-memory-only cooldown.
	restarted := &Service{DB: s.DB, Config: s.Config}
	lease, err := restarted.AcquireProviderKey(r.ID, nil)
	if err != nil || lease.Key.ID != r.Keys[1].ID {
		t.Fatalf("paused key selected after restart: %v", err)
	}
	if err := lease.Finish("", nil, true); err != nil {
		t.Fatal(err)
	}
	if err := s.DB.Model(&model.AIProviderKey{}).Where("id = ?", r.Keys[1].ID).Update("enabled", false).Error; err != nil {
		t.Fatal(err)
	}
	ago := time.Now().Add(-time.Minute)
	if err := s.DB.Model(&model.AIProviderKey{}).Where("id = ?", paused.ID).Update("retry_at", ago).Error; err != nil {
		t.Fatal(err)
	}
	// Holding the admitted probe excludes every other concurrent caller.
	probe, err := s.AcquireProviderKey(r.ID, nil)
	if err != nil || !probe.Probe {
		t.Fatalf("due key was not admitted as a probe: %v", err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 16)
	for range 16 {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := s.AcquireProviderKey(r.ID, nil); results <- err }()
	}
	wg.Wait()
	close(results)
	for err := range results {
		wantStatus(t, err, 503)
	}
	if err := probe.Finish("", nil, true); err != nil {
		t.Fatal(err)
	}
	paused = model.AIProviderKey{ID: paused.ID}
	if err := s.DB.First(&paused, paused.ID).Error; err != nil || paused.State != ProviderKeyAvailable || paused.RetryAt != nil {
		t.Fatalf("successful probe: state=%s retry=%v revision=%d probeRevision=%d", paused.State, paused.RetryAt, paused.Revision, probe.Key.Revision)
	}
	old, err := s.AcquireProviderKey(r.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.SaveAIProviderKey(paused.ID, AIProviderKeyInput{ResourceID: r.ID, Name: "replaced", APIKey: "new-secret", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := old.Finish(ProviderKeyQuota, &next, false); err != nil {
		t.Fatal(err)
	}
	fresh, err := s.AcquireProviderKey(r.ID, nil)
	if err != nil || fresh.Secret != "new-secret" || fresh.Probe {
		t.Fatalf("old response affected replacement key: %v", err)
	}
	if err := fresh.Finish("", nil, true); err != nil {
		t.Fatal(err)
	}
}

func TestProviderKeyOwnershipAndUpdates(t *testing.T) {
	s := newService(t)
	s.Config.DataDir = t.TempDir()
	create := func(name, secret string) model.AIResource {
		t.Helper()
		r, err := providerTestResource(s, AIResourceInput{Name: name, BaseURL: "https://example.com/v1", Enabled: true}, []AIProviderKeyInput{{Name: "key", APIKey: secret, Enabled: true}})
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	one, two := create("one", "first-secret"), create("two", "second-secret")
	if string(one.Keys[0].Secret) == "first-secret" || len(one.Keys[0].Secret) == 0 {
		t.Fatal("provider credential not encrypted")
	}
	update := AIProviderKeyInput{ResourceID: one.ID, Name: "renamed", Enabled: true}
	changed, err := s.SaveAIProviderKey(one.Keys[0].ID, update)
	if err != nil || string(changed.Secret) != string(one.Keys[0].Secret) {
		t.Fatal("empty key did not preserve encrypted credential")
	}
	update.ResourceID = two.ID
	_, err = s.SaveAIProviderKey(one.Keys[0].ID, update)
	wantStatus(t, err, 400)
	update.ResourceID = one.ID
	_, err = s.SaveAIProviderKey(0, update)
	wantStatus(t, err, 400)
	update.Enabled = false
	if _, err := s.SaveAIProviderKey(one.Keys[0].ID, update); err != nil {
		t.Fatal(err)
	}
	_, err = s.AcquireProviderKey(one.ID, nil)
	wantStatus(t, err, 503)
	var count int64
	if err := s.DB.Model(&model.AIProviderKey{}).Count(&count).Error; err != nil || count != 2 {
		t.Fatal("invalid key updates changed ownership or row count")
	}
}

func TestProviderFailureClassification(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	for _, test := range []struct {
		name        string
		status      int
		header      http.Header
		body, state string
		wait        time.Duration
	}{
		{"quota fallback", 429, nil, `{"error":{"code":"insufficient_quota"}}`, ProviderKeyQuota, 5 * time.Minute},
		{"quota ignores rate reset", 429, http.Header{"X-Ratelimit-Reset-Requests": []string{"1s"}}, `{"error":{"type":"insufficient_quota"}}`, ProviderKeyQuota, 5 * time.Minute},
		{"retry seconds", 429, http.Header{"Retry-After": []string{"120"}}, `{}`, ProviderKeyRateLimit, 2 * time.Minute},
		{"retry date", 402, http.Header{"Retry-After": []string{now.Add(time.Hour).UTC().Format(http.TimeFormat)}}, `{}`, ProviderKeyQuota, time.Hour},
		{"rate duration", 429, http.Header{"X-Ratelimit-Reset-Tokens": []string{"6m30s"}}, `{}`, ProviderKeyRateLimit, 390 * time.Second},
		{"invalid", 401, nil, `{}`, ProviderKeyInvalid, 0},
		{"ordinary forbidden", 403, nil, `{"error":{"code":"model_not_allowed"}}`, "", 0},
		{"server failure", 500, nil, `{}`, "", 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			state, retry := providerFailure(test.status, test.header, []byte(test.body), now)
			if state != test.state {
				t.Fatalf("state = %s", state)
			}
			if test.wait == 0 {
				if retry != nil {
					t.Fatal("unexpected retry")
				}
			} else if retry == nil || retry.Sub(now) != test.wait {
				t.Fatalf("retry = %v", retry)
			}
		})
	}
}

func providerTestResource(s *Service, input AIResourceInput, keys []AIProviderKeyInput) (model.AIResource, error) {
	r, err := s.SaveAIResource(0, input)
	if err != nil {
		return r, err
	}
	for _, input := range keys {
		input.ResourceID = r.ID
		key, err := s.SaveAIProviderKey(0, input)
		if err != nil {
			return r, err
		}
		r.Keys = append(r.Keys, key)
	}
	return r, nil
}
