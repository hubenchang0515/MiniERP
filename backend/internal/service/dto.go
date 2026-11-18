package service

import "minioa/internal/model"

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type ChangePasswordInput struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}
type UserInput struct {
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	Password    string `json:"password"`
	RoleIDs     []uint `json:"roleIds"`
}
type RoleInput struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}
type CurrentUser struct {
	model.User
	Permissions []string `json:"permissions"`
}
type Session struct {
	Token string      `json:"token"`
	User  CurrentUser `json:"user"`
}

func Effective(u model.User) CurrentUser {
	seen := map[string]bool{}
	for _, r := range u.Roles {
		for _, p := range r.Permissions {
			seen[p.Code] = true
			if p.Code == "users:manage" {
				seen["users:read"] = true
			}
			if p.Code == "roles:manage" {
				seen["roles:read"] = true
			}
		}
	}
	result := CurrentUser{User: u, Permissions: []string{}}
	for _, p := range model.Permissions {
		if seen[p.Code] {
			result.Permissions = append(result.Permissions, p.Code)
		}
	}
	return result
}
func Can(u model.User, code string) bool {
	for _, p := range Effective(u).Permissions {
		if p == code {
			return true
		}
	}
	return false
}
