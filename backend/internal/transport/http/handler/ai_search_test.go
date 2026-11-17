package handler_test

import (
	"encoding/json"
	"minierp/internal/model"
	"minierp/internal/service"
	"testing"
)

func TestAIListSearch(t *testing.T) {
	r, s := server(t)
	a := admin(t, r)
	u, err := s.CreateUser(service.UserInput{Username: "search-member", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	login, err := s.Login(service.Credentials{Username: u.Username, Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range []model.AIKey{
		{Name: "private-key", UserID: a.User.ID, TokenHash: "admin-hash", Enabled: true},
		{Name: "work-key", UserID: u.ID, TokenHash: "member-hash", Enabled: true},
		{Name: "personal", UserID: u.ID, TokenHash: "personal-hash", Enabled: true},
	} {
		if err := s.DB.Create(&k).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, resource := range []model.AIResource{
		{Name: "alpha", BaseURL: "https://alpha.example/v1", Enabled: true},
		{Name: "beta", BaseURL: "https://beta.example/v1", Enabled: true},
	} {
		if err := s.DB.Create(&resource).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, test := range []struct {
		path, token string
		total       int64
	}{
		{"/api/ai/resources?q=alpha", a.Token, 1},
		{"/api/ai/resources?q=beta.example", a.Token, 1},
		{"/api/ai/resources?q=missing", a.Token, 0},
		{"/api/ai/keys?all=true&q=search-member&pageSize=1", a.Token, 2},
		{"/api/ai/keys?all=true&q=private", a.Token, 1},
		{"/api/ai/keys?q=work-key", login.Token, 1},
		{"/api/ai/keys?q=private-key", login.Token, 0},
		{"/api/ai/keys?q=search-member", a.Token, 0},
	} {
		w := call(t, r, "GET", test.path, "", test.token, 200)
		var result struct {
			Total int64             `json:"total"`
			Items []json.RawMessage `json:"items"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || result.Total != test.total {
			t.Fatalf("%s: %s %v", test.path, w.Body.String(), err)
		}
		if test.total == 0 && len(result.Items) != 0 {
			t.Fatal("empty search returned records")
		}
	}
}
