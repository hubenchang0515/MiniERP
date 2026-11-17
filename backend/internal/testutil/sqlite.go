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

func Databases(t *testing.T) (*gorm.DB, *gorm.DB) {
	t.Helper()
	dir := t.TempDir()
	writer, err := database.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	writePool, _ := writer.DB()
	t.Cleanup(func() { writePool.Close() })
	reader, err := database.OpenReader(dir)
	if err != nil {
		t.Fatal(err)
	}
	readPool, _ := reader.DB()
	t.Cleanup(func() { readPool.Close() })
	return writer, reader
}
