package handler_test

import (
	"minierp/internal/service"
	"testing"
)

func TestUserHTTPPermissions(t *testing.T) {
	r, s := server(t)
	a := admin(t, r)
	call(t, r, "GET", "/api/users", "", "", 401)
	call(t, r, "GET", "/api/users?page=0", "", a.Token, 400)
	call(t, r, "GET", "/api/users?pageSize=101", "", a.Token, 400)
	call(t, r, "GET", "/api/users/bad", "", a.Token, 400)
	call(t, r, "GET", "/api/users/999", "", a.Token, 404)
	call(t, r, "PATCH", "/api/users/1/status", `{}`, a.Token, 400)
	call(t, r, "PATCH", "/api/users/1/status", `{"enabled":false}`, a.Token, 409)
	u, err := s.CreateUser(service.UserInput{Username: "member", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	_ = u
	login, err := s.Login(service.Credentials{Username: "member", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	call(t, r, "GET", "/api/users", "", login.Token, 403)
	call(t, r, "POST", "/api/users", `{"username":"other","password":"password123","roleIds":[]}`, a.Token, 201)
	role, err := s.SaveRole(0, service.RoleInput{Name: "用户管理", Permissions: []string{"users:manage"}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.UpdateUser(u.ID, service.UserInput{Username: u.Username, RoleIDs: []uint{role.ID}})
	if err != nil {
		t.Fatal(err)
	}
	call(t, r, "GET", "/api/users", "", login.Token, 200)
	call(t, r, "GET", "/api/roles", "", login.Token, 200)
	call(t, r, "GET", "/api/permissions", "", login.Token, 403)
	call(t, r, "POST", "/api/users/2/password-reset", `{"password":"changed123"}`, a.Token, 204)
	call(t, r, "GET", "/api/auth/me", "", login.Token, 401)
}
