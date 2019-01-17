package scope

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestScopeGo(t *testing.T) {
	s := newScope(t)
	call := newCall(nil)
	s.Go(call.f)

	if err := call.wait(time.Second); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !call.called() {
		t.Fatal("expected function to be called")
	}

	if err := closeScope(s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScopeDefer(t *testing.T) {
	s := newScope(t)
	call := newCall(nil)
	s.Defer(call.f)

	if err := closeScope(s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := call.wait(time.Second); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !call.called() {
		t.Fatal("expected function to be called")
	}
}

func TestScopeStart(t *testing.T) {
	t.Run("short-run-ok", func(t *testing.T) {
		start := newCall(nil)
		stop := newCall(nil)

		s := newScope(t)
		s.Start(Service{
			Start: start.f,
			Stop:  stop.f,
		})

		if err := start.wait(time.Second); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !start.called() {
			t.Fatal("expected start function to be called")
		}

		if err := closeScope(s); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := stop.wait(time.Second); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stop.called() {
			t.Fatal("expected stop function to be called")
		}
	})

	t.Run("short-run-fail", func(t *testing.T) {
		start := newCall(func(context.Context) error { return errors.New("error") })
		stop := newCall(nil)

		s := newScope(t)
		s.onError = func(error) {}
		s.Start(Service{
			Start: start.f,
			Stop:  stop.f,
		})

		if err := start.wait(time.Second); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !start.called() {
			t.Fatal("expected start function to be called")
		}

		if err := closeScope(s); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if stop.called() {
			t.Fatal("expected stop function not to be called")
		}
	})

	t.Run("long-run", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		start := newCall(func(context.Context) error {
			<-ctx.Done()
			return nil
		})
		stop := newCall(func(context.Context) error {
			cancel()
			return nil
		})

		s := newScope(t)
		s.Start(Service{
			Start: start.f,
			Stop:  stop.f,
		})

		if err := closeScope(s); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := start.wait(time.Second); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !start.called() {
			t.Fatal("expected start function to be called")
		}

		if err := stop.wait(time.Second); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stop.called() {
			t.Fatal("expected stop function to be called")
		}
	})
}

func newScope(t *testing.T) *Scope {
	return New(WithErrorHandler(func(err error) { t.Fatalf("unexpected error: %v", err) }))
}

func closeScope(s *Scope) error {
	closed := make(chan error, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go func() { closed <- s.Close() }()

	select {
	case err := <-closed:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

type call struct {
	f    Func
	flag uint64
	done chan struct{}
}

func newCall(f Func) *call {
	c := &call{done: make(chan struct{})}
	c.f = func(ctx context.Context) error {
		defer close(c.done)
		defer atomic.StoreUint64(&c.flag, 1)
		if f != nil {
			return f(ctx)
		}
		return nil
	}
	return c
}

func (c *call) called() bool {
	return atomic.LoadUint64(&c.flag) != 0
}

func (c *call) wait(d time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	select {
	case <-c.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
