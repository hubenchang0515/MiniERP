package service

import (
	"minierp/internal/repository"
	"testing"
)

func TestRoleRules(t *testing.T) {
	s := newService(t)
	session := bootstrap(t, s)
	systemID := session.User.Roles[0].ID
	wantStatus(t, s.DeleteRole(systemID), 409)
	_, err := s.SaveRole(systemID, RoleInput{Name: "rename"})
	wantStatus(t, err, 409)
	_, err = s.SaveRole(0, RoleInput{Name: "bad", Permissions: []string{"unknown"}})
	wantStatus(t, err, 400)
	r, err := s.SaveRole(0, RoleInput{Name: "人事", Permissions: []string{"users:manage", "users:manage"}})
	if err != nil || len(r.Permissions) != 1 {
		t.Fatalf("%+v %v", r, err)
	}
	u, err := s.CreateUser(UserInput{Username: "member", Password: "password123", RoleIDs: []uint{r.ID}})
	if err != nil {
		t.Fatal(err)
	}
	wantStatus(t, s.DeleteRole(r.ID), 409)
	if _, err = s.SaveRole(r.ID, RoleInput{Name: "人事", Permissions: []string{"roles:read"}}); err != nil {
		t.Fatal(err)
	}
	u, err = repository.User(s.DB, u.ID)
	if err != nil || !Can(u, "roles:read") || Can(u, "users:manage") {
		t.Fatal("permissions did not update")
	}
	if _, err = s.UpdateUser(u.ID, UserInput{Username: u.Username, RoleIDs: []uint{}}); err != nil {
		t.Fatal(err)
	}
	if err = s.DeleteRole(r.ID); err != nil {
		t.Fatal(err)
	}
}
