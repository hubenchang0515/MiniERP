package handler_test

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"minierp/internal/model"
	"minierp/internal/service"
)

func TestAIWeeklyQuotaProxyAndPermissions(t *testing.T) {
	r, s := server(t)
	s.Config.DataDir = t.TempDir()
	a := admin(t, r)
	var calls atomic.Int64
	finishStream := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		calls.Add(1)
		var payload struct {
			StreamOptions map[string]any `json:"stream_options"`
		}
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		switch req.URL.Query().Get("mode") {
		case "reject":
			w.WriteHeader(400)
		case "failover":
			if req.Header.Get("Authorization") == "Bearer first-key" {
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(429)
				_, _ = io.WriteString(w, `{"error":{"code":"insufficient_quota"}}`)
				return
			}
			_, _ = io.WriteString(w, `{"usage":{"prompt_tokens":2000,"completion_tokens":500}}`)
		case "missing":
			_, _ = io.WriteString(w, `{"choices":[]}`)
		case "cancel":
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(w, "data: {\"choices\":[]}\n\n")
			w.(http.Flusher).Flush()
			<-req.Context().Done()
		default:
			if payload.StreamOptions["include_usage"] != true || payload.StreamOptions["custom_field"] != true {
				t.Errorf("stream options not merged: %+v", payload.StreamOptions)
			}
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"first\"}}]}\n\n")
			w.(http.Flusher).Flush()
			select {
			case <-finishStream:
				_, _ = io.WriteString(w, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":2000,\"completion_tokens\":500}}\n\ndata: [DONE]\n\n")
			case <-req.Context().Done():
			}
		}
	}))
	defer upstream.Close()
	resource, err := saveProviderTestResource(s, service.AIResourceInput{
		Name: "billing", BaseURL: upstream.URL + "/v1", Enabled: true,
	}, []service.AIProviderKeyInput{{Name: "one", APIKey: "first-key", Enabled: true}, {Name: "two", APIKey: "second-key", Enabled: true}})
	if err != nil {
		t.Fatal(err)
	}
	mapping, err := s.SaveAIModel(0, service.AIModelInput{
		InternalID: "chat", ExternalID: "provider-chat", ResourceID: resource.ID, Enabled: true,
		Pricing: &model.AIModelPricing{InputPerMillion: 1000, OutputPerMillion: 4000, ReservePoints: 5},
	})
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	for _, name := range []string{"one", "two"} {
		_, secret, err := s.CreateAIKey(a.User.ID, service.AIKeyInput{Name: name})
		if err != nil {
			t.Fatal(err)
		}
		keys = append(keys, secret)
	}
	const chat = "/ai/v1/chat/completions"
	const request = `{"model":"chat"}`
	const myQuota = "/api/ai/quota"
	const adminQuota = "/api/ai/quota/users/1"
	settings := call(t, r, "GET", myQuota+"/settings", "", a.Token, 200)
	if !strings.Contains(settings.Body.String(), `"enabled":false`) {
		t.Fatalf("expected fresh settings to be disabled: %s", settings.Body)
	}
	call(t, r, "PATCH", myQuota+"/settings", `{"enabled":true}`, a.Token, 204)
	settings = call(t, r, "GET", myQuota+"/settings", "", a.Token, 200)
	if !strings.Contains(settings.Body.String(), `"enabled":true`) {
		t.Fatalf("expected persisted settings: %s", settings.Body)
	}
	call(t, r, "POST", chat, request, keys[0], 403)
	call(t, r, "GET", "/ai/v1/models", "", keys[0], 200)
	call(t, r, "PATCH", adminQuota, `{"weeklyPoints":5}`, a.Token, 204)
	gateway := httptest.NewServer(r)
	defer gateway.Close()
	client := gateway.Client()
	client.Timeout = 5 * time.Second
	stream := func(token, query, payload string) *http.Response {
		t.Helper()
		req, err := http.NewRequestWithContext(t.Context(), "POST", gateway.URL+chat+query, strings.NewReader(payload))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != 200 {
			resp.Body.Close()
			t.Fatalf("stream status=%d", resp.StatusCode)
		}
		return resp
	}
	response := stream(keys[0], "", `{"model":"chat","stream":true,"stream_options":{"custom_field":true}}`)
	defer response.Body.Close()
	buffered := bufio.NewReader(response.Body)
	if line, err := buffered.ReadString('\n'); err != nil || !strings.Contains(line, "first") {
		t.Fatalf("the first event must arrive before the stream ends: %q, %v", line, err)
	}
	denied := call(t, r, "POST", chat, request, keys[1], 403)
	if !strings.Contains(denied.Body.String(), "insufficient_quota") || calls.Load() != 1 {
		t.Fatal("second key was not blocked against the shared reservation")
	}
	close(finishStream)
	if rest, err := io.ReadAll(buffered); err != nil || !strings.Contains(string(rest), "[DONE]") {
		t.Fatalf("stream completion: %q, %v", rest, err)
	}
	getBalance := func() service.AIQuotaBalance {
		t.Helper()
		record := call(t, r, "GET", myQuota, "", a.Token, 200)
		var balance service.AIQuotaBalance
		if err := json.Unmarshal(record.Body.Bytes(), &balance); err != nil {
			t.Fatal(err)
		}
		return balance
	}
	const point = service.AIPointMicros
	waitProxyCondition(t, func() bool { return getBalance().Used == 4*point })
	if balance := getBalance(); balance.Available != point || balance.Reserved != 0 {
		t.Fatalf("settled balance: %+v", balance)
	}
	call(t, r, "POST", adminQuota+"/grants", `{"points":20,"reason":" "}`, a.Token, 400)
	call(t, r, "POST", adminQuota+"/grants", `{"points":20,"reason":"测试项目"}`, a.Token, 204)

	call(t, r, "POST", chat+"?mode=reject", request, keys[0], 400)
	if got := getBalance(); got.Used != 4*point || got.Reserved != 0 {
		t.Fatalf("rejected request consumed quota: %+v", got)
	}
	before := calls.Load()
	call(t, r, "POST", chat+"?mode=failover", request, keys[1], 200)
	if got := getBalance(); got.Used != 8*point || got.Reserved != 0 || calls.Load()-before != 2 {
		t.Fatalf("failover billed more than once or did not retry: %+v, calls=%d", got, calls.Load()-before)
	}
	call(t, r, "POST", chat+"?mode=missing", request, keys[0], 200)
	if got := getBalance(); got.Used != 13*point || got.Reserved != 0 {
		t.Fatalf("missing usage: %+v", got)
	}
	cancelled := stream(keys[0], "?mode=cancel", `{"model":"chat","stream":true}`)
	cancelled.Body.Close()
	waitProxyCondition(t, func() bool { return getBalance().Used == 18*point })
	wantUsage := service.AIQuotaUsageTotal{InputTokens: 4_000, OutputTokens: 1000, Charged: 18 * point}
	if got := getBalance().TotalUsage; got != wantUsage {
		t.Fatalf("total usage=%+v, want %+v", got, wantUsage)
	}
	ledger := call(t, r, "GET", myQuota+"/usage", "", a.Token, 200)
	var page struct {
		Items []model.AIQuotaCharge
		Total int
	}
	if err := json.Unmarshal(ledger.Body.Bytes(), &page); err != nil || page.Total != 5 {
		t.Fatalf("ledger: %s (%v)", ledger.Body, err)
	}
	statuses := map[string]int{}
	for _, entry := range page.Items {
		statuses[entry.Status]++
	}
	if statuses["actual"] != 2 || statuses["estimated"] != 2 || statuses["released"] != 1 {
		t.Fatalf("ledger statuses: %v", statuses)
	}
	_, err = s.SaveAIModel(mapping.ID, service.AIModelInput{
		InternalID: "chat", ExternalID: "provider-chat", ResourceID: resource.ID, Enabled: true,
		Pricing: &model.AIModelPricing{},
	})
	if err != nil {
		t.Fatal(err)
	}
	unpriced := call(t, r, "POST", chat, request, keys[0], 403)
	if !strings.Contains(unpriced.Body.String(), "ai_model_unpriced") {
		t.Fatal(unpriced.Body.String())
	}

	if _, err := s.CreateUser(service.UserInput{Username: "quota-member", Password: "password123"}); err != nil {
		t.Fatal(err)
	}
	member, err := s.Login(service.Credentials{Username: "quota-member", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{adminQuota, myQuota + "/settings"} {
		call(t, r, "PATCH", path, `{}`, member.Token, 403)
	}
	call(t, r, "GET", adminQuota, "", member.Token, 403)
	call(t, r, "GET", myQuota+"/users", "", member.Token, 403)
	call(t, r, "GET", myQuota+"/settings", "", member.Token, 403)
	call(t, r, "POST", adminQuota+"/grants", `{}`, member.Token, 403)
	mine := call(t, r, "GET", myQuota+"/usage?userId=1", "", member.Token, 200)
	if !strings.Contains(mine.Body.String(), `"total":0`) {
		t.Fatalf("another user's usage was visible: %s", mine.Body)
	}
	var quotaPage struct {
		Items []service.AIQuotaUser
		Total int
	}
	list := call(t, r, "GET", myQuota+"/users?pageSize=1", "", a.Token, 200)
	if err := json.Unmarshal(list.Body.Bytes(), &quotaPage); err != nil || quotaPage.Total != 2 || len(quotaPage.Items) != 1 {
		t.Fatalf("list: %s, %v", list.Body, err)
	}
	adminBalance := quotaPage.Items[0]
	if adminBalance.UserID != a.User.ID || !adminBalance.Configured || adminBalance.WeeklyPoints != 5 ||
		adminBalance.Added != 20*point || adminBalance.Used != 18*point || adminBalance.Available != 7*point ||
		adminBalance.TotalUsage != wantUsage {
		t.Fatalf("wrong balance totals: %+v", adminBalance)
	}
	list = call(t, r, "GET", myQuota+"/users?q=quota-member", "", a.Token, 200)
	if err := json.Unmarshal(list.Body.Bytes(), &quotaPage); err != nil || quotaPage.Total != 1 || len(quotaPage.Items) != 1 ||
		quotaPage.Items[0].UserID != member.User.ID || quotaPage.Items[0].Configured || quotaPage.Items[0].Available != 0 ||
		quotaPage.Items[0].TotalUsage != (service.AIQuotaUsageTotal{}) {
		t.Fatalf("unconfigured user missing from search: %s, %v", list.Body, err)
	}
}
