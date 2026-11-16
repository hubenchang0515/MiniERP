package service

import (
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"minierp/internal/config"
	"minierp/internal/model"
	"minierp/internal/repository"
	"strconv"
	"time"
)

type Service struct {
	DB        *gorm.DB
	Config    config.Config
	resources resourceMonitor
}
type claims struct {
	Version uint `json:"v"`
	jwt.RegisteredClaims
}

func (s *Service) Initialized() (bool, error) {
	var n int64
	err := s.DB.Model(&model.User{}).Count(&n).Error
	return n > 0, err
}
func (s *Service) Bootstrap(in Credentials) (Session, error) {
	if err := validateUsername(in.Username); err != nil {
		return Session{}, err
	}
	if err := validatePassword(in.Password); err != nil {
		return Session{}, err
	}
	if err := validateDisplay(in.DisplayName); err != nil {
		return Session{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return Session{}, err
	}
	var u model.User
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		var n int64
		if err := tx.Model(&model.User{}).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			return Fail(409, "already_initialized", "系统已完成初始化")
		}
		r := model.Role{Name: "系统管理员", Description: "系统内置角色，拥有所有权限", System: true, Permissions: model.Permissions}
		if err := repository.SaveRole(tx, &r); err != nil {
			return err
		}
		u = model.User{Username: in.Username, DisplayName: in.DisplayName, PasswordHash: string(hash), Enabled: true, TokenVersion: 1, Roles: []model.Role{r}}
		return repository.SaveUser(tx, &u)
	})
	if err != nil {
		return Session{}, Normalize(err)
	}
	return s.session(u)
}

// A fixed valid bcrypt hash ensures unknown usernames also perform a password comparison.
const dummyHash = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"

func (s *Service) Login(in Credentials) (Session, error) {
	u, err := repository.UserByName(s.DB, in.Username)
	hash := u.PasswordHash
	if err != nil {
		hash = dummyHash
	}
	compare := bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password))
	if err != nil || compare != nil || !u.Enabled {
		return Session{}, Fail(401, "invalid_credentials", "账号或密码错误，或账号已停用")
	}
	return s.session(u)
}

func (s *Service) ChangePassword(id uint, in ChangePasswordInput) error {
	if err := validatePassword(in.NewPassword); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return Normalize(s.DB.Transaction(func(tx *gorm.DB) error {
		u, err := repository.User(tx, id)
		if err != nil {
			return err
		}
		if !u.Enabled {
			return Fail(401, "unauthorized", "登录已失效，请重新登录")
		}
		if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(in.CurrentPassword)) != nil {
			return Fail(400, "invalid_password", "当前密码不正确")
		}
		return tx.Model(&u).Updates(map[string]any{
			"password_hash": string(hash),
			"token_version": u.TokenVersion + 1,
		}).Error
	}))
}
func (s *Service) session(u model.User) (Session, error) {
	now := time.Now()
	c := claims{u.TokenVersion, jwt.RegisteredClaims{Subject: strconv.FormatUint(uint64(u.ID), 10), Issuer: "minierp", IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(s.Config.JWTTTL))}}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString([]byte(s.Config.JWTSecret))
	return Session{token, Effective(u)}, err
}
func (s *Service) Authenticate(raw string) (model.User, error) {
	c := &claims{}
	token, err := jwt.ParseWithClaims(raw, c, func(t *jwt.Token) (any, error) { return []byte(s.Config.JWTSecret), nil }, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer("minierp"), jwt.WithExpirationRequired())
	invalid := Fail(401, "unauthorized", "登录已失效，请重新登录")
	if err != nil || !token.Valid {
		return model.User{}, invalid
	}
	id, err := strconv.ParseUint(c.Subject, 10, 32)
	if err != nil || id == 0 {
		return model.User{}, invalid
	}
	u, err := repository.User(s.DB, uint(id))
	if err != nil || !u.Enabled || u.TokenVersion != c.Version {
		return model.User{}, invalid
	}
	return u, nil
}
