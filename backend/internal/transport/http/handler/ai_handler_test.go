package handler_test

import (
	"encoding/json"
	"minioa/internal/model"
	"minioa/internal/service"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestAIResourcesKeysAndProxy(t *testing.T) {
	r, s := server(t)
	s.Config.DataDir = t.TempDir()
	a := admin(t, r)
	const external = "external-secret-test"
	const externalModel = "provider-chat-v2"
	upstreamCalls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		upstreamCalls++
		var payload map[string]json.RawMessage
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil || string(payload["model"]) != `"`+externalModel+`"` {
			t.Errorf("internal model was not mapped: %v %s", err, payload["model"])
		}
		if value, ok := payload["custom_integer"]; ok && string(value) != "9007199254740993" {
			t.Errorf("request fields changed: %s", value)
		}
		if string(payload["stream"]) == "true" && !strings.Contains(string(payload["messages"]), "你好") {
			t.Error("streaming request messages were not preserved")
		}
		if req.Header.Get("Authorization") != "Bearer "+external {
			t.Error("external credential not substituted")
		}
		if req.Header.Get("Cookie") != "" || req.Header.Get("X-Api-Key") != "" {
			t.Error("client credentials forwarded")
		}
		if req.URL.Path != "/v1/chat/completions" && req.URL.Path != "/v1/models" && req.URL.Path != "/v1/embeddings" && req.URL.Path != "/v1/responses" {
			t.Errorf("wrong upstream path: %s", req.URL.Path)
		}
		if req.URL.Query().Get("fail") == "true" {
			w.WriteHeader(503)
			w.Write([]byte(external))
			return
		}
		if req.URL.Path != "/v1/chat/completions" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"data":[],"ok":true}`))
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: {\"ok\":true}\n\n"))
		w.(http.Flusher).Flush()
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer upstream.Close()
	payload, _ := json.Marshal(service.AIResourceInput{Name: "test", BaseURL: upstream.URL + "/v1", Enabled: true})
	w := call(t, r, "POST", "/api/ai/resources", string(payload), a.Token, 200)
	if strings.Contains(w.Body.String(), external) {
		t.Fatal("external secret returned")
	}
	var resource model.AIResource
	if err := json.Unmarshal(w.Body.Bytes(), &resource); err != nil {
		t.Fatal(err)
	}
	keyPayload, _ := json.Marshal(service.AIProviderKeyInput{ResourceID: resource.ID, Name: "primary", APIKey: external, Enabled: true})
	call(t, r, "POST", "/api/ai/provider-keys", string(keyPayload), a.Token, 200)
	if _, err := s.SaveAIModel(0, service.AIModelInput{InternalID: "chat-default", ExternalID: externalModel, ResourceID: resource.ID, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	var stored model.AIResource
	if err := s.DB.Preload("Keys").First(&stored, resource.ID).Error; err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(stored.Keys[0].Secret), external) {
		t.Fatal("external key stored in plaintext")
	}
	u, err := s.CreateUser(service.UserInput{Username: "member", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	login, err := s.Login(service.Credentials{Username: "member", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	call(t, r, "GET", "/api/ai/resources", "", login.Token, 403)
	call(t, r, "POST", "/api/ai/keys", `{}`, login.Token, 403)
	call(t, r, "GET", "/api/ai/resources/options", "", login.Token, 403)
	aiRole, err := s.SaveRole(0, service.RoleInput{Name: "AI用户", Permissions: []string{"ai:use"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpdateUser(u.ID, service.UserInput{RoleIDs: []uint{aiRole.ID}}); err != nil {
		t.Fatal(err)
	}
	call(t, r, "GET", "/api/ai/resources/options", "", login.Token, 403)
	call(t, r, "POST", "/api/ai/keys", `{"name":"forged","userId":1,"resourceId":1}`, login.Token, 400)
	body, _ := json.Marshal(service.AIKeyInput{Name: "member key"})
	w = call(t, r, "POST", "/api/ai/keys", string(body), login.Token, 201)
	var issued struct {
		Key    model.AIKey `json:"key"`
		Secret string      `json:"secret"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &issued); err != nil {
		t.Fatal(err)
	}
	if issued.Secret == "" || issued.Key.UserID != u.ID {
		t.Fatal("missing one-time secret")
	}
	var persisted model.AIKey
	if err := s.DB.First(&persisted, issued.Key.ID).Error; err != nil {
		t.Fatal(err)
	}
	if persisted.TokenHash == issued.Secret || len(persisted.TokenHash) != 64 {
		t.Fatal("internal key is not stored as a digest")
	}
	w = call(t, r, "GET", "/api/ai/keys", "", login.Token, 200)
	if !strings.Contains(w.Body.String(), "member key") || strings.Contains(w.Body.String(), issued.Secret) || strings.Contains(w.Body.String(), external) || strings.Contains(w.Body.String(), `"prefix"`) {
		t.Fatal("key list isolation or secrecy broken")
	}
	call(t, r, "GET", "/api/ai/keys?all=true", "", login.Token, 403)
	w = call(t, r, "GET", "/api/ai/keys", "", a.Token, 200)
	if strings.Contains(w.Body.String(), "member key") {
		t.Fatal("self list contains another user's key")
	}
	// The proxy authenticates internal keys independently of login tokens.
	call(t, r, "POST", "/ai/v1/chat/completions", `{}`, login.Token, 401)
	w = call(t, r, "POST", "/ai/v1/chat/completions", `{"model":"chat-default","messages":[{"role":"user","content":"你好"}],"stream":true}`, issued.Secret, 200)
	if !w.Flushed || !strings.Contains(w.Body.String(), "[DONE]") {
		t.Fatal("stream not relayed")
	}
	beforeModels := upstreamCalls
	w = call(t, r, "GET", "/ai/v1/models", "", issued.Secret, 200)
	if upstreamCalls != beforeModels || !strings.Contains(w.Body.String(), `"id":"chat-default"`) || strings.Contains(w.Body.String(), externalModel) {
		t.Fatal("model list must be local and expose internal IDs only")
	}
	for _, body := range []string{`{}`, `{"model":null}`, `{"model":123}`, `{"model":"unknown"}`, `{"model":"` + externalModel + `"}`, `[]`, `null`, `{"model":"chat-default"} {}`} {
		call(t, r, "POST", "/ai/v1/chat/completions", body, issued.Secret, 400)
	}
	call(t, r, "POST", "/ai/v1/chat/completions", `{"model":"chat-default","input":"`+strings.Repeat("x", 8<<20)+`"}`, issued.Secret, 413)
	if upstreamCalls != beforeModels {
		t.Fatal("invalid requests reached upstream")
	}
	otherUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil || body.Model != "provider-other" || req.Header.Get("Authorization") != "Bearer other-key" {
			t.Error("second supplier did not receive its model and credential")
		}
		w.Write([]byte(`{"supplier":"other"}`))
	}))
	defer otherUpstream.Close()
	otherResource, err := saveProviderTestResource(s, service.AIResourceInput{Name: "other resource", BaseURL: otherUpstream.URL + "/v1", Enabled: true}, []service.AIProviderKeyInput{{Name: "primary", APIKey: "other-key", Enabled: true}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveAIModel(0, service.AIModelInput{InternalID: "other-model", ExternalID: "provider-other", ResourceID: otherResource.ID, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	otherResponse := call(t, r, "POST", "/ai/v1/chat/completions", `{"model":"other-model"}`, issued.Secret, 200)
	if !strings.Contains(otherResponse.Body.String(), `"supplier":"other"`) {
		t.Fatal("same internal key could not call another supplier")
	}
	allModels := call(t, r, "GET", "/ai/v1/models", "", issued.Secret, 200)
	if !strings.Contains(allModels.Body.String(), "other-model") || !strings.Contains(allModels.Body.String(), "chat-default") || strings.Contains(allModels.Body.String(), "provider-other") {
		t.Fatal("global model list incorrect")
	}
	call(t, r, "POST", "/ai/v1/embeddings", `{"model":"chat-default","input":"test"}`, issued.Secret, 200)
	call(t, r, "POST", "/ai/v1/responses", `{"model":"chat-default","input":"test","custom_integer":9007199254740993}`, issued.Secret, 200)
	request := httptest.NewRequest("POST", "/ai/v1/chat/completions", strings.NewReader(`{"model":"chat-default"}`))
	request.Header.Set("Authorization", "Bearer "+issued.Secret)
	request.Header.Set("Cookie", "private=cookie")
	request.Header.Set("X-Api-Key", "private-client-key")
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, request)
	if recorder.Code != 200 {
		t.Fatal("header sanitization request failed")
	}
	w = call(t, r, "POST", "/ai/v1/chat/completions?fail=true", `{"model":"chat-default"}`, issued.Secret, 503)
	if strings.Contains(w.Body.String(), external) {
		t.Fatal("upstream error leaked external key")
	}
	// Reloading the service uses the same on-disk encryption key.
	restarted := &service.Service{DB: s.DB, Config: s.Config}
	lease, err := restarted.AcquireProviderKey(resource.ID, nil)
	if err != nil || lease.Secret != external {
		t.Fatalf("restart lost encrypted key: %v", err)
	}
	if err := lease.Finish("", nil, false); err != nil {
		t.Fatal(err)
	}
	other, err := s.CreateUser(service.UserInput{Username: "other", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	otherLogin, err := s.Login(service.Credentials{Username: other.Username, Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	route := "/api/ai/keys/" + strconv.FormatUint(uint64(issued.Key.ID), 10)
	call(t, r, "POST", route+"/rotate", ``, a.Token, 404)
	call(t, r, "POST", route+"/rotate", ``, otherLogin.Token, 404)
	call(t, r, "DELETE", route, ``, otherLogin.Token, 404)
	w = call(t, r, "POST", route+"/rotate", ``, login.Token, 200)
	var rotated struct {
		Secret string `json:"secret"`
	}
	json.Unmarshal(w.Body.Bytes(), &rotated)
	call(t, r, "POST", "/ai/v1/chat/completions", `{}`, issued.Secret, 401)
	call(t, r, "POST", "/ai/v1/chat/completions", `{"model":"chat-default"}`, rotated.Secret, 200)
	if _, err := s.SaveRole(aiRole.ID, service.RoleInput{Name: "AI用户", Permissions: []string{}}); err != nil {
		t.Fatal(err)
	}
	call(t, r, "POST", "/ai/v1/chat/completions", `{}`, rotated.Secret, 403)
	call(t, r, "POST", route+"/rotate", ``, login.Token, 403)
	call(t, r, "POST", "/api/ai/keys", string(body), login.Token, 403)
	if _, err := s.SaveRole(aiRole.ID, service.RoleInput{Name: "AI用户", Permissions: []string{"ai:use"}}); err != nil {
		t.Fatal(err)
	}
	call(t, r, "POST", "/ai/v1/chat/completions", `{"model":"chat-default"}`, rotated.Secret, 200)
	if _, err := s.UpdateUser(u.ID, service.UserInput{RoleIDs: []uint{}}); err != nil {
		t.Fatal(err)
	}
	call(t, r, "POST", "/ai/v1/chat/completions", `{}`, rotated.Secret, 403)
	if _, err := s.UpdateUser(u.ID, service.UserInput{RoleIDs: []uint{aiRole.ID}}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetStatus(u.ID, false); err != nil {
		t.Fatal(err)
	}
	call(t, r, "POST", "/ai/v1/chat/completions", `{}`, rotated.Secret, 401)
	if err := s.SetStatus(u.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveAIResource(resource.ID, service.AIResourceInput{Name: "test", BaseURL: upstream.URL + "/v1", Enabled: false}); err != nil {
		t.Fatal(err)
	}
	call(t, r, "POST", "/ai/v1/chat/completions", `{"model":"chat-default"}`, rotated.Secret, 400)
	call(t, r, "POST", "/ai/v1/chat/completions", `{"model":"other-model"}`, rotated.Secret, 200)
	visible := call(t, r, "GET", "/ai/v1/models", "", rotated.Secret, 200)
	if strings.Contains(visible.Body.String(), "chat-default") || !strings.Contains(visible.Body.String(), "other-model") {
		t.Fatal("disabled supplier affected other models")
	}
	if _, err := s.SaveAIResource(resource.ID, service.AIResourceInput{Name: "test", BaseURL: upstream.URL + "/v1", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	call(t, r, "DELETE", route, ``, a.Token, 204)
	call(t, r, "POST", "/ai/v1/chat/completions", `{}`, rotated.Secret, 401)
}
