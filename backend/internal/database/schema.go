package database

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"minierp/internal/model"
)

func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&model.Permission{}, &model.Role{}, &model.User{}); err != nil {
		return err
	}
	return db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&model.Permissions).Error
}
