package repository

import (
	"fmt"
	"gorm.io/gorm"
	"minioa/internal/model"
	"minioa/internal/testutil"
	"testing"
)

func TestRolesCountsAndPermissions(t *testing.T) {
	db := testutil.DB(t)
	r := model.Role{Name: "reader", Permissions: []model.Permission{model.Permissions[0]}}
	if err := SaveRole(db, &r); err != nil {
		t.Fatal(err)
	}
	u := model.User{Username: "alice", Enabled: true, Roles: []model.Role{r}}
	if err := SaveUser(db, &u); err != nil {
		t.Fatal(err)
	}
	roles, total, err := Roles(db, "", 1, 10)
	if err != nil || total != 1 || len(roles) != 1 || roles[0].UserCount != 1 {
		t.Fatalf("%+v %v", roles, err)
	}
	r.Permissions = []model.Permission{}
	if err := SaveRole(db, &r); err != nil {
		t.Fatal(err)
	}
	found, err := Role(db, r.ID)
	if err != nil || len(found.Permissions) != 0 {
		t.Fatalf("%+v %v", found, err)
	}
}

func TestRolePaginationSearchAndBatchCounts(t *testing.T) {
	db := testutil.DB(t)
	for i := 0; i < 25; i++ {
		r := model.Role{Name: fmt.Sprintf("reader-%02d", i), Description: "searchable", Permissions: []model.Permission{model.Permissions[0]}}
		if err := SaveRole(db, &r); err != nil {
			t.Fatal(err)
		}
		if i == 10 {
			if err := SaveUser(db, &model.User{Username: "alice", Roles: []model.Role{r}}); err != nil {
				t.Fatal(err)
			}
		}
	}
	queries := 0
	db.Callback().Query().After("gorm:query").Register("test:query-count", func(*gorm.DB) { queries++ })
	roles, total, err := Roles(db, "searchable", 2, 10)
	if err != nil || total != 25 || len(roles) != 10 || roles[0].Name != "reader-10" || roles[0].UserCount != 1 || roles[1].UserCount != 0 {
		t.Fatalf("page: %+v %d %v", roles, total, err)
	}
	if queries > 6 {
		t.Fatalf("per-role queries returned: %d", queries)
	}
	roles, total, err = Roles(db, "reader-24", 1, 10)
	if err != nil || total != 1 || len(roles) != 1 {
		t.Fatalf("search: %+v %d %v", roles, total, err)
	}
	roles, total, err = Roles(db, "reader", 9, 10)
	if err != nil || total != 25 || roles == nil || len(roles) != 0 {
		t.Fatalf("empty page: %+v %d %v", roles, total, err)
	}
}
