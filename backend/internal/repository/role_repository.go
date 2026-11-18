package repository

import (
	"gorm.io/gorm"
	"minioa/internal/model"
)

func Role(db *gorm.DB, id uint) (model.Role, error) {
	var r model.Role
	err := db.Preload("Permissions").First(&r, id).Error
	return r, err
}
func roleSearch(db *gorm.DB, q string) *gorm.DB {
	query := db.Model(&model.Role{})
	if q != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+q+"%", "%"+q+"%")
	}
	return query
}
func Roles(db *gorm.DB, q string, page, size int) ([]model.Role, int64, error) {
	roles := []model.Role{}
	var total int64
	query := roleSearch(db, q)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Preload("Permissions").Order("id ASC").Offset((page - 1) * size).Limit(size).Find(&roles).Error; err != nil {
		return nil, 0, err
	}
	if len(roles) == 0 {
		return roles, total, nil
	}
	ids := make([]uint, len(roles))
	for i := range roles {
		ids[i] = roles[i].ID
	}
	var counts []struct {
		RoleID uint
		Count  int64
	}
	if err := db.Table("user_roles").Select("role_id, COUNT(*) AS count").Where("role_id IN ?", ids).Group("role_id").Scan(&counts).Error; err != nil {
		return nil, 0, err
	}
	byRole := map[uint]int64{}
	for _, count := range counts {
		byRole[count.RoleID] = count.Count
	}
	for i := range roles {
		roles[i].UserCount = byRole[roles[i].ID]
	}
	return roles, total, nil
}

type RoleOption struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func RoleOptions(db *gorm.DB, q string, page, size int) ([]RoleOption, int64, error) {
	items := []RoleOption{}
	var total int64
	query := roleSearch(db, q).Where("system = ?", false)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Select("id", "name", "description").Order("id ASC").Offset((page - 1) * size).Limit(size).Find(&items).Error
	return items, total, err
}
func SaveRole(db *gorm.DB, r *model.Role) error {
	if err := db.Omit("Permissions").Save(r).Error; err != nil {
		return err
	}
	return db.Model(r).Association("Permissions").Replace(r.Permissions)
}
