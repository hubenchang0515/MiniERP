package handler_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"minioa/internal/service"
)

func TestLatencyDiagnostic(t *testing.T) {
	previous := gin.DefaultWriter
	gin.DefaultWriter = io.Discard
	defer func() { gin.DefaultWriter = previous }()
	r, s := server(t)
	s.Config.DataDir = t.TempDir()
	a := admin(t, r)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer upstream.Close()
	resource, err := saveProviderTestResource(s, service.AIResourceInput{Name: "test", BaseURL: upstream.URL + "/v1", Enabled: true}, []service.AIProviderKeyInput{{Name: "key", APIKey: "dummy-test-key", Enabled: true}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveAIModel(0, service.AIModelInput{InternalID: "test", ExternalID: "upstream", ResourceID: resource.ID, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	_, key, err := s.CreateAIKey(a.User.ID, service.AIKeyInput{Name: "test"})
	if err != nil {
		t.Fatal(err)
	}
	var queries, updates atomic.Int64
	for _, db := range []*gorm.DB{s.DB, s.ReadDB()} {
		_ = db.Callback().Query().Before("gorm:query").Register("diag:query", func(*gorm.DB) { queries.Add(1) })
		_ = db.Callback().Row().Before("gorm:row").Register("diag:row", func(*gorm.DB) { queries.Add(1) })
		_ = db.Callback().Update().Before("gorm:update").Register("diag:update", func(*gorm.DB) { updates.Add(1) })
	}
	measure := func(label string, f func() error) {
		t.Helper()
		q, u := queries.Load(), updates.Load()
		start := time.Now()
		if err := f(); err != nil {
			t.Fatal(err)
		}
		t.Logf("PHASE %s time=%s reads=%d writes=%d", label, time.Since(start), queries.Load()-q, updates.Load()-u)
	}
	measure("auth", func() error { _, err := s.ResolveAIKey(key); return err })
	measure("model", func() error { _, err := s.ResolveAIModel("test"); return err })
	measure("user-stats", func() error { return s.RecordAIRequest(a.User.ID) })
	measure("provider-select", func() error {
		k, err := s.AcquireProviderKey(resource.ID, nil)
		if err == nil {
			err = k.Finish("", nil, true)
		}
		return err
	})
	invoke := func(path, token string) time.Duration {
		req := httptest.NewRequest("POST", path, strings.NewReader(`{"model":"test"}`))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		start := time.Now()
		r.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("proxy status=%d", w.Code)
		}
		return time.Since(start)
	}
	for _, parallel := range []int{1, 16} {
		sqlDB, _ := s.DB.DB()
		before := sqlDB.Stats()
		var mu sync.Mutex
		var times []time.Duration
		var wg sync.WaitGroup
		for range parallel {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for range 10 {
					elapsed := invoke("/ai/v1/chat/completions", key)
					mu.Lock()
					times = append(times, elapsed)
					mu.Unlock()
				}
			}()
		}
		wg.Wait()
		sort.Slice(times, func(i, j int) bool { return times[i] < times[j] })
		after := sqlDB.Stats()
		t.Logf("PROXY clients=%d p50=%s p95=%s sql_waits=%d sql_wait_total=%s", parallel, times[len(times)/2], times[len(times)*95/100], after.WaitCount-before.WaitCount, after.WaitDuration-before.WaitDuration)
	}
	locked := make(chan struct{})
	unlock := make(chan struct{})
	done := make(chan error, 1)
	go func() { done <- s.DB.Transaction(func(tx *gorm.DB) error { close(locked); <-unlock; return nil }) }()
	<-locked
	go func() { time.Sleep(300 * time.Millisecond); close(unlock) }()
	request := httptest.NewRequest("GET", "/api/users", nil)
	request.Header.Set("Authorization", "Bearer "+a.Token)
	w := httptest.NewRecorder()
	start := time.Now()
	r.ServeHTTP(w, request)
	t.Logf("ADMIN_READ_DURING_WRITE elapsed=%s status=%d", time.Since(start), w.Code)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
