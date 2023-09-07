package conc

// copy from Milvus: milvus/pkg/util/conc/options.go

import (
	"github.com/panjf2000/ants/v2"
	"time"
)

type poolOption struct {
	// pre-allocs workers
	preAlloc bool
	// block or not when conc is full
	nonBlocking bool
	// duration to cleanup worker goroutine
	expiryDuration time.Duration
	// disable purge worker
	disablePurge bool
	// whether conceal panic when job has panic
	concealPanic bool
	// panicHandler when task panics
	panicHandler func(any)

	// preHandler function executed before actual method executed
	preHandler func()
}

func (opt *poolOption) antsOptions() []ants.Option {
	var result []ants.Option
	result = append(result, ants.WithPreAlloc(opt.preAlloc))
	result = append(result, ants.WithNonblocking(opt.nonBlocking))
	result = append(result, ants.WithDisablePurge(opt.disablePurge))
	// ants recovers panic by default
	// however the error is not returned
	result = append(result, ants.WithPanicHandler(func(v any) {
		if !opt.concealPanic {
			panic(v)
		}
	}))
	if opt.panicHandler != nil {
		result = append(result, ants.WithPanicHandler(opt.panicHandler))
	}
	if opt.expiryDuration > 0 {
		result = append(result, ants.WithExpiryDuration(opt.expiryDuration))
	}

	return result
}

// PoolOption options function to setup conc.
type PoolOption func(opt *poolOption)

func defaultPoolOption() *poolOption {
	return &poolOption{
		preAlloc:       false,
		nonBlocking:    false,
		expiryDuration: 0,
		disablePurge:   false,
		concealPanic:   false,
	}
}

func WithPreAlloc(v bool) PoolOption {
	return func(opt *poolOption) {
		opt.preAlloc = v
	}
}

func WithNonBlocking(v bool) PoolOption {
	return func(opt *poolOption) {
		opt.nonBlocking = v
	}
}

func WithDisablePurge(v bool) PoolOption {
	return func(opt *poolOption) {
		opt.disablePurge = v
	}
}

func WithExpiryDuration(d time.Duration) PoolOption {
	return func(opt *poolOption) {
		opt.expiryDuration = d
	}
}

func WithConcealPanic(v bool) PoolOption {
	return func(opt *poolOption) {
		opt.concealPanic = v
	}
}

func WithPreHandler(fn func()) PoolOption {
	return func(opt *poolOption) {
		opt.preHandler = fn
	}
}
