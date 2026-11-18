package handler_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"minierp/internal/model"
	"minierp/internal/service"
)

func TestAIKeyLimitWithConcurrentCreation(t *testing.T) {
	r, s := server(t)
	a := admin(t, r)
	type attempt struct {
		key model.AIKey
		err error
	}
	const concurrent = service.MaxActiveAIKeys + 4
	results := make(chan attempt, concurrent)
	start := make(chan struct{})
	for i := range concurrent {
		go func() {
			<-start
			key, _, err := s.CreateAIKey(a.User.ID, service.AIKeyInput{Name: fmt.Sprintf("key-%d", i)})
			results <- attempt{key, err}
		}()
	}
	close(start)
	var active []model.AIKey
	for range concurrent {
		result := <-results
		if result.err != nil {
			var failure *service.Error
			if !errors.As(result.err, &failure) || failure.Code != "ai_key_limit" {
				t.Errorf("unexpected creation error: %v", result.err)
			}
			continue
		}
		active = append(active, result.key)
	}
	if len(active) != service.MaxActiveAIKeys {
		t.Fatalf("created %d keys, expected %d", len(active), service.MaxActiveAIKeys)
	}
	keyPath := fmt.Sprintf("/api/ai/keys/%d", active[0].ID)
	call(t, r, "POST", keyPath+"/rotate", "", a.Token, 200)
	rejected := call(t, r, "POST", "/api/ai/keys", `{"name":"overflow"}`, a.Token, 409)
	if !strings.Contains(rejected.Body.String(), `"code":"ai_key_limit"`) {
		t.Fatalf("unexpected response: %s", rejected.Body)
	}
	call(t, r, "DELETE", keyPath, "", a.Token, 204)
	call(t, r, "POST", "/api/ai/keys", `{"name":"replacement"}`, a.Token, 201)
	list := call(t, r, "GET", "/api/ai/keys?q=no-match", "", a.Token, 200)
	var data struct {
		ActiveCount int
		Limit       int
		Total       int
	}
	if err := json.Unmarshal(list.Body.Bytes(), &data); err != nil ||
		data.ActiveCount != service.MaxActiveAIKeys || data.Limit != service.MaxActiveAIKeys || data.Total != 0 {
		t.Fatalf("capacity should be independent of filters: %s, %v", list.Body, err)
	}
}
