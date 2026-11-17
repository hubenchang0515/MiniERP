package handler_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
	"minierp/internal/service"
)

func TestReadAPIsAndProxyProceedDuringWriteTransaction(t *testing.T) {
	r, s := server(t)
	s.Config.DataDir = t.TempDir()
	a := admin(t, r)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, `{"ok":true}`) }))
	defer upstream.Close()
	resource, err := saveProviderTestResource(s, service.AIResourceInput{Name: "test", BaseURL: upstream.URL + "/v1", Enabled: true}, []service.AIProviderKeyInput{{Name: "key", APIKey: "dummy", Enabled: true}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveAIModel(0, service.AIModelInput{InternalID: "test", ExternalID: "upstream", ResourceID: resource.ID, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	_, token, err := s.CreateAIKey(a.User.ID, service.AIKeyInput{Name: "test"})
	if err != nil {
		t.Fatal(err)
	}
	locked, release := make(chan struct{}), make(chan struct{})
	txDone := make(chan error, 1)
	go func() { txDone <- s.DB.Transaction(func(tx *gorm.DB) error { close(locked); <-release; return nil }) }()
	<-locked
	defer func() {
		close(release)
		if err := <-txDone; err != nil {
			t.Error(err)
		}
	}()
	for _, test := range []struct{ method, path, body, key string }{
		{"GET", "/api/users", "", a.Token}, {"GET", "/api/roles", "", a.Token}, {"GET", "/api/ai/resources", "", a.Token},
		{"GET", "/api/ai/provider-keys", "", a.Token}, {"GET", "/api/ai/models", "", a.Token}, {"GET", "/api/ai/keys", "", a.Token},
		{"POST", "/ai/v1/chat/completions", `{"model":"test"}`, token},
	} {
		request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
		request.Header.Set("Authorization", "Bearer "+test.key)
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		request = request.WithContext(ctx)
		done := make(chan *httptest.ResponseRecorder, 1)
		go func() { w := httptest.NewRecorder(); r.ServeHTTP(w, request); done <- w }()
		select {
		case w := <-done:
			if w.Code != 200 {
				t.Errorf("%s: status=%d", test.path, w.Code)
			}
		case <-ctx.Done():
			t.Errorf("%s waited for an unrelated writer", test.path)
			cancel()
			return
		}
		cancel()
	}
}
