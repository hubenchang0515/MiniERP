package handler_test

import (
	"encoding/json"
	"minierp/internal/model"
	"minierp/internal/service"
	"strconv"
	"strings"
	"testing"
)

func saveProviderTestResource(s *service.Service, input service.AIResourceInput, keys []service.AIProviderKeyInput) (model.AIResource, error) {
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

func TestAIProviderKeyManagement(t *testing.T) {
	r, s := server(t)
	s.Config.DataDir = t.TempDir()
	a := admin(t, r)
	resource, err := s.SaveAIResource(0, service.AIResourceInput{Name: "supplier", BaseURL: "https://example.com/v1", Enabled: true})
	if err != nil || len(resource.Keys) != 0 {
		t.Fatalf("supplier must be creatable before keys: %v", err)
	}
	input := service.AIProviderKeyInput{ResourceID: resource.ID, Name: "primary", APIKey: "external-credential", Enabled: true}
	invoke := func(method, path string, in service.AIProviderKeyInput, status int) string {
		t.Helper()
		body, _ := json.Marshal(in)
		return call(t, r, method, path, string(body), a.Token, status).Body.String()
	}
	body := invoke("POST", "/api/ai/provider-keys", input, 200)
	if strings.Contains(body, "external-credential") || strings.Contains(body, "secret") {
		t.Fatal("provider key creation leaked secret")
	}
	var key model.AIProviderKey
	if err := json.Unmarshal([]byte(body), &key); err != nil || key.ID == 0 {
		t.Fatal("missing created key")
	}
	var stored model.AIProviderKey
	if err := s.DB.First(&stored, key.ID).Error; err != nil || string(stored.Secret) == "external-credential" || len(stored.Secret) == 0 {
		t.Fatal("key was not encrypted")
	}
	list := call(t, r, "GET", "/api/ai/provider-keys?q=supplier&pageSize=1", "", a.Token, 200)
	var page struct {
		Items []struct {
			ResourceID   uint   `json:"resourceId"`
			ResourceName string `json:"resourceName"`
		}
		Total int64
	}
	if err := json.Unmarshal(list.Body.Bytes(), &page); err != nil || page.Total != 1 || len(page.Items) != 1 || page.Items[0].ResourceID != resource.ID || page.Items[0].ResourceName != "supplier" {
		t.Fatalf("provider key list invalid: %s %v", list.Body.String(), err)
	}
	if strings.Contains(list.Body.String(), "external-credential") || strings.Contains(list.Body.String(), "secret") {
		t.Fatal("key list leaked secret")
	}
	path := "/api/ai/provider-keys/" + strconv.Itoa(int(key.ID))
	input.APIKey = ""
	input.Name = "renamed"
	input.Enabled = false
	invoke("PATCH", path, input, 200)
	var updated model.AIProviderKey
	if err := s.DB.First(&updated, key.ID).Error; err != nil || updated.Enabled || string(updated.Secret) != string(stored.Secret) {
		t.Fatal("blank secret update or manual disable failed")
	}
	input.ResourceID = resource.ID + 100
	invoke("PATCH", path, input, 404)
	call(t, r, "POST", "/api/ai/resources", `{"name":"bad","baseURL":"https://example.com/v1","enabled":true,"keys":[]}`, a.Token, 400)
	u, err := s.CreateUser(service.UserInput{Username: "reader", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	login, err := s.Login(service.Credentials{Username: u.Username, Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	for _, method := range []string{"GET", "POST", "PATCH", "DELETE"} {
		target := "/api/ai/provider-keys"
		if method == "PATCH" || method == "DELETE" {
			target = path
		}
		call(t, r, method, target, `{}`, login.Token, 403)
	}
	call(t, r, "GET", "/api/ai/provider-keys", "", "", 401)
	call(t, r, "DELETE", path, "", a.Token, 204)
	call(t, r, "DELETE", path, "", a.Token, 404)
	list = call(t, r, "GET", "/api/ai/provider-keys", "", a.Token, 200)
	if !strings.Contains(list.Body.String(), `"total":0`) {
		t.Fatal("last provider key could not be deleted independently")
	}
}
