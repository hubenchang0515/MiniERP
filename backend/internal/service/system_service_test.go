package service

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSystemResourcesUnavailableDisk(t *testing.T) {
	s := newService(t)
	s.Config.DataDir = filepath.Join(t.TempDir(), "missing-directory")
	result := s.collectSystemResources(context.Background())
	if result.Disk.Error == "" {
		t.Fatal("missing directory must not appear as an empty disk")
	}
	if result.SampledAt.IsZero() {
		t.Fatal("missing sample timestamp")
	}
}

func TestSystemResourcesCancellation(t *testing.T) {
	s := newService(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result := s.collectSystemResources(ctx)
	if result.CPU.Error == "" {
		t.Fatal("cancelled sample must not appear as zero CPU usage")
	}
}

func TestResourceRequestsReadCacheWithoutWaiting(t *testing.T) {
	s := &Service{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started, release := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	want := SystemResources{CPU: CPUUsage{Percent: 17}, SampledAt: time.Now()}
	collect := func(ctx context.Context) SystemResources {
		calls.Add(1)
		close(started)
		select {
		case <-release:
		case <-ctx.Done():
		}
		return want
	}
	s.resources.start(ctx, time.Hour, collect)
	s.resources.start(ctx, time.Hour, collect)
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("sampler did not start")
	}
	var wg sync.WaitGroup
	for range 64 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.SystemResources()
			var serviceErr *Error
			if !errors.As(err, &serviceErr) || serviceErr.Status != 503 {
				t.Errorf("expected warming response, got %v", err)
			}
		}()
	}
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("requests waited for the blocked sampler")
	}
	close(release)
	deadline := time.Now().Add(2 * time.Second)
	for s.resources.latest.Load() == nil && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	for range 100 {
		got, err := s.SystemResources()
		if err != nil || got != want {
			t.Fatalf("unexpected cached result: %+v %v", got, err)
		}
		got.CPU.Percent = 99 // Callers receive a copy of the snapshot.
	}
	if calls.Load() != 1 {
		t.Fatalf("requests or repeated starts triggered %d samples", calls.Load())
	}
}

func TestResourceMonitorRefreshAndShutdown(t *testing.T) {
	var monitor resourceMonitor
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	refreshStarted, done := make(chan struct{}), make(chan struct{})
	first := SystemResources{CPU: CPUUsage{Percent: 17}, SampledAt: time.Now()}
	calls := 0
	go func() {
		defer close(done)
		monitor.run(ctx, time.Millisecond, func(ctx context.Context) SystemResources {
			calls++
			if calls <= 2 {
				result := first
				result.CPU.Percent += float64(calls - 1)
				return result
			}
			close(refreshStarted)
			<-ctx.Done()
			return SystemResources{CPU: CPUUsage{Error: "cancelled"}}
		})
	}()
	select {
	case <-refreshStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("periodic refresh did not run")
	}
	if got := monitor.latest.Load(); got == nil || got.CPU.Percent != 18 {
		t.Fatalf("previous snapshot unavailable during refresh: %+v", got)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("sampler did not stop on cancellation")
	}
	if got := monitor.latest.Load(); got == nil || got.CPU.Percent != 18 {
		t.Fatalf("cancelled collection replaced the cache: %+v", got)
	}
}
