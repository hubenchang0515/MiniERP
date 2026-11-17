package handler_test

import (
	"encoding/json"
	"minierp/internal/model"
	"minierp/internal/service"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAIProviderPoolFailoverAndRecovery(t *testing.T) {
	r, s := server(t)
	s.Config.DataDir = t.TempDir()
	admin := admin(t, r)
	mode := map[string]string{"first-secret": "ok", "second-secret": "ok"}
	calls := map[string]int{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		secret := strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer ")
		calls[secret]++
		if req.URL.Path == "/v1/models" {
			w.Write([]byte(`{"data":[{"id":"provider-chat"}]}`))
			return
		}
		if mode[secret] == "quota" {
			if secret == "first-secret" {
				w.Header().Set("Retry-After", "3600")
			}
			w.WriteHeader(429)
			w.Write([]byte(`{"error":{"code":"insufficient_quota","message":"` + secret + `"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()
	resource, err := saveProviderTestResource(s, service.AIResourceInput{Name: "pool", BaseURL: upstream.URL + "/v1", Enabled: true}, []service.AIProviderKeyInput{{Name: "first", APIKey: "first-secret", Enabled: true}, {Name: "second", APIKey: "second-secret", Enabled: true}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveAIModel(0, service.AIModelInput{InternalID: "chat", ResourceID: resource.ID, ExternalID: "provider-chat", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	_, key, err := s.CreateAIKey(admin.User.ID, service.AIKeyInput{Name: "client"})
	if err != nil {
		t.Fatal(err)
	}
	invoke := func(status int) string {
		t.Helper()
		return call(t, r, "POST", "/ai/v1/chat/completions", `{"model":"chat","stream":false}`, key, status).Body.String()
	}
	for range 8 {
		invoke(200)
	}
	if calls["first-secret"] != 4 || calls["second-secret"] != 4 {
		t.Fatalf("sequential requests not balanced: %v", calls)
	}
	mode["first-secret"] = "quota"
	invoke(200)
	if calls["first-secret"] != 5 || calls["second-secret"] != 5 {
		t.Fatalf("quota rejection did not fail over: %v", calls)
	}
	var first model.AIProviderKey
	if err := s.DB.First(&first, resource.Keys[0].ID).Error; err != nil || first.State != service.ProviderKeyQuota || first.RetryAt == nil || time.Until(*first.RetryAt) < 59*time.Minute {
		t.Fatalf("quota state or retry time missing: %+v %v", first, err)
	}
	invoke(200)
	if calls["first-secret"] != 5 {
		t.Fatal("exhausted key was reused before cooldown")
	}
	// A model-list success cannot prove that an inference quota was refreshed.
	ago := time.Now().Add(-time.Minute)
	if err := s.DB.Model(&model.AIProviderKey{}).Where("id = ?", first.ID).Update("retry_at", ago).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.DB.Model(&model.AIProviderKey{}).Where("id = ?", resource.Keys[1].ID).Update("enabled", false).Error; err != nil {
		t.Fatal(err)
	}
	_, err = s.FetchAIModels(t.Context(), service.AIModelListInput{ResourceID: resource.ID})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.DB.First(&first, first.ID).Error; err != nil || first.State != service.ProviderKeyQuota {
		t.Fatal("metadata request incorrectly restored quota")
	}
	mode["first-secret"] = "ok"
	invoke(200)
	first = model.AIProviderKey{ID: first.ID}
	if err := s.DB.First(&first, first.ID).Error; err != nil || first.State != service.ProviderKeyAvailable || first.RetryAt != nil {
		t.Fatalf("successful inference probe: state=%s retry=%v revision=%d", first.State, first.RetryAt, first.Revision)
	}
	if err := s.DB.Model(&model.AIProviderKey{}).Where("id = ?", resource.Keys[1].ID).Update("enabled", true).Error; err != nil {
		t.Fatal(err)
	}
	mode["first-secret"], mode["second-secret"] = "quota", "quota"
	if body := invoke(429); strings.Contains(body, "secret") {
		t.Fatal("quota response exposed upstream credential")
	}
	before := calls["first-secret"] + calls["second-secret"]
	invoke(503)
	if calls["first-secret"]+calls["second-secret"] != before {
		t.Fatal("unavailable keys still contacted upstream")
	}
	var second model.AIProviderKey
	if err := s.DB.First(&second, resource.Keys[1].ID).Error; err != nil || second.RetryAt == nil || time.Until(*second.RetryAt) < 4*time.Minute || time.Until(*second.RetryAt) > 6*time.Minute {
		t.Fatal("missing reset hint did not use five-minute fallback")
	}
	if err := s.DB.Model(&model.AIProviderKey{}).Where("id = ?", second.ID).Update("retry_at", ago).Error; err != nil {
		t.Fatal(err)
	}
	mode["second-secret"] = "ok"
	invoke(200)
	list := call(t, r, "GET", "/api/ai/resources", "", admin.Token, 200).Body.String()
	if strings.Contains(list, "first-secret") || strings.Contains(list, "second-secret") {
		t.Fatal("supplier list exposed credentials")
	}
	var metadata struct {
		Items []model.AIResource `json:"items"`
	}
	if err := json.Unmarshal([]byte(list), &metadata); err != nil || len(metadata.Items[0].Keys) != 2 {
		t.Fatal("supplier key metadata missing")
	}
}
