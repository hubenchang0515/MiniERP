package service

import (
	"gorm.io/gorm"
	"minierp/internal/model"
	"minierp/internal/repository"
	"strings"
)

func (s *Service) SaveRole(id uint, in RoleInput) (model.Role, error) {
	var r model.Role
	if err := validateRole(in); err != nil {
		return r, err
	}
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if id != 0 {
			var err error
			r, err = repository.Role(tx, id)
			if err != nil {
				return err
			}
			if r.System {
				return Fail(409, "system_role", "系统管理员角色不可修改")
			}
		}
		permissions := []model.Permission{}
		seen := map[string]bool{}
		for _, code := range in.Permissions {
			valid := false
			for _, p := range model.Permissions {
				if p.Code == code {
					valid = true
					if !seen[code] {
						permissions = append(permissions, p)
						seen[code] = true
					}
					break
				}
			}
			if !valid {
				return Fail(400, "invalid_permission", "未知权限: "+code)
			}
		}
		r.Name = strings.TrimSpace(in.Name)
		r.Description = in.Description
		r.Permissions = permissions
		return repository.SaveRole(tx, &r)
	})
	return r, Normalize(err)
}
func (s *Service) DeleteRole(id uint) error {
	return Normalize(s.DB.Transaction(func(tx *gorm.DB) error {
		r, err := repository.Role(tx, id)
		if err != nil {
			return err
		}
		if r.System {
			return Fail(409, "system_role", "系统管理员角色不可删除")
		}
		var n int64
		if err := tx.Table("user_roles").Where("role_id = ?", id).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			return Fail(409, "role_in_use", "该角色已分配给用户，无法删除")
		}
		if err := tx.Model(&r).Association("Permissions").Clear(); err != nil {
			return err
		}
		return tx.Delete(&r).Error
	}))
}
