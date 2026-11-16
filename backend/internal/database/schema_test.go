package database_test

import (
	"minierp/internal/database"
	"minierp/internal/model"
	"testing"
)

func TestMigrateAndReopen(t *testing.T) {
	dir := t.TempDir()
	db, err := database.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err = database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	var n int64
	db.Model(&model.Permission{}).Count(&n)
	if n != 4 {
		t.Fatalf("permissions = %d", n)
	}
	u := model.User{Username: "saved", Enabled: true}
	if err = db.Create(&u).Error; err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.Close()
	db, err = database.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ = db.DB()
	defer sqlDB.Close()
	var found model.User
	if err = db.First(&found, u.ID).Error; err != nil || found.Username != "saved" {
		t.Fatalf("persisted: %+v %v", found, err)
	}
}
