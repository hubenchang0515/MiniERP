package handler_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"minioa/internal/service"
)

func TestSystemResourcesAccess(t *testing.T) {
	r, s := server(t)
	a := admin(t, r)
	call(t, r, "GET", "/api/system/resources", "", "", 401)
	role, err := s.SaveRole(0, service.RoleInput{Name: "全部业务权限", Permissions: []string{"users:manage", "roles:manage"}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.CreateUser(service.UserInput{Username: "manager", Password: "password123", RoleIDs: []uint{role.ID}})
	if err != nil {
		t.Fatal(err)
	}
	login, err := s.Login(service.Credentials{Username: "manager", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	call(t, r, "GET", "/api/system/resources", "", login.Token, 403)
	call(t, r, "GET", "/api/system/resources", "", a.Token, 503)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	s.StartResourceMonitor(ctx)
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err = s.SystemResources(); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("resource cache did not become ready")
		}
		time.Sleep(10 * time.Millisecond)
	}
	w := call(t, r, "GET", "/api/system/resources", "", a.Token, 200)
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("system response must not be cached")
	}
	var result service.SystemResources
	if err = json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.SampledAt.IsZero() || result.Disk.Path == "" {
		t.Fatalf("missing resource metadata: %+v", result)
	}
	if result.CPU.Error == "" && (result.CPU.Percent < 0 || result.CPU.Percent > 100) {
		t.Fatalf("invalid CPU usage: %+v", result.CPU)
	}
	if result.Memory.Error == "" && (result.Memory.Total == 0 || result.Memory.Used > result.Memory.Total) {
		t.Fatalf("invalid memory usage: %+v", result.Memory)
	}
	if result.Disk.Error == "" && (result.Disk.Total == 0 || result.Disk.Used > result.Disk.Total) {
		t.Fatalf("invalid disk usage: %+v", result.Disk)
	}
}
