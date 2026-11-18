package database

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"minierp/internal/model"
)

func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&model.Permission{}, &model.Role{}, &model.User{}, &model.AIResource{}, &model.AIProviderKey{}, &model.AIModel{}, &model.AIKey{},
		&model.AIQuotaSetting{}, &model.AIUserQuota{}, &model.AIQuotaWeek{}, &model.AIQuotaGrant{}, &model.AIQuotaCharge{},
	); err != nil {
		return err
	}
	return db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&model.Permissions).Error
}
