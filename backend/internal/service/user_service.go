package service

import (
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"minierp/internal/model"
	"minierp/internal/repository"
)

func rolesFor(tx *gorm.DB, ids []uint) ([]model.Role, error) {
	roles := []model.Role{}
	seen := map[uint]bool{}
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		r, err := repository.Role(tx, id)
		if err != nil {
			return nil, Fail(400, "invalid_role", "所选角色不存在")
		}
		if r.System {
			return nil, Fail(409, "system_role", "系统管理员角色不可分配给其他账号")
		}
		roles = append(roles, r)
	}
	return roles, nil
}
func (s *Service) CreateUser(in UserInput) (model.User, error) {
	var u model.User
	if err := validateUsername(in.Username); err != nil {
		return u, err
	}
	if err := validateDisplay(in.DisplayName); err != nil {
		return u, err
	}
	if err := validatePassword(in.Password); err != nil {
		return u, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return u, err
	}
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		roles, err := rolesFor(tx, in.RoleIDs)
		if err != nil {
			return err
		}
		u = model.User{Username: in.Username, DisplayName: in.DisplayName, PasswordHash: string(hash), Enabled: true, TokenVersion: 1, Roles: roles}
		return repository.SaveUser(tx, &u)
	})
	return u, Normalize(err)
}
func isAdmin(u model.User) bool {
	for _, r := range u.Roles {
		if r.System {
			return true
		}
	}
	return false
}
func (s *Service) UpdateUser(id uint, in UserInput) (model.User, error) {
	var result model.User
	if err := validateDisplay(in.DisplayName); err != nil {
		return result, err
	}
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		before, err := repository.User(tx, id)
		if err != nil {
			return err
		}
		if in.Username != "" && in.Username != before.Username {
			return Fail(400, "username_immutable", "用户名创建后不可修改")
		}
		result = before
		result.DisplayName = in.DisplayName
		if in.RoleIDs != nil {
			if isAdmin(before) {
				return Fail(409, "system_admin", "系统管理员账号的角色不可修改")
			}
			roles, err := rolesFor(tx, in.RoleIDs)
			if err != nil {
				return err
			}
			result.Roles = roles
		}
		return repository.SaveUser(tx, &result)
	})
	return result, Normalize(err)
}
func (s *Service) SetStatus(id uint, enabled bool) error {
	return Normalize(s.DB.Transaction(func(tx *gorm.DB) error {
		u, err := repository.User(tx, id)
		if err != nil {
			return err
		}
		if isAdmin(u) && !enabled {
			return Fail(409, "system_admin", "系统管理员账号不可停用")
		}
		after := u
		after.Enabled = enabled
		if u.Enabled != enabled {
			after.TokenVersion++
		}
		return repository.SaveUser(tx, &after)
	}))
}
func (s *Service) ResetPassword(id uint, password string) error {
	if err := validatePassword(password); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return Normalize(s.DB.Transaction(func(tx *gorm.DB) error {
		u, err := repository.User(tx, id)
		if err != nil {
			return err
		}
		u.PasswordHash = string(hash)
		u.TokenVersion++
		return repository.SaveUser(tx, &u)
	}))
}
