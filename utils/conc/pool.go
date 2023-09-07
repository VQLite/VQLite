package conc

// copy from Milvus: milvus/pkg/util/conc/conc.go

import (
	"fmt"
	ants "github.com/panjf2000/ants/v2"
	"runtime"
	"sync"
)

// A goroutine conc
type Pool[T any] struct {
	inner *ants.Pool
	opt   *poolOption
}

// NewPool returns a goroutine pool.
// cap: the number of workers.
// This panic if provide any invalid option.
func NewPool[T any](cap int, opts ...PoolOption) *Pool[T] {
	opt := defaultPoolOption()
	for _, o := range opts {
		o(opt)
	}

	pool, err := ants.NewPool(cap, opt.antsOptions()...)
	if err != nil {
		panic(err)
	}

	return &Pool[T]{
		inner: pool,
		opt:   opt,
	}
}

// NewDefaultPool returns a conc with cap of the number of logical CPU,
// and pre-alloced goroutines.
func NewDefaultPool[T any]() *Pool[T] {
	return NewPool[T](runtime.GOMAXPROCS(0), WithPreAlloc(true))
}

// Submit a task into the conc,
// executes it asynchronously.
// This will block if the conc has finite workers and no idle worker.
// NOTE: As now golang doesn't support the member method being generic, we use Future[any]
func (pool *Pool[T]) Submit(method func() (T, error)) *Future[T] {
	future := newFuture[T]()
	err := pool.inner.Submit(func() {
		defer close(future.ch)
		defer func() {
			if x := recover(); x != nil {
				future.err = fmt.Errorf("panicked with error: %v", x)
				panic(x) // throw panic out
			}
		}()
		// execute pre handler
		if pool.opt.preHandler != nil {
			pool.opt.preHandler()
		}
		res, err := method()
		if err != nil {
			future.err = err
		} else {
			future.value = res
		}
	})
	if err != nil {
		future.err = err
		close(future.ch)
	}

	return future
}

// The number of workers
func (pool *Pool[T]) Cap() int {
	return pool.inner.Cap()
}

// The number of running workers
func (pool *Pool[T]) Running() int {
	return pool.inner.Running()
}

// Free returns the number of free workers
func (pool *Pool[T]) Free() int {
	return pool.inner.Free()
}

func (pool *Pool[T]) Release() {
	pool.inner.Release()
}

// WarmupPool do warm up logic for each goroutine in conc
func WarmupPool[T any](pool *Pool[T], warmup func()) {
	cap := pool.Cap()
	ch := make(chan struct{})
	wg := sync.WaitGroup{}
	wg.Add(cap)
	for i := 0; i < cap; i++ {
		pool.Submit(func() (T, error) {
			warmup()
			wg.Done()
			<-ch
			return *new(T), nil
		})
	}
	wg.Wait()
	close(ch)
}
