package model

import "time"

type User struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Username     string    `json:"username" gorm:"uniqueIndex;not null"`
	DisplayName  string    `json:"displayName"`
	PasswordHash string    `json:"-"`
	Enabled      bool      `json:"enabled"`
	TokenVersion uint      `json:"-"`
	Roles        []Role    `json:"roles" gorm:"many2many:user_roles;"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}
