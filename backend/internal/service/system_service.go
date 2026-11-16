package service

import (
	"context"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
)

type CPUUsage struct {
	Percent float64 `json:"percent"`
	Error   string  `json:"error,omitempty"`
}

type MemoryUsage struct {
	Total     uint64  `json:"total"`
	Used      uint64  `json:"used"`
	Available uint64  `json:"available"`
	Percent   float64 `json:"percent"`
	Error     string  `json:"error,omitempty"`
}

type DiskUsage struct {
	Path    string  `json:"path"`
	Total   uint64  `json:"total"`
	Used    uint64  `json:"used"`
	Free    uint64  `json:"free"`
	Percent float64 `json:"percent"`
	Error   string  `json:"error,omitempty"`
}

type SystemResources struct {
	CPU       CPUUsage    `json:"cpu"`
	Memory    MemoryUsage `json:"memory"`
	Disk      DiskUsage   `json:"disk"`
	SampledAt time.Time   `json:"sampledAt"`
}

type resourceMonitor struct {
	once   sync.Once
	latest atomic.Pointer[SystemResources]
}

// StartResourceMonitor starts one sampler for the service lifetime. Requests
// only read snapshots, so client count does not affect the sampling workload.
func (s *Service) StartResourceMonitor(ctx context.Context) {
	s.resources.start(ctx, time.Second, s.collectSystemResources)
}

func (m *resourceMonitor) start(ctx context.Context, interval time.Duration, collect func(context.Context) SystemResources) {
	m.once.Do(func() {
		go m.run(ctx, interval, collect)
	})
}

func (m *resourceMonitor) run(ctx context.Context, interval time.Duration, collect func(context.Context) SystemResources) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		result := collect(ctx)
		if ctx.Err() != nil {
			return
		}
		m.latest.Store(&result)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// SystemResources returns an immutable snapshot without collecting OS counters.
func (s *Service) SystemResources() (SystemResources, error) {
	result := s.resources.latest.Load()
	if result == nil {
		return SystemResources{}, Fail(503, "resources_not_ready", "资源数据正在采集，请稍后重试")
	}
	return *result, nil
}

// Each metric reports its own failure so one unavailable OS counter does not
// hide the others. This collection runs only in the background sampler.
func (s *Service) collectSystemResources(ctx context.Context) SystemResources {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	var result SystemResources
	percent, err := cpu.PercentWithContext(ctx, 500*time.Millisecond, false)
	if err != nil {
		result.CPU.Error = "CPU 使用率采集失败"
	} else if len(percent) == 0 {
		result.CPU.Error = "CPU 使用率暂不可用"
	} else {
		result.CPU.Percent = percent[0]
	}
	memory, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		result.Memory.Error = "内存使用量采集失败"
	} else {
		result.Memory = MemoryUsage{Total: memory.Total, Used: memory.Used, Available: memory.Available, Percent: memory.UsedPercent}
	}
	result.Disk.Path, err = filepath.Abs(s.Config.DataDir)
	if err != nil {
		result.Disk.Error = "数据目录路径无效"
	} else {
		usage, err := disk.UsageWithContext(ctx, result.Disk.Path)
		if err != nil {
			result.Disk.Error = "磁盘使用量采集失败"
		} else {
			result.Disk.Total, result.Disk.Used, result.Disk.Free, result.Disk.Percent = usage.Total, usage.Used, usage.Free, usage.UsedPercent
		}
	}
	result.SampledAt = time.Now()
	return result
}
