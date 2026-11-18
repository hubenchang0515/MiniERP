package handler_test

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"minioa/internal/service"
)

func TestAIProxyTruncatedStreamAbortsDownstream(t *testing.T) {
	r, s := server(t)
	s.Config.DataDir = t.TempDir()
	a := admin(t, r)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		conn, buffer, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close()
		// Close without a terminating chunk, as a broken upstream stream would.
		_, _ = buffer.WriteString("HTTP/1.1 200 OK\r\nContent-Type: text/event-stream\r\nTransfer-Encoding: chunked\r\n\r\n5\r\nhello\r\n")
		_ = buffer.Flush()
	}))
	defer upstream.Close()
	resource, err := saveProviderTestResource(s, service.AIResourceInput{Name: "test", BaseURL: upstream.URL + "/v1", Enabled: true}, []service.AIProviderKeyInput{{Name: "key", APIKey: "private-provider-key", Enabled: true}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.SaveAIModel(0, service.AIModelInput{InternalID: "test", ExternalID: "upstream", ResourceID: resource.ID, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	_, token, err := s.CreateAIKey(a.User.ID, service.AIKeyInput{Name: "test"})
	if err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	original := log.Writer()
	log.SetOutput(&logs)
	defer log.SetOutput(original)
	proxy := httptest.NewServer(r)
	defer proxy.Close()
	request, err := http.NewRequest("POST", proxy.URL+"/ai/v1/chat/completions", strings.NewReader(`{"model":"test","stream":true}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err == nil {
		t.Fatalf("broken stream ended cleanly: %q", body)
	}
	if string(body) != "hello" {
		t.Fatalf("partial response=%q", body)
	}
	// Wait for handler cleanup before inspecting the log buffer.
	proxy.Close()
	if !strings.Contains(logs.String(), "reason=upstream_truncated") {
		t.Fatalf("missing abort reason: %s", &logs)
	}
	if strings.Contains(logs.String(), token) || strings.Contains(logs.String(), "private-provider-key") || strings.Contains(logs.String(), "abort Handler") {
		t.Fatalf("unexpected request dump: %s", &logs)
	}
}
