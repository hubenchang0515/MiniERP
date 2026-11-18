package database_test

import (
	"minioa/internal/database"
	"minioa/internal/model"
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
	if n != int64(len(model.Permissions)) {
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

func TestAddAIQuotaToExistingDatabase(t *testing.T) {
	dir := t.TempDir()
	db, err := database.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	defer pool.Close()
	resource := model.AIResource{Name: "existing", BaseURL: "https://example.com/v1", Enabled: true}
	if err := db.Create(&resource).Error; err != nil {
		t.Fatal(err)
	}
	mapping := model.AIModel{InternalID: "existing", ExternalID: "provider", ResourceID: resource.ID, Enabled: true}
	if err := db.Create(&mapping).Error; err != nil {
		t.Fatal(err)
	}
	for _, column := range []string{"pricing_input_per_million", "pricing_output_per_million", "pricing_reserve_points", "pricing_skip_stream_usage"} {
		if err := db.Exec("ALTER TABLE ai_models DROP COLUMN " + column).Error; err != nil {
			t.Fatal(err)
		}
	}
	pool.Close()
	db, err = database.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	pool, _ = db.DB()
	defer pool.Close()
	var existing model.AIModel
	if err := db.First(&existing, mapping.ID).Error; err != nil || existing.InternalID != "existing" ||
		existing.ResourceID != resource.ID || !existing.Enabled || existing.Pricing.ReservePoints != 0 {
		t.Fatalf("the existing mapping changed: %+v, %v", existing, err)
	}
}
