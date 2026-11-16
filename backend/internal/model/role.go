package model

type Role struct {
	ID          uint         `json:"id" gorm:"primaryKey"`
	Name        string       `json:"name" gorm:"uniqueIndex;not null"`
	Description string       `json:"description"`
	System      bool         `json:"system"`
	Permissions []Permission `json:"permissions" gorm:"many2many:role_permissions;"`
	UserCount   int64        `json:"userCount" gorm:"-"`
}
