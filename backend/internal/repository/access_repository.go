package repository

import (
	"gorm.io/gorm"
	"minioa/internal/model"
	"time"
)

type accessRow struct {
	UserID                                                uint
	Username, DisplayName                                 string
	UserEnabled                                           bool
	TokenVersion                                          uint
	CreatedAt, UpdatedAt                                  time.Time
	RoleID                                                uint
	RoleName, RoleDescription                             string
	RoleSystem                                            bool
	PermissionCode, PermissionName, PermissionDescription string
}

const accessColumns = `u.id AS user_id,u.username,u.display_name,u.enabled AS user_enabled,u.token_version,u.created_at,u.updated_at,r.id AS role_id,r.name AS role_name,r.description AS role_description,r.system AS role_system,p.code AS permission_code,p.name AS permission_name,p.description AS permission_description`

func accessQuery(db *gorm.DB) *gorm.DB {
	return db.Table("users AS u").Joins("LEFT JOIN user_roles ur ON ur.user_id = u.id").Joins("LEFT JOIN roles r ON r.id = ur.role_id").Joins("LEFT JOIN role_permissions rp ON rp.role_id = r.id").Joins("LEFT JOIN permissions p ON p.code = rp.permission_code").Order("r.id,p.code")
}

func accessUser(rows []accessRow) model.User {
	r := rows[0]
	u := model.User{ID: r.UserID, Username: r.Username, DisplayName: r.DisplayName, Enabled: r.UserEnabled, TokenVersion: r.TokenVersion, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, Roles: []model.Role{}}
	indices := map[uint]int{}
	for _, r := range rows {
		if r.RoleID == 0 {
			continue
		}
		index, exists := indices[r.RoleID]
		if !exists {
			index = len(u.Roles)
			indices[r.RoleID] = index
			u.Roles = append(u.Roles, model.Role{ID: r.RoleID, Name: r.RoleName, Description: r.RoleDescription, System: r.RoleSystem, Permissions: []model.Permission{}})
		}
		if r.PermissionCode != "" {
			u.Roles[index].Permissions = append(u.Roles[index].Permissions, model.Permission{Code: r.PermissionCode, Name: r.PermissionName, Description: r.PermissionDescription})
		}
	}
	return u
}

// Access data is read in one statement on every request, so revocation remains
// immediate without loading password hashes or caching stale role assignments.
func UserAccess(db *gorm.DB, id uint) (model.User, error) {
	var rows []accessRow
	if err := accessQuery(db).Select(accessColumns).Where("u.id = ?", id).Scan(&rows).Error; err != nil {
		return model.User{}, err
	}
	if len(rows) == 0 {
		return model.User{}, gorm.ErrRecordNotFound
	}
	return accessUser(rows), nil
}

func AIKeyAccess(db *gorm.DB, hash string) (model.AIKey, error) {
	var rows []struct {
		Access        accessRow `gorm:"embedded"`
		KeyID         uint
		KeyName       string
		KeyRequests   uint64
		KeyLastUsedAt *time.Time
		KeyCreatedAt  time.Time
	}
	if err := accessQuery(db).Joins("JOIN ai_keys k ON k.user_id = u.id").Select(accessColumns+",k.id AS key_id,k.name AS key_name,k.requests AS key_requests,k.last_used_at AS key_last_used_at,k.created_at AS key_created_at").Where("k.token_hash = ? AND k.enabled = ?", hash, true).Scan(&rows).Error; err != nil {
		return model.AIKey{}, err
	}
	if len(rows) == 0 {
		return model.AIKey{}, gorm.ErrRecordNotFound
	}
	users := make([]accessRow, len(rows))
	for i, r := range rows {
		users[i] = r.Access
	}
	r := rows[0]
	u := accessUser(users)
	return model.AIKey{ID: r.KeyID, Name: r.KeyName, UserID: u.ID, Enabled: true, Requests: r.KeyRequests, LastUsedAt: r.KeyLastUsedAt, CreatedAt: r.KeyCreatedAt, User: u}, nil
}
