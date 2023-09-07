package conc

// copy from Milvus: milvus/pkg/util/conc/future.go
type future interface {
	wait()
	OK() bool
	Err() error
}

// Future is a result type of async-await style.
// It contains the result (or error) of an async task.
// Trying to obtain the result (or error) blocks until the async task completes.
type Future[T any] struct {
	ch    chan struct{}
	value T
	err   error
}

func newFuture[T any]() *Future[T] {
	return &Future[T]{
		ch: make(chan struct{}),
	}
}

func (future *Future[T]) wait() {
	<-future.ch
}

// Return the result and error of the async task.
func (future *Future[T]) Await() (T, error) {
	future.wait()
	return future.value, future.err
}

// Return the result of the async task,
// nil if no result or error occurred.
func (future *Future[T]) Value() T {
	<-future.ch

	return future.value
}

// False if error occurred,
// true otherwise.
func (future *Future[T]) OK() bool {
	<-future.ch

	return future.err == nil
}

// Return the error of the async task,
// nil if no error.
func (future *Future[T]) Err() error {
	<-future.ch

	return future.err
}

// Return a read-only channel,
// which will be closed if the async task completes.
// Use this if you need to wait the async task in a select statement.
func (future *Future[T]) Inner() <-chan struct{} {
	return future.ch
}

// Go spawns a goroutine to execute fn,
// returns a future that contains the result of fn.
// NOTE: use Pool if you need limited goroutines.
func Go[T any](fn func() (T, error)) *Future[T] {
	future := newFuture[T]()
	go func() {
		future.value, future.err = fn()
		close(future.ch)
	}()
	return future
}

// Await for multiple futures,
// Return nil if no future returns error,
// or return the first error in these futures.
func AwaitAll[T future](futures ...T) error {
	for i := range futures {
		if !futures[i].OK() {
			return futures[i].Err()
		}
	}

	return nil
}
