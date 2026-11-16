package repository

import (
	"gorm.io/gorm"
	"minierp/internal/model"
)

func User(db *gorm.DB, id uint) (model.User, error) {
	var u model.User
	err := db.Preload("Roles.Permissions").First(&u, id).Error
	return u, err
}
func UserByName(db *gorm.DB, name string) (model.User, error) {
	var u model.User
	err := db.Preload("Roles.Permissions").Where("username = ?", name).First(&u).Error
	return u, err
}
func Users(db *gorm.DB, q string, page, size int) ([]model.User, int64, error) {
	var users []model.User
	var total int64
	query := db.Model(&model.User{})
	if q != "" {
		query = query.Where("username LIKE ? OR display_name LIKE ?", "%"+q+"%", "%"+q+"%")
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Preload("Roles.Permissions").Order("id ASC").Offset((page - 1) * size).Limit(size).Find(&users).Error
	return users, total, err
}
func SaveUser(db *gorm.DB, u *model.User) error {
	if err := db.Omit("Roles").Save(u).Error; err != nil {
		return err
	}
	return db.Model(u).Association("Roles").Replace(u.Roles)
}
