package metrics

import (
	"runtime"
)

// RuntimeSample is a point-in-time process resource sample.
type RuntimeSample struct {
	AllocBytes uint64
	SysBytes   uint64
	Goroutines int
	CPUCount   int
}

// RuntimeSampler abstracts runtime/memory sampling for tests.
type RuntimeSampler interface {
	Sample() RuntimeSample
}

// DefaultRuntimeSampler reads from the Go runtime.
type DefaultRuntimeSampler struct{}

// Sample implements RuntimeSampler.
func (DefaultRuntimeSampler) Sample() RuntimeSample {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return RuntimeSample{
		AllocBytes: ms.Alloc,
		SysBytes:   ms.Sys,
		Goroutines: runtime.NumGoroutine(),
		CPUCount:   runtime.NumCPU(),
	}
}
