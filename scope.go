package scope

import (
	"context"
	"sync"
	"sync/atomic"
)

// Func represents the function type the scope is able to call.
type Func func(context.Context) error

// Service holds the start and stop function of a specific service or
// server. Normally start is a blocking function which returns when
// Stop will be called.
type Service struct {
	Start Func
	Stop  Func
}

// Scope provides a way to run several functions concurrently and register
// clean-up functions which are run when the scope is closed.
type Scope struct {
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	onError func(error)
	mtx     sync.Mutex
	tasks   []*task
}

// New creates a new scope with the given options.
func New(o ...Option) *Scope {
	opts := defaultOptions()
	for _, apply := range o {
		apply(&opts)
	}

	ctx, cancel := context.WithCancel(opts.ctx)
	return &Scope{
		ctx:     ctx,
		cancel:  cancel,
		onError: opts.errorHandler,
	}
}

// Ctx returns the scope's context. The context is derived from the
// configured base context (see WithContext) and is cancelled when
// the scope is closed.
func (s *Scope) Ctx() context.Context {
	return s.ctx
}

// Go runs the given function in a new Goroutine. If the function
// returns an error, it will be reported by the registered error
// handler (see WithErrorHandler).
func (s *Scope) Go(f Func) {
	s.Start(Service{Start: f})
}

// Defer registers a function which will be called when the scope
// is closed. All deferred functions are called in reverse order
// of registration to mimic the `defer` behaviour.
func (s *Scope) Defer(f Func) {
	s.Start(Service{
		Start: func(context.Context) error { return nil },
		Stop:  f,
	})
}

// Start tries to run the given service. The service's Start function will
// be called in a new Goroutine. The optional Stop function is called
// when the scope will be closed. However, if the Start function returns
// an error before the scope is closed, the error handler will be called
// and the Stop function will not be invoked.
func (s *Scope) Start(svc Service) {
	t := &task{stop: svc.Stop}

	s.mtx.Lock()
	s.tasks = append(s.tasks, t)
	s.mtx.Unlock()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		if err := svc.Start(s.ctx); err == nil {
			t.state.set(succeeded)
		} else {
			t.state.set(failed)
			s.onError(err)
		}
	}()
}

// Close closes the scope and runs all deferred functions. It waits
// until all functions have completed.
func (s *Scope) Close() error {
	s.mtx.Lock()
	tasks := s.tasks
	s.mtx.Unlock()

	defer s.cancel()

	var errs errorlist
	for i := len(tasks); i > 0; {
		i--

		// If the start function failed we don't
		// want to call the deferred function.
		if t := tasks[i]; t.stop != nil && !t.state.is(failed) {
			errs.append(t.stop(s.ctx))
		}
	}
	s.wg.Wait()
	return errs.err()
}

type state uint64

const (
	running state = iota
	failed
	succeeded
)

func (s *state) set(v state)     { atomic.StoreUint64((*uint64)(s), uint64(v)) }
func (s *state) is(v state) bool { return state(atomic.LoadUint64((*uint64)(s))) == v }

type task struct {
	stop  Func
	state state
}
