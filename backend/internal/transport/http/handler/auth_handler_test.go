package handler_test

import (
	"bytes"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"minierp/internal/config"
	"minierp/internal/service"
	"minierp/internal/testutil"
	transport "minierp/internal/transport/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func server(t *testing.T) (*gin.Engine, *service.Service) {
	gin.SetMode(gin.TestMode)
	s := &service.Service{DB: testutil.DB(t), Config: config.Config{JWTSecret: "test", JWTTTL: time.Hour}}
	return transport.Router(s), s
}
func call(t *testing.T, r *gin.Engine, method, path, body, token string, status int) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != status {
		t.Fatalf("%s %s: want %d, got %d %s", method, path, status, w.Code, w.Body.String())
	}
	return w
}
func admin(t *testing.T, r *gin.Engine) service.Session {
	t.Helper()
	w := call(t, r, "POST", "/api/auth/bootstrap", `{"username":"admin","password":"password123"}`, "", 201)
	var v service.Session
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
		t.Fatal(err)
	}
	return v
}
func TestAuthHTTP(t *testing.T) {
	r, _ := server(t)
	w := call(t, r, "GET", "/api/auth/status", "", "", 200)
	if !bytes.Contains(w.Body.Bytes(), []byte(`false`)) {
		t.Fatal(w.Body)
	}
	call(t, r, "POST", "/api/auth/bootstrap", `{`, "", 400)
	a := admin(t, r)
	call(t, r, "POST", "/api/auth/bootstrap", `{"username":"admin","password":"password123"}`, "", 409)
	call(t, r, "POST", "/api/auth/login", `{"username":"admin","password":"wrong"}`, "", 401)
	call(t, r, "POST", "/api/auth/login", `{"username":"admin","password":"password123"}`, "", 200)
	w = call(t, r, "GET", "/api/auth/me", "", a.Token, 200)
	if bytes.Contains(w.Body.Bytes(), []byte("password")) {
		t.Fatal("password leaked")
	}
	call(t, r, "GET", "/api/auth/me", "", "bad", 401)
}

func TestChangeOwnPasswordWithoutPermissions(t *testing.T) {
	r, s := server(t)
	a := admin(t, r)
	_, err := s.CreateUser(service.UserInput{Username: "member", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	first, err := s.Login(service.Credentials{Username: "member", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.Login(service.Credentials{Username: "member", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	call(t, r, "POST", "/api/auth/password", `{}`, "", 401)
	call(t, r, "GET", "/api/users", "", first.Token, 403)
	call(t, r, "POST", "/api/auth/password", `{"currentPassword":"wrong","newPassword":"changed123"}`, first.Token, 400)
	call(t, r, "POST", "/api/auth/password", `{"currentPassword":"password123","newPassword":"short"}`, first.Token, 400)
	call(t, r, "GET", "/api/auth/me", "", first.Token, 200)
	// An injected user ID cannot target another account.
	call(t, r, "POST", "/api/auth/password", `{"userId":1,"currentPassword":"password123","newPassword":"changed123"}`, first.Token, 400)
	call(t, r, "POST", "/api/auth/password", `{"currentPassword":"password123","newPassword":"changed123"}`, first.Token, 204)
	call(t, r, "GET", "/api/auth/me", "", first.Token, 401)
	call(t, r, "GET", "/api/auth/me", "", second.Token, 401)
	call(t, r, "POST", "/api/auth/login", `{"username":"member","password":"password123"}`, "", 401)
	call(t, r, "POST", "/api/auth/login", `{"username":"member","password":"changed123"}`, "", 200)
	call(t, r, "GET", "/api/auth/me", "", a.Token, 200)
	call(t, r, "POST", "/api/auth/login", `{"username":"admin","password":"password123"}`, "", 200)
}
