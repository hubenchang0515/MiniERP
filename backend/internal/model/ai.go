package model

import "time"

type AIResource struct {
	ID        uint            `json:"id" gorm:"primaryKey"`
	Name      string          `json:"name"`
	BaseURL   string          `json:"baseURL"`
	Keys      []AIProviderKey `json:"keys" gorm:"foreignKey:ResourceID"`
	Enabled   bool            `json:"enabled"`
	CreatedAt time.Time       `json:"createdAt"`
}

type AIProviderKey struct {
	ID         uint       `json:"id" gorm:"primaryKey"`
	ResourceID uint       `json:"-" gorm:"index;not null"`
	Name       string     `json:"name"`
	Secret     []byte     `json:"-"`
	Enabled    bool       `json:"enabled"`
	State      string     `json:"state"`
	RetryAt    *time.Time `json:"retryAt"`
	Revision   uint       `json:"-"`
	Requests   uint64     `json:"requests"`
	LastUsedAt *time.Time `json:"lastUsedAt"`
	CreatedAt  time.Time  `json:"createdAt"`
}

type AIModel struct {
	ID         uint           `json:"id" gorm:"primaryKey"`
	InternalID string         `json:"internalId" gorm:"uniqueIndex;not null"`
	ResourceID uint           `json:"resourceId" gorm:"index;not null"`
	ExternalID string         `json:"externalId"`
	Enabled    bool           `json:"enabled"`
	CreatedAt  time.Time      `json:"createdAt"`
	Resource   AIResource     `json:"-" gorm:"foreignKey:ResourceID"`
	Pricing    AIModelPricing `json:"pricing" gorm:"embedded;embeddedPrefix:pricing_"`
}

type AIModelPricing struct {
	InputPerMillion  int64 `json:"inputPerMillion" gorm:"not null;default:0"`
	OutputPerMillion int64 `json:"outputPerMillion" gorm:"not null;default:0"`
	ReservePoints    int64 `json:"reservePoints" gorm:"not null;default:0"`
	SkipStreamUsage  bool  `json:"skipStreamUsage" gorm:"not null;default:false"`
}
type AIKey struct {
	ID         uint       `json:"id" gorm:"primaryKey"`
	Name       string     `json:"name"`
	UserID     uint       `json:"userId" gorm:"index;not null"`
	TokenHash  string     `json:"-" gorm:"uniqueIndex;not null"`
	Prefix     string     `json:"-"`
	Enabled    bool       `json:"enabled"`
	Requests   uint64     `json:"requests"`
	LastUsedAt *time.Time `json:"lastUsedAt"`
	CreatedAt  time.Time  `json:"createdAt"`
	User       User       `json:"-" gorm:"foreignKey:UserID"`
}
