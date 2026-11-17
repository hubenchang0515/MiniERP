package service

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type AIProviderTransport struct {
	Service    *Service
	ResourceID uint
	Body       []byte
	Inference  bool
	Timing     *AIProxyTiming
}

type AIProxyTiming struct {
	UpstreamHeaders time.Duration
	Attempts        int
}

var aiHTTPTransport = func() *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConnsPerHost = 32
	transport.MaxIdleConns = 128
	return transport
}()

type providerResponseBody struct {
	io.ReadCloser
	once   sync.Once
	finish func() error
	err    error
}

func (b *providerResponseBody) Close() error {
	b.once.Do(func() {
		b.err = b.ReadCloser.Close()
		if err := b.finish(); b.err == nil {
			b.err = err
		}
	})
	return b.err
}

func (t *AIProviderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	excluded := map[uint]bool{}
	var last *http.Response
	for len(excluded) < 100 {
		if err := req.Context().Err(); err != nil {
			return nil, err
		}
		lease, err := t.Service.AcquireProviderKey(t.ResourceID, excluded)
		if err != nil {
			if e, ok := err.(*Error); ok && e.Code == "no_available_key" && last != nil {
				return last, nil
			}
			return nil, err
		}
		excluded[lease.Key.ID] = true
		out := req.Clone(req.Context())
		out.Header = req.Header.Clone()
		out.Header.Set("Authorization", "Bearer "+lease.Secret)
		if t.Body != nil {
			out.Body = io.NopCloser(bytes.NewReader(t.Body))
			out.ContentLength = int64(len(t.Body))
		}
		started := time.Now()
		response, err := aiHTTPTransport.RoundTrip(out)
		if t.Timing != nil {
			t.Timing.Attempts++
			t.Timing.UpstreamHeaders += time.Since(started)
		}
		if err != nil {
			state := ""
			var retryAt *time.Time
			if lease.Probe {
				state = lease.Key.State
				next := time.Now().Add(5 * time.Minute)
				retryAt = &next
			}
			if finishErr := lease.Finish(state, retryAt, false); finishErr != nil {
				return nil, finishErr
			}
			// A network error can be ambiguous; do not replay a possibly accepted request.
			return nil, err
		}
		if response.StatusCode < 400 {
			success := t.Inference && response.StatusCode >= 200 && response.StatusCode < 300
			response.Body = &providerResponseBody{ReadCloser: response.Body, finish: func() error { return lease.Finish("", nil, success) }}
			return response, nil
		}
		// Error bodies are inspected locally and never returned to the caller.
		started = time.Now()
		body, _ := io.ReadAll(io.LimitReader(response.Body, 64<<10))
		if t.Timing != nil {
			t.Timing.UpstreamHeaders += time.Since(started)
		}
		response.Body.Close()
		response.Body = io.NopCloser(bytes.NewReader(nil))
		state, retryAt := providerFailure(response.StatusCode, response.Header, body, time.Now())
		// Only forward a validated cooldown, never arbitrary provider headers.
		response.Header.Del("Retry-After")
		if retryAt != nil {
			seconds := int64(time.Until(*retryAt).Seconds()) + 1
			if seconds < 1 {
				seconds = 1
			}
			response.Header.Set("Retry-After", strconv.FormatInt(seconds, 10))
		}
		retryable := state != ""
		if state == "" && lease.Probe {
			state = lease.Key.State
			next := time.Now().Add(5 * time.Minute)
			retryAt = &next
		}
		if err := lease.Finish(state, retryAt, false); err != nil {
			return nil, err
		}
		last = response
		if !retryable {
			return response, nil
		}
	}
	return last, nil
}

func providerFailure(status int, header http.Header, body []byte, now time.Time) (string, *time.Time) {
	var payload struct {
		Error struct {
			Code string `json:"code"`
			Type string `json:"type"`
		} `json:"error"`
	}
	_ = json.Unmarshal(body, &payload)
	code, kind := strings.ToLower(payload.Error.Code), strings.ToLower(payload.Error.Type)
	state := ""
	quotaCodes := map[string]bool{"insufficient_quota": true, "quota_exceeded": true, "credit_balance_exhausted": true, "organization_usage_limit_exceeded": true, "billing_hard_limit_reached": true, "insufficient_balance": true, "balance_not_enough": true}
	if status == 402 || quotaCodes[code] || quotaCodes[kind] {
		state = ProviderKeyQuota
	} else if status == 401 || code == "invalid_api_key" || kind == "authentication_error" {
		return ProviderKeyInvalid, nil
	} else if status == 429 {
		state = ProviderKeyRateLimit
	}
	if state == "" {
		return "", nil
	}
	next := now.Add(5 * time.Minute)
	// Quota/billing errors must not use short request-rate reset headers.
	if value := header.Get("Retry-After"); value != "" {
		if seconds, err := strconv.ParseFloat(value, 64); err == nil && seconds >= 0 && seconds <= 366*24*3600 {
			next = now.Add(time.Duration(seconds * float64(time.Second)))
		} else if date, err := http.ParseTime(value); err == nil {
			next = date
		}
	} else if state == ProviderKeyRateLimit {
		var delay time.Duration
		for _, name := range []string{"X-Ratelimit-Reset-Requests", "X-Ratelimit-Reset-Tokens"} {
			if value, err := time.ParseDuration(header.Get(name)); err == nil && value > delay {
				delay = value
			}
		}
		if delay > 0 {
			next = now.Add(delay)
		}
	}
	if !next.After(now) {
		next = now.Add(time.Second)
	}
	return state, &next
}
