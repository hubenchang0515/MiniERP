package handler

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

func TestProxyStopReasons(t *testing.T) {
	client, cancelClient := context.WithCancel(context.Background())
	cancelClient()
	timedOut, cancelTimeout := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancelTimeout()
	for _, test := range []struct {
		name             string
		client, upstream context.Context
		err              error
		want             string
	}{
		{"cancel", client, client, context.Canceled, "client_canceled"},
		{"client timeout", timedOut, timedOut, context.DeadlineExceeded, "client_timeout"},
		{"gateway timeout", context.Background(), timedOut, context.DeadlineExceeded, "gateway_timeout"},
		{"truncated", context.Background(), context.Background(), io.ErrUnexpectedEOF, "upstream_truncated"},
		{"read failure", context.Background(), context.Background(), io.ErrClosedPipe, "upstream_read_failed"},
		{"write failure", context.Background(), context.Background(), nil, "response_write_failed"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := proxyStopReason(test.client, test.upstream, test.err); got != test.want {
				t.Fatalf("reason=%s want=%s", got, test.want)
			}
		})
	}
}

func TestProxyActivityTimeoutAndCancellation(t *testing.T) {
	ctx, activity := watchProxyActivity(t.Context(), 100*time.Millisecond)
	defer activity.close()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	activeUntil := time.After(350 * time.Millisecond)
active:
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("active stream canceled: %v", context.Cause(ctx))
		case <-ticker.C:
			activity.touch()
		case <-activeUntil:
			break active
		}
	}
	select {
	case <-ctx.Done():
		if !errors.Is(context.Cause(ctx), errAIIdleTimeout) {
			t.Fatalf("cause=%v", context.Cause(ctx))
		}
	case <-time.After(time.Second):
		t.Fatal("idle stream did not time out")
	}
	parent, cancel := context.WithCancel(t.Context())
	child, watcher := watchProxyActivity(parent, time.Hour)
	defer watcher.close()
	cancel()
	if !errors.Is(context.Cause(child), context.Canceled) {
		t.Fatal("client cancellation not propagated")
	}
	if got := proxyStopReason(t.Context(), ctx, context.Canceled); got != "gateway_idle_timeout" {
		t.Fatalf("idle reason=%s", got)
	}
}
