package handler_test

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"minierp/internal/service"
)

func proxyFixture(t *testing.T, upstreamHandler http.HandlerFunc, configure func(*service.Service), http2 bool) (*http.Client, string, *service.Service) {
	t.Helper()
	r, s := server(t)
	s.Config.DataDir = t.TempDir()
	if configure != nil {
		configure(s)
	}
	a := admin(t, r)
	upstream := httptest.NewServer(upstreamHandler)
	t.Cleanup(upstream.Close)
	resource, err := saveProviderTestResource(s, service.AIResourceInput{Name: "concurrency", BaseURL: upstream.URL + "/v1", Enabled: true}, []service.AIProviderKeyInput{{Name: "one", APIKey: "dummy-one", Enabled: true}, {Name: "two", APIKey: "dummy-two", Enabled: true}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveAIModel(0, service.AIModelInput{InternalID: "test", ExternalID: "upstream", ResourceID: resource.ID, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	_, token, err := s.CreateAIKey(a.User.ID, service.AIKeyInput{Name: "test"})
	if err != nil {
		t.Fatal(err)
	}
	proxy := httptest.NewUnstartedServer(r)
	// A short ordinary API deadline reproduces production's 30s deadline
	// without waiting 30s; AI writes must renew their own deadline.
	proxy.Config.WriteTimeout = 50 * time.Millisecond
	if http2 {
		proxy.EnableHTTP2 = true
		proxy.StartTLS()
	} else {
		proxy.Start()
	}
	t.Cleanup(proxy.Close)
	client := proxy.Client()
	client.Timeout = 10 * time.Second
	transport := client.Transport.(*http.Transport)
	transport.MaxIdleConnsPerHost = 64
	transport.MaxIdleConns = 64
	if http2 {
		transport.MaxConnsPerHost = 1
	}
	requestURL := proxy.URL + "/ai/v1/chat/completions"
	// Supply authentication in a wrapper, keeping request setup identical.
	client.Transport = authenticatedTransport{RoundTripper: transport, token: token}
	return client, requestURL, s
}

type authenticatedTransport struct {
	http.RoundTripper
	token string
}

func (t authenticatedTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	out := request.Clone(request.Context())
	out.Header.Set("Authorization", "Bearer "+t.token)
	out.Header.Set("Content-Type", "application/json")
	return t.RoundTripper.RoundTrip(out)
}

func proxyRequest(ctx context.Context, client *http.Client, url, content string) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(`{"model":"test","stream":true,"messages":[{"role":"user","content":"`+content+`"}]}`))
	if err != nil {
		return nil, err
	}
	return client.Do(request)
}

func waitProxyCondition(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for !condition() {
		select {
		case <-deadline.C:
			t.Fatal("timed out waiting for proxy cleanup")
		case <-ticker.C:
		}
	}
}

func TestAIProxyConcurrentStreamsCancellationAndCapacity(t *testing.T) {
	previous := gin.DefaultWriter
	gin.DefaultWriter = io.Discard
	defer func() { gin.DefaultWriter = previous }()
	for _, useHTTP2 := range []bool{false, true} {
		t.Run(fmt.Sprintf("http2=%v", useHTTP2), func(t *testing.T) {
			var active, calls atomic.Int64
			var keyMu sync.Mutex
			keyCalls := map[string]int{}
			client, url, _ := proxyFixture(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				active.Add(1)
				defer active.Add(-1)
				keyMu.Lock()
				keyCalls[r.Header.Get("Authorization")]++
				keyMu.Unlock()
				w.Header().Set("Content-Type", "text/event-stream")
				w.Header().Set("X-Request-ID", "untrusted-provider-id")
				ticker := time.NewTicker(15 * time.Millisecond)
				defer ticker.Stop()
				for {
					_, _ = io.WriteString(w, "data: tick\n\n")
					w.(http.Flusher).Flush()
					select {
					case <-r.Context().Done():
						return
					case <-ticker.C:
					}
				}
			}, nil, useHTTP2)
			type stream struct {
				response *http.Response
				cancel   context.CancelFunc
				err      error
			}
			results := make(chan stream, 32)
			streamsCtx, cancelStreams := context.WithCancel(t.Context())
			var pending sync.WaitGroup
			pending.Add(32)
			t.Cleanup(func() {
				cancelStreams()
				pending.Wait()
				close(results)
				for stream := range results {
					if stream.response != nil {
						stream.response.Body.Close()
					}
				}
			})
			for range 32 {
				go func() {
					defer pending.Done()
					ctx, cancel := context.WithCancel(streamsCtx)
					response, err := proxyRequest(ctx, client, url, "hold")
					results <- stream{response, cancel, err}
				}()
			}
			streams := make([]stream, 0, 32)
			t.Cleanup(func() {
				for _, stream := range streams {
					stream.cancel()
					if stream.response != nil {
						stream.response.Body.Close()
					}
				}
			})
			ids := map[string]bool{}
			for range 32 {
				stream := <-results
				streams = append(streams, stream)
				if stream.err != nil {
					t.Fatal(stream.err)
				}
				if stream.response.StatusCode != 200 {
					body, _ := io.ReadAll(stream.response.Body)
					t.Fatalf("status=%d body=%s", stream.response.StatusCode, body)
				}
				if useHTTP2 && stream.response.ProtoMajor != 2 {
					t.Fatal("HTTP/2 not negotiated")
				}
				id := stream.response.Header.Get("X-Request-ID")
				if id == "" || id == "untrusted-provider-id" || ids[id] || len(stream.response.Header.Values("X-Request-ID")) != 1 {
					t.Fatalf("invalid request ID: %q", id)
				}
				ids[id] = true
			}
			if active.Load() != 32 || calls.Load() != 32 {
				t.Fatalf("active=%d calls=%d", active.Load(), calls.Load())
			}
			keyMu.Lock()
			one, two := keyCalls["Bearer dummy-one"], keyCalls["Bearer dummy-two"]
			keyMu.Unlock()
			if one != 16 || two != 16 {
				t.Fatalf("unbalanced held streams: %d/%d", one, two)
			}
			rejected, err := proxyRequest(t.Context(), client, url, "over capacity")
			if err != nil {
				t.Fatal(err)
			}
			io.Copy(io.Discard, rejected.Body)
			rejected.Body.Close()
			if rejected.StatusCode != 429 || rejected.Header.Get("Retry-After") != "1" || calls.Load() != 32 {
				t.Fatalf("capacity rejection: status=%d retry=%q calls=%d", rejected.StatusCode, rejected.Header.Get("Retry-After"), calls.Load())
			}
			streams[0].cancel()
			streams[0].response.Body.Close()
			waitProxyCondition(t, func() bool { return active.Load() == 31 })
			// All remaining requests keep receiving data after their sibling is canceled.
			for _, stream := range streams[1:] {
				line, err := bufio.NewReader(stream.response.Body).ReadString('\n')
				if err != nil || line != "data: tick\n" {
					t.Fatalf("sibling stream interrupted: %q %v", line, err)
				}
			}
			var replacement *http.Response
			waitProxyCondition(t, func() bool {
				response, err := proxyRequest(t.Context(), client, url, "replacement")
				if err != nil {
					t.Fatal(err)
				}
				if response.StatusCode == 200 {
					replacement = response
					return true
				}
				response.Body.Close()
				if response.StatusCode != 429 {
					t.Fatalf("replacement status=%d", response.StatusCode)
				}
				return false
			})
			replacement.Body.Close()
			for _, stream := range streams {
				stream.cancel()
				stream.response.Body.Close()
			}
			waitProxyCondition(t, func() bool { return active.Load() == 0 })
			t.Logf("32 concurrent streams balanced %d/%d; 33rd returned 429; cancellation freed capacity without interrupting siblings", one, two)
		})
	}
}

func TestAIProxyIdleTimeoutAndActiveLongStream(t *testing.T) {
	for _, useHTTP2 := range []bool{false, true} {
		t.Run(fmt.Sprintf("http2=%v", useHTTP2), func(t *testing.T) { testAIProxyIdleTimeoutAndActiveLongStream(t, useHTTP2) })
	}
}

func testAIProxyIdleTimeoutAndActiveLongStream(t *testing.T, useHTTP2 bool) {
	var active atomic.Int64
	client, url, s := proxyFixture(t, func(w http.ResponseWriter, r *http.Request) {
		active.Add(1)
		defer active.Add(-1)
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "no headers") {
			<-r.Context().Done()
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: first\n\n")
		w.(http.Flusher).Flush()
		if strings.Contains(string(body), "idle body") {
			<-r.Context().Done()
			return
		}
		for range 12 {
			select {
			case <-r.Context().Done():
				return
			case <-time.After(40 * time.Millisecond):
			}
			io.WriteString(w, "data: tick\n\n")
			w.(http.Flusher).Flush()
		}
		io.WriteString(w, "data: [DONE]\n\n")
	}, func(s *service.Service) {
		s.Config.AIIdleTimeout = 200 * time.Millisecond
		s.Config.AIMaxConcurrentRequests = 1
	}, useHTTP2)
	for _, test := range []struct {
		content   string
		status    int
		readError bool
	}{
		{"no headers", 504, false}, {"idle body", 200, true}, {"long active stream", 200, false},
	} {
		t.Run(test.content, func(t *testing.T) {
			response, err := proxyRequest(t.Context(), client, url, test.content)
			if err != nil {
				t.Fatal(err)
			}
			body, err := io.ReadAll(response.Body)
			response.Body.Close()
			if response.StatusCode != test.status || (err != nil) != test.readError {
				t.Fatalf("status=%d err=%v body=%s", response.StatusCode, err, body)
			}
			if test.content == "long active stream" && !strings.Contains(string(body), "[DONE]") {
				t.Fatalf("active stream cut off: %s", body)
			}
			waitProxyCondition(t, func() bool { return active.Load() == 0 })
			waitProxyCondition(t, func() bool {
				release, err := s.AcquireAIProxy()
				if err != nil {
					return false
				}
				release()
				return true
			})
		})
	}
}

func TestAIProxyConcurrentQuotaFailover(t *testing.T) {
	var successes atomic.Int64
	client, url, _ := proxyFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer dummy-one" {
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(429)
			io.WriteString(w, `{"error":{"code":"insufficient_quota"}}`)
			return
		}
		successes.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: first\n\n")
		w.(http.Flusher).Flush()
		io.WriteString(w, "data: [DONE]\n\n")
	}, nil, false)
	var pending sync.WaitGroup
	results := make(chan error, 16)
	for range 16 {
		pending.Add(1)
		go func() {
			defer pending.Done()
			response, err := proxyRequest(t.Context(), client, url, "quota failover")
			if err != nil {
				results <- err
				return
			}
			body, err := io.ReadAll(response.Body)
			response.Body.Close()
			if err == nil && (response.StatusCode != 200 || !strings.Contains(string(body), "[DONE]")) {
				err = fmt.Errorf("status=%d body=%s", response.StatusCode, body)
			}
			results <- err
		}()
	}
	pending.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Error(err)
		}
	}
	if successes.Load() != 16 {
		t.Fatalf("accepted requests replayed or lost: %d", successes.Load())
	}
}

