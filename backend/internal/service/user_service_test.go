package service

import (
	"minioa/internal/repository"
	"testing"
)

func TestSystemAdminProtection(t *testing.T) {
	s := newService(t)
	first := bootstrap(t, s)
	id := first.User.ID
	role, err := s.SaveRole(0, RoleInput{Name: "普通角色", Permissions: []string{"users:read"}})
	if err != nil {
		t.Fatal(err)
	}
	wantStatus(t, s.SetStatus(id, false), 409)
	for _, ids := range [][]uint{{}, {role.ID}, {first.User.Roles[0].ID}, {first.User.Roles[0].ID, role.ID}} {
		_, err = s.UpdateUser(id, UserInput{DisplayName: "不应保存", RoleIDs: ids})
		wantStatus(t, err, 409)
	}
	_, err = s.CreateUser(UserInput{Username: "admin2", Password: "password123", RoleIDs: []uint{first.User.Roles[0].ID}})
	wantStatus(t, err, 409)
	updated, err := s.UpdateUser(id, UserInput{DisplayName: "新管理员姓名"})
	if err != nil {
		t.Fatal(err)
	}
	if !isAdmin(updated) || !updated.Enabled || len(updated.Roles) != 1 || updated.DisplayName != "新管理员姓名" {
		t.Fatalf("admin profile update changed protected fields: %+v", updated)
	}
	second, err := s.CreateUser(UserInput{Username: "member", Password: "password123", RoleIDs: []uint{role.ID}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.UpdateUser(second.ID, UserInput{RoleIDs: []uint{role.ID, first.User.Roles[0].ID}})
	wantStatus(t, err, 409)
	saved, err := repository.User(s.DB, second.ID)
	if err != nil || isAdmin(saved) || len(saved.Roles) != 1 || saved.Roles[0].ID != role.ID {
		t.Fatalf("rejected promotion changed roles: %+v %v", saved, err)
	}
	_, total, err := repository.Users(s.DB, "admin2", 1, 1)
	if err != nil || total != 0 {
		t.Fatalf("rejected admin creation persisted a user: %d %v", total, err)
	}
}

func TestUserStatusAndInvalidation(t *testing.T) {
	s := newService(t)
	bootstrap(t, s)
	second, err := s.CreateUser(UserInput{Username: "member", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	login, err := s.Login(Credentials{Username: "member", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.SetStatus(second.ID, false); err != nil {
		t.Fatal(err)
	}
	_, err = s.Authenticate(login.Token)
	wantStatus(t, err, 401)
	_, err = s.Login(Credentials{Username: "member", Password: "password123"})
	wantStatus(t, err, 401)
	if err = s.SetStatus(second.ID, true); err != nil {
		t.Fatal(err)
	}
	_, err = s.Authenticate(login.Token)
	wantStatus(t, err, 401)
	users, total, err := repository.Users(s.DB, "member", 1, 1)
	if err != nil || total != 1 || len(users) != 1 {
		t.Fatalf("list %v %d %v", users, total, err)
	}
}
func TestUserValidation(t *testing.T) {
	s := newService(t)
	bootstrap(t, s)
	for _, in := range []UserInput{{Username: "x", Password: "password123"}, {Username: "valid", Password: "short"}, {Username: "valid", Password: "password123", RoleIDs: []uint{999}}} {
		if _, err := s.CreateUser(in); err == nil {
			t.Fatal("invalid user accepted")
		}
	}
	_, err := s.CreateUser(UserInput{Username: "admin", Password: "password123"})
	wantStatus(t, err, 409)
}

func TestUsernameIsImmutable(t *testing.T) {
	s := newService(t)
	bootstrap(t, s)
	u, err := s.CreateUser(UserInput{Username: "member", DisplayName: "原姓名", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.UpdateUser(u.ID, UserInput{Username: "renamed", DisplayName: "不应保存"})
	wantStatus(t, err, 400)
	saved, err := repository.User(s.DB, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Username != "member" || saved.DisplayName != "原姓名" {
		t.Fatalf("rejected update changed user: %+v", saved)
	}
	updated, err := s.UpdateUser(u.ID, UserInput{DisplayName: "新姓名"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Username != "member" || updated.DisplayName != "新姓名" {
		t.Fatalf("profile update did not preserve username: %+v", updated)
	}
}
