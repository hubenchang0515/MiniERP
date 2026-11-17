package handler

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

var errAIIdleTimeout = errors.New("AI proxy idle timeout")

// A stream may run for hours as long as it keeps making progress. The mutex
// also prevents a stale timer callback from canceling a recently active stream.
type proxyActivity struct {
	mu     sync.Mutex
	timer  *time.Timer
	last   time.Time
	idle   time.Duration
	cancel context.CancelCauseFunc
	closed bool
}

func watchProxyActivity(parent context.Context, idle time.Duration) (context.Context, *proxyActivity) {
	ctx, cancel := context.WithCancelCause(parent)
	a := &proxyActivity{last: time.Now(), idle: idle, cancel: cancel}
	a.timer = time.AfterFunc(idle, a.expire)
	return ctx, a
}

func (a *proxyActivity) expire() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed {
		return
	}
	if remaining := a.idle - time.Since(a.last); remaining > 0 {
		a.timer.Reset(remaining)
		return
	}
	a.closed = true
	a.cancel(errAIIdleTimeout)
}

func (a *proxyActivity) touch() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.closed {
		a.last = time.Now()
		a.timer.Reset(a.idle)
	}
}

func (a *proxyActivity) close() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.closed = true
	a.timer.Stop()
	a.cancel(context.Canceled)
}

// A client that stops reading must not hold a concurrency slot indefinitely.
// Unwrap preserves ResponseController support for HTTP/1 and HTTP/2.
type proxyStreamWriter struct {
	http.ResponseWriter
	idle     time.Duration
	timedOut *atomic.Bool
}

func (w *proxyStreamWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (w *proxyStreamWriter) Write(p []byte) (int, error) {
	controller := http.NewResponseController(w.ResponseWriter)
	_ = controller.SetWriteDeadline(time.Now().Add(w.idle))
	defer controller.SetWriteDeadline(time.Time{})
	n, err := w.ResponseWriter.Write(p)
	w.observeError(err)
	return n, err
}

func (w *proxyStreamWriter) FlushError() error {
	controller := http.NewResponseController(w.ResponseWriter)
	_ = controller.SetWriteDeadline(time.Now().Add(w.idle))
	defer controller.SetWriteDeadline(time.Time{})
	err := controller.Flush()
	w.observeError(err)
	return err
}

func (w *proxyStreamWriter) observeError(err error) {
	var networkError net.Error
	if errors.As(err, &networkError) && networkError.Timeout() {
		w.timedOut.Store(true)
	}
}

type observedProxyBody struct {
	io.ReadCloser
	readErr *error
	touch   func()
}

func (b *observedProxyBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if n > 0 {
		b.touch()
	}
	if err != nil && err != io.EOF {
		*b.readErr = err
	}
	return n, err
}

func proxyStopReason(client, upstream context.Context, readErr error) string {
	if errors.Is(client.Err(), context.Canceled) {
		return "client_canceled"
	}
	if errors.Is(client.Err(), context.DeadlineExceeded) {
		return "client_timeout"
	}
	if errors.Is(context.Cause(upstream), errAIIdleTimeout) {
		return "gateway_idle_timeout"
	}
	if errors.Is(upstream.Err(), context.DeadlineExceeded) {
		return "gateway_timeout"
	}
	if errors.Is(readErr, io.ErrUnexpectedEOF) {
		return "upstream_truncated"
	}
	if readErr != nil {
		return "upstream_read_failed"
	}
	return "response_write_failed"
}

func logProxyAbort(client, upstream context.Context, readErr *error, writeTimedOut *atomic.Bool, path, requestID string, started time.Time) {
	if recovered := recover(); recovered != nil {
		if recovered == http.ErrAbortHandler {
			reason := proxyStopReason(client, upstream, *readErr)
			if writeTimedOut.Load() {
				reason = "downstream_write_timeout"
			}
			log.Printf("ai proxy stopped request_id=%s stage=body reason=%s path=%s elapsed=%s", requestID, reason, path, time.Since(started).Round(time.Millisecond))
		}
		panic(recovered)
	}
}