func TestAIProxyCooldownRetryHints(t *testing.T) {
	var calls atomic.Int64
	client, url, _ := proxyFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(429)
		io.WriteString(w, `{"error":{"code":"rate_limit_exceeded","message":"private-provider-error"}}`)
	}, nil, false)
	for _, status := range []int{429, 503} {
		response, err := proxyRequest(t.Context(), client, url, "limited")
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(response.Body)
		response.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		retry, err := strconv.Atoi(response.Header.Get("Retry-After"))
		if response.StatusCode != status || err != nil || retry < 28 || retry > 31 {
			t.Fatalf("status=%d retry=%q", response.StatusCode, response.Header.Get("Retry-After"))
		}
		if strings.Contains(string(body), "private-provider-error") {
			t.Fatal("provider error leaked")
		}
	}
	if calls.Load() != 2 {
		t.Fatalf("cooling keys were retried: calls=%d", calls.Load())
	}
}

func TestAIProxyClientNotReadingReleasesCapacity(t *testing.T) {
	for _, useHTTP2 := range []bool{false, true} {
		t.Run(fmt.Sprintf("http2=%v", useHTTP2), func(t *testing.T) {
			var active atomic.Int64
			client, url, _ := proxyFixture(t, func(w http.ResponseWriter, r *http.Request) {
				active.Add(1)
				defer active.Add(-1)
				body, _ := io.ReadAll(r.Body)
				w.Header().Set("Content-Type", "text/event-stream")
				if !strings.Contains(string(body), "blocked reader") {
					io.WriteString(w, "data: [DONE]\n\n")
					return
				}
				payload := []byte("data: " + strings.Repeat("x", 32<<10) + "\n\n")
				for {
					select {
					case <-r.Context().Done():
						return
					default:
					}
					if _, err := w.Write(payload); err != nil {
						return
					}
					w.(http.Flusher).Flush()
				}
			}, func(s *service.Service) {
				s.Config.AIIdleTimeout = 500 * time.Millisecond
				s.Config.AIMaxConcurrentRequests = 1
			}, useHTTP2)
			blocked, err := proxyRequest(t.Context(), client, url, "blocked reader")
			if err != nil {
				t.Fatal(err)
			}
			defer blocked.Body.Close()
			if blocked.StatusCode != 200 {
				t.Fatalf("status=%d", blocked.StatusCode)
			}
			// Do not read or close the body: the downstream write deadline must
			// release the stream while the client connection is still present.
			waitProxyCondition(t, func() bool { return active.Load() == 0 })
			waitProxyCondition(t, func() bool {
				response, err := proxyRequest(t.Context(), client, url, "after timeout")
				if err != nil {
					t.Fatal(err)
				}
				body, err := io.ReadAll(response.Body)
				response.Body.Close()
				if response.StatusCode == 429 {
					return false
				}
				if response.StatusCode != 200 || err != nil || !strings.Contains(string(body), "[DONE]") {
					t.Fatalf("slot not recovered: status=%d err=%v body=%s", response.StatusCode, err, body)
				}
				return true
			})
		})
	}
}
