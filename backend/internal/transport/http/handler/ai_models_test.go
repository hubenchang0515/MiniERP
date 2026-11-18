package handler_test

import (
	"encoding/json"
	"minioa/internal/model"
	"minioa/internal/service"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestAIFetchUpstreamModels(t *testing.T) {
	r, s := server(t)
	s.Config.DataDir = t.TempDir()
	a := admin(t, r)
	const savedSecret = "external-model-list-key"
	expectedSecret, mode := savedSecret, "ok"
	calls := 0
	redirectCalls := 0
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		redirectCalls++
		w.Write([]byte(`{"data":[]}`))
	}))
	defer redirect.Close()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		calls++
		if req.Method != "GET" || req.URL.Path != "/custom/v1/models" || req.Header.Get("Authorization") != "Bearer "+expectedSecret {
			t.Error("upstream model request did not use the configured path and credential")
		}
		if req.Header.Get("Cookie") != "" {
			t.Error("browser credentials were forwarded")
		}
		switch mode {
		case "unauthorized":
			w.WriteHeader(401)
			w.Write([]byte(savedSecret))
		case "redirect":
			http.Redirect(w, req, redirect.URL, 302)
		case "invalid":
			w.Write([]byte(`{"error":"` + savedSecret + `"}`))
		case "large":
			w.Write([]byte(strings.Repeat("x", (2<<20)+1)))
		case "empty":
			w.Write([]byte(`{"data":[]}`))
		default:
			w.Write([]byte(`{"data":[{"id":"provider-b","private":"` + savedSecret + `"},{"id":"provider-a"},{"id":"provider-b"},{"id":""},{"id":"invalid space"}]}`))
		}
	}))
	defer upstream.Close()
	baseURL := upstream.URL + "/custom/v1"
	resource, err := saveProviderTestResource(s, service.AIResourceInput{Name: "saved", BaseURL: baseURL, Enabled: true}, []service.AIProviderKeyInput{{Name: "primary", APIKey: savedSecret, Enabled: true}})
	if err != nil {
		t.Fatal(err)
	}
	invoke := func(in service.AIModelListInput, status int) string {
		t.Helper()
		body, _ := json.Marshal(in)
		return call(t, r, "POST", "/api/ai/resources/models", string(body), a.Token, status).Body.String()
	}
	in := service.AIModelListInput{ResourceID: resource.ID}
	body := invoke(in, 200)
	var result struct {
		ModelIDs []string `json:"modelIds"`
	}
	if err := json.Unmarshal([]byte(body), &result); err != nil || !reflect.DeepEqual(result.ModelIDs, []string{"provider-a", "provider-b"}) || strings.Contains(body, savedSecret) {
		t.Fatalf("model list was not sanitized and deduplicated: %s %v", body, err)
	}
	// Unsaved credentials work for both new resources and edits, without persisting them.
	expectedSecret = "unsaved-key"
	invoke(service.AIModelListInput{BaseURL: baseURL + "/", APIKey: expectedSecret}, 200)
	invoke(service.AIModelListInput{ResourceID: resource.ID, BaseURL: baseURL, APIKey: expectedSecret}, 200)
	var stored model.AIResource
	if err := s.DB.Preload("Keys").First(&stored, resource.ID).Error; err != nil || string(stored.Keys[0].Secret) != string(resource.Keys[0].Secret) {
		t.Fatal("fetching model IDs changed the resource")
	}
	expectedSecret = savedSecret
	beforeInvalid := calls
	invoke(service.AIModelListInput{BaseURL: baseURL}, 400)
	invoke(service.AIModelListInput{BaseURL: "file:///tmp/models", APIKey: savedSecret}, 400)
	invoke(service.AIModelListInput{BaseURL: baseURL, APIKey: "bad\nkey"}, 400)
	invoke(service.AIModelListInput{ResourceID: resource.ID, BaseURL: redirect.URL + "/v1"}, 400)
	invoke(service.AIModelListInput{ResourceID: resource.ID + 100, BaseURL: baseURL}, 404)
	call(t, r, "POST", "/api/ai/resources/models", `{}`, "", 401)
	u, err := s.CreateUser(service.UserInput{Username: "ai-model-reader", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	login, err := s.Login(service.Credentials{Username: u.Username, Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	call(t, r, "POST", "/api/ai/resources/models", `{}`, login.Token, 403)
	if calls != beforeInvalid || redirectCalls != 0 {
		t.Fatal("invalid or unauthorized requests contacted an upstream")
	}
	for _, failure := range []string{"unauthorized", "redirect", "invalid", "large"} {
		mode = failure
		if err := s.DB.Model(&model.AIProviderKey{}).Where("resource_id = ?", resource.ID).Updates(map[string]any{"state": service.ProviderKeyAvailable, "retry_at": nil}).Error; err != nil {
			t.Fatal(err)
		}
		if body := invoke(in, 502); strings.Contains(body, savedSecret) {
			t.Fatal("upstream error exposed external credentials")
		}
	}
	if redirectCalls != 0 {
		t.Fatal("upstream redirects were followed")
	}
	mode = "empty"
	if body := invoke(in, 200); !strings.Contains(body, `"modelIds":[]`) {
		t.Fatalf("empty upstream list not handled: %s", body)
	}
}
