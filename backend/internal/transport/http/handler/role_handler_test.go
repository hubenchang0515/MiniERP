package handler_test

import (
	"encoding/json"
	"minioa/internal/service"
	"testing"
)

func TestRoleHTTP(t *testing.T) {
	r, _ := server(t)
	a := admin(t, r)
	call(t, r, "GET", "/api/permissions", "", a.Token, 200)
	call(t, r, "POST", "/api/roles", `{"name":"阅读者","permissions":["users:read"]}`, a.Token, 201)
	call(t, r, "PATCH", "/api/roles/2", `{"name":"阅读者","description":"查看成员","permissions":[]}`, a.Token, 200)
	call(t, r, "DELETE", "/api/roles/2", "", a.Token, 204)
	call(t, r, "DELETE", "/api/roles/1", "", a.Token, 409)
	call(t, r, "GET", "/api/not-real", "", "", 404)
	call(t, r, "GET", "/assets/missing.js", "", "", 404)
	call(t, r, "GET", "/placeholder.html", "", "", 404)
	call(t, r, "POST", "/api/roles", `{"name":"a"} {"name":"b"}`, a.Token, 400)
}

func TestRoleListsAndOptions(t *testing.T) {
	r, s := server(t)
	a := admin(t, r)
	for _, name := range []string{"reader-a", "reader-b"} {
		if _, err := s.SaveRole(0, service.RoleInput{Name: name, Description: "searchable", Permissions: []string{"users:read"}}); err != nil {
			t.Fatal(err)
		}
	}
	for _, base := range []string{"/api/roles", "/api/roles/options"} {
		call(t, r, "GET", base, "", "", 401)
		for _, suffix := range []string{"?page=0", "?page=x", "?pageSize=0", "?pageSize=101"} {
			call(t, r, "GET", base+suffix, "", a.Token, 400)
		}
	}
	w := call(t, r, "GET", "/api/roles?q=reader&page=2&pageSize=1", "", a.Token, 200)
	var result struct {
		Items []map[string]any `json:"items"`
		Total int              `json:"total"`
		Page  int              `json:"page"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Total != 2 || result.Page != 2 || len(result.Items) != 1 || result.Items[0]["name"] != "reader-b" || result.Items[0]["permissions"] == nil {
		t.Fatalf("bad role page: %s", w.Body.String())
	}
	w = call(t, r, "GET", "/api/roles/options?pageSize=1", "", a.Token, 200)
	result.Items = nil
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Total != 2 || len(result.Items) != 1 || len(result.Items[0]) != 3 || result.Items[0]["name"] != "reader-a" {
		t.Fatalf("options must be small and exclude system role: %s", w.Body.String())
	}
	u, err := s.CreateUser(service.UserInput{Username: "member", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	login, err := s.Login(service.Credentials{Username: "member", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	call(t, r, "GET", "/api/roles/options", "", login.Token, 403)
	manager, err := s.SaveRole(0, service.RoleInput{Name: "manager", Permissions: []string{"users:manage"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpdateUser(u.ID, service.UserInput{RoleIDs: []uint{manager.ID}}); err != nil {
		t.Fatal(err)
	}
	call(t, r, "GET", "/api/roles/options?q=reader", "", login.Token, 200)
}
