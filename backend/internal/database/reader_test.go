package database_test

import (
	"minierp/internal/database"
	"minierp/internal/model"
	"testing"
)

func TestReaderIsQueryOnlyAndSeesCommittedChanges(t *testing.T) {
	dir := t.TempDir()
	writer, err := database.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	w, _ := writer.DB()
	defer w.Close()
	reader, err := database.OpenReader(dir)
	if err != nil {
		t.Fatal(err)
	}
	r, _ := reader.DB()
	defer r.Close()
	u := model.User{Username: "reader-check", Enabled: true}
	if err := writer.Create(&u).Error; err != nil {
		t.Fatal(err)
	}
	var found model.User
	if err := reader.First(&found, u.ID).Error; err != nil || !found.Enabled {
		t.Fatal("reader did not see committed insert")
	}
	if err := reader.Model(&model.User{}).Where("id = ?", u.ID).Update("enabled", false).Error; err == nil {
		t.Fatal("reader connection accepted a write")
	}
	if err := writer.Model(&model.User{}).Where("id = ?", u.ID).Update("enabled", false).Error; err != nil {
		t.Fatal(err)
	}
	if err := reader.First(&found, u.ID).Error; err != nil || found.Enabled {
		t.Fatal("reader did not see revocation immediately")
	}
}
