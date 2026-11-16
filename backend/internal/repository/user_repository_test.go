package repository

import (
	"gorm.io/gorm"
	"minierp/internal/model"
	"minierp/internal/testutil"
	"testing"
)

func TestUserAssociationRollback(t *testing.T) {
	db := testutil.DB(t)
	r := model.Role{Name: "reader", Permissions: []model.Permission{model.Permissions[0]}}
	if err := SaveRole(db, &r); err != nil {
		t.Fatal(err)
	}
	u := model.User{Username: "alice", Enabled: true, Roles: []model.Role{r}}
	if err := db.Transaction(func(tx *gorm.DB) error { return SaveUser(tx, &u) }); err != nil {
		t.Fatal(err)
	}
	found, err := UserByName(db, "alice")
	if err != nil || len(found.Roles) != 1 || len(found.Roles[0].Permissions) != 1 {
		t.Fatalf("%+v %v", found, err)
	}
	found.DisplayName = "张三"
	found.Roles = []model.Role{}
	if err := db.Transaction(func(tx *gorm.DB) error { return SaveUser(tx, &found) }); err != nil {
		t.Fatal(err)
	}
	found, err = User(db, u.ID)
	if err != nil || len(found.Roles) != 0 || found.DisplayName != "张三" {
		t.Fatalf("%+v %v", found, err)
	}
	_, total, err := Users(db, "张三", 1, 20)
	if err != nil || total != 1 {
		t.Fatalf("%d %v", total, err)
	}
}
