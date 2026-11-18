package model

import "time"

type AIQuotaSetting struct {
	ID      uint `gorm:"primaryKey;autoIncrement:false"`
	Enabled bool `gorm:"not null"`
}

type AIUserQuota struct {
	UserID       uint  `gorm:"primaryKey;autoIncrement:false"`
	WeeklyPoints int64 `gorm:"not null"`
}

func (AIUserQuota) TableName() string { return "ai_user_quota" }

type AIQuotaWeek struct {
	UserID   uint   `gorm:"primaryKey;autoIncrement:false"`
	Week     string `gorm:"primaryKey"`
	Added    int64  `gorm:"not null;default:0"`
	Used     int64  `gorm:"not null;default:0"`
	Reserved int64  `gorm:"not null;default:0"`
}

type AIQuotaGrant struct {
	ID        uint
	UserID    uint   `gorm:"index:idx_ai_grant_user_week;not null"`
	Week      string `gorm:"index:idx_ai_grant_user_week;not null"`
	AdminID   uint   `gorm:"not null"`
	Points    int64  `gorm:"not null"`
	Reason    string `gorm:"not null"`
	CreatedAt time.Time
}

type AIQuotaCharge struct {
	RequestID        string    `json:"requestId" gorm:"primaryKey"`
	UserID           uint      `json:"-" gorm:"index:idx_ai_charge_user_time,priority:1;not null"`
	KeyID            uint      `json:"-" gorm:"not null"`
	Week             string    `json:"week" gorm:"not null"`
	Model            string    `json:"model" gorm:"not null"`
	Status           string    `json:"status" gorm:"index;not null"`
	Reserved         int64     `json:"reservedMicros,string" gorm:"not null"`
	Charged          int64     `json:"chargedMicros,string" gorm:"not null;default:0"`
	InputPerMillion  int64     `json:"inputPerMillion" gorm:"not null"`
	OutputPerMillion int64     `json:"outputPerMillion" gorm:"not null"`
	InputTokens      *int64    `json:"inputTokens"`
	OutputTokens     *int64    `json:"outputTokens"`
	CreatedAt        time.Time `json:"createdAt" gorm:"index:idx_ai_charge_user_time,priority:2"`
}
