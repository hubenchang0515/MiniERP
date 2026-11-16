package service

import (
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"minierp/internal/config"
	"minierp/internal/model"
	"minierp/internal/testutil"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func newService(t *testing.T) *Service {
	return &Service{DB: testutil.DB(t), Config: config.Config{JWTSecret: "test-secret", JWTTTL: time.Hour}}
}
func bootstrap(t *testing.T, s *Service) Session {
	t.Helper()
	v, err := s.Bootstrap(Credentials{Username: "admin", Password: "password123", DisplayName: "管理员"})
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func wantStatus(t *testing.T, err error, status int) {
	t.Helper()
	var e *Error
	if !errors.As(err, &e) || e.Status != status {
		t.Fatalf("want %d, got %v", status, err)
	}
}
func TestBootstrapConcurrent(t *testing.T) {
	s := newService(t)
	var ok atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.Bootstrap(Credentials{Username: "admin", Password: "password123"})
			if err == nil {
				ok.Add(1)
			} else {
				var e *Error
				if !errors.As(err, &e) || e.Status != 409 {
					t.Errorf("unexpected: %v", err)
				}
			}
		}()
	}
	wg.Wait()
	if ok.Load() != 1 {
		t.Fatalf("bootstrap winners = %d", ok.Load())
	}
	var n int64
	s.DB.Model(&model.User{}).Count(&n)
	if n != 1 {
		t.Fatal(n)
	}
}
func TestLoginAndTokens(t *testing.T) {
	s := newService(t)
	session := bootstrap(t, s)
	if len(session.User.Permissions) != 4 {
		t.Fatal(session)
	}
	var stored model.User
	s.DB.First(&stored, session.User.ID)
	if stored.PasswordHash == "password123" || stored.PasswordHash == "" {
		t.Fatal("plaintext password")
	}
	for _, in := range []Credentials{{Username: "missing", Password: "password123"}, {Username: "admin", Password: "wrong"}} {
		_, err := s.Login(in)
		wantStatus(t, err, 401)
	}
	if _, err := s.Authenticate(session.Token); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Authenticate(session.Token + "tampered"); err == nil {
		t.Fatal("forged token accepted")
	}
	expired := claims{stored.TokenVersion, jwt.RegisteredClaims{Issuer: "minierp", Subject: "1", ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour))}}
	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, expired).SignedString([]byte(s.Config.JWTSecret))
	_, err := s.Authenticate(token)
	wantStatus(t, err, 401)
	if err := s.ResetPassword(stored.ID, "newpassword123"); err != nil {
		t.Fatal(err)
	}
	_, err = s.Authenticate(session.Token)
	wantStatus(t, err, 401)
	if _, err = s.Login(Credentials{Username: "admin", Password: "newpassword123"}); err != nil {
		t.Fatal(err)
	}
}
func TestPermissionUnion(t *testing.T) {
	u := model.User{Roles: []model.Role{{Permissions: []model.Permission{{Code: "users:manage"}}}, {Permissions: []model.Permission{{Code: "roles:read"}, {Code: "users:read"}}}}}
	v := Effective(u)
	if len(v.Permissions) != 3 || !Can(u, "users:read") || Can(u, "roles:manage") {
		t.Fatalf("%+v", v.Permissions)
	}
}
