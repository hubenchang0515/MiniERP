package handler_test

import (
	"encoding/json"
	"minierp/internal/model"
	"minierp/internal/service"
	"strconv"
	"strings"
	"testing"
)

func TestAIGlobalMappingManagement(t *testing.T) {
	r, s := server(t)
	s.Config.DataDir = t.TempDir()
	a := admin(t, r)
	resource, err := saveProviderTestResource(s, service.AIResourceInput{Name: "first", BaseURL: "https://example.com/v1", Enabled: true}, []service.AIProviderKeyInput{{Name: "primary", APIKey: "external-secret", Enabled: true}})
	if err != nil {
		t.Fatal(err)
	}
	second, err := saveProviderTestResource(s, service.AIResourceInput{Name: "second", BaseURL: "https://second.example/v1", Enabled: true}, []service.AIProviderKeyInput{{Name: "primary", APIKey: "second-secret", Enabled: true}})
	if err != nil {
		t.Fatal(err)
	}
	input := service.AIModelInput{InternalID: "chat", ExternalID: "external-v1", ResourceID: resource.ID, Enabled: true}
	invoke := func(path, method string, in service.AIModelInput, status int) string {
		t.Helper()
		body, _ := json.Marshal(in)
		return call(t, r, method, path, string(body), a.Token, status).Body.String()
	}
	body := invoke("/api/ai/models", "POST", input, 200)
	var mapping model.AIModel
	if err := json.Unmarshal([]byte(body), &mapping); err != nil {
		t.Fatal(err)
	}
	input.ResourceID = second.ID
	invoke("/api/ai/models", "POST", input, 409)
	path := "/api/ai/models/" + strconv.Itoa(int(mapping.ID))
	input.ExternalID = "external-v2"
	invoke(path, "PATCH", input, 200)
	resolved, err := s.ResolveAIModel("chat")
	if err != nil || resolved.ResourceID != second.ID || resolved.ExternalID != "external-v2" {
		t.Fatal("mapping could not switch supplier")
	}
	for _, text := range []string{"", "invalid space", strings.Repeat("x", 257)} {
		invalid := input
		invalid.InternalID = text
		invoke(path, "PATCH", invalid, 400)
	}
	invalid := input
	invalid.ResourceID = second.ID + 100
	invoke(path, "PATCH", invalid, 400)
	list := call(t, r, "GET", "/api/ai/models?q=second&pageSize=1", "", a.Token, 200).Body.String()
	if !strings.Contains(list, `"total":1`) || !strings.Contains(list, `"resourceName":"second"`) || strings.Contains(list, "second-secret") {
		t.Fatalf("mapping list invalid: %s", list)
	}
	input.Enabled = false
	invoke(path, "PATCH", input, 200)
	if _, err := s.ResolveAIModel("chat"); err == nil {
		t.Fatal("disabled mapping resolved")
	}
	call(t, r, "DELETE", path, "", a.Token, 204)
	call(t, r, "DELETE", path, "", a.Token, 404)
	if _, err := s.ResolveAIModel("chat"); err == nil {
		t.Fatal("deleted mapping resolved")
	}
	user, err := s.CreateUser(service.UserInput{Username: "ai-user", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	role, err := s.SaveRole(0, service.RoleInput{Name: "AI", Permissions: []string{"ai:use"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpdateUser(user.ID, service.UserInput{RoleIDs: []uint{role.ID}}); err != nil {
		t.Fatal(err)
	}
	login, err := s.Login(service.Credentials{Username: user.Username, Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	for _, method := range []string{"GET", "POST", "PATCH", "DELETE"} {
		target := "/api/ai/models"
		if method == "PATCH" || method == "DELETE" {
			target = path
		}
		call(t, r, method, target, `{}`, login.Token, 403)
	}
	// A new key needs neither a supplier nor an existing mapping.
	call(t, r, "POST", "/api/ai/keys", `{"name":"global-key"}`, login.Token, 201)
	call(t, r, "POST", "/api/ai/keys", `{"name":"bound-key","resourceId":1}`, login.Token, 400)
}
