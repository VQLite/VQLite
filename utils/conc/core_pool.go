package conc

import (
	"runtime"
	"sync"
)

// copy from Milvus: milvus/internal/querynodev2/segments/pool.go

import (
	"go.uber.org/atomic"
)

var (
	// Use separate pool for search/query
	// and other operations (insert/delete/statistics/etc.)
	// since in concurrent situation, there operation may block each other in high payload

	sqp     atomic.Pointer[Pool[any]]
	sqOnce  sync.Once
	dp      atomic.Pointer[Pool[any]]
	dynOnce sync.Once
)

// initSQPool initialize
func initSQPool() {
	sqOnce.Do(func() {
		pool := NewPool[any](
			runtime.GOMAXPROCS(0)*2,
			WithPreAlloc(true),
			WithDisablePurge(true),
		)

		WarmupPool(pool, runtime.LockOSThread)

		sqp.Store(pool)
	})
}

// initDynamicPool initialize
func initDynamicPool() {
	dynOnce.Do(func() {
		pool := NewPool[any](
			runtime.GOMAXPROCS(0),
			WithPreAlloc(false),
			WithDisablePurge(false),
			WithPreHandler(runtime.LockOSThread), // lock os thread for cgo thread disposal
		)

		dp.Store(pool)
	})
}

// GetSQPool returns the singleton conc instance.
func GetSQPool() *Pool[any] {
	initSQPool()
	return sqp.Load()
}

// GetDynamicPool returns the singleton conc instance.
func GetDynamicPool() *Pool[any] {
	initDynamicPool()
	return dp.Load()
}
