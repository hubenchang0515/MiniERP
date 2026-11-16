package testutil

import (
	"gorm.io/gorm"
	"minierp/internal/database"
	"testing"
)

func DB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { sqlDB.Close() })
	return db
}
