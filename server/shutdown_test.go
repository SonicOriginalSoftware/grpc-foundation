package server

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"
)

// stopperStub records that the handler stopped the server. Closing the channel
// gives a test something to wait on instead of a sleep or the handler's return.
type stopperStub struct {
	stopped chan struct{}
}

func newStopperStub() *stopperStub {
	return &stopperStub{stopped: make(chan struct{})}
}

func (s *stopperStub) GracefulStop() { close(s.stopped) }

// blockingStopperStub stands in for a server whose in-flight requests never
// finish, so the handler has to stop waiting on its own.
type blockingStopperStub struct {
	release chan struct{}
}

func (s *blockingStopperStub) GracefulStop() { <-s.release }

// flushStub records the context the handler flushed with, so a test can assert
// on what the providers would have been given.
type flushStub struct {
	called bool
	err    error
	ctxErr error
}

func (f *flushStub) Flush(ctx context.Context) error {
	f.called = true
	f.ctxErr = ctx.Err()

	return f.err
}

// await fails the test if signal does not fire. The ceiling turns a handler
// that never gets there into a failure rather than a hang.
func await(t *testing.T, signal <-chan struct{}, message string) {
	t.Helper()

	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		t.Fatal(message)
	}
}

func TestHandleGracefulShutdown(t *testing.T) {
	t.Run("stops the server then flushes with a live context", func(t *testing.T) {
		stopper := newStopperStub()
		flush := &flushStub{}

		HandleGracefulShutdown(
			t.Context(), slog.New(slog.DiscardHandler), stopper, flush.Flush, 5*time.Second,
		)

		await(t, stopper.stopped, "server was not stopped")

		if !flush.called {
			t.Fatal("telemetry was not flushed")
		}

		if flush.ctxErr != nil {
			t.Fatalf("flush context should still be live, got %v", flush.ctxErr)
		}
	})

	t.Run("flushes when in-flight requests do not finish in time", func(t *testing.T) {
		stopper := &blockingStopperStub{release: make(chan struct{})}
		defer close(stopper.release)

		flush := &flushStub{}

		returned := make(chan struct{})
		go func() {
			HandleGracefulShutdown(
				t.Context(), slog.New(slog.DiscardHandler), stopper, flush.Flush, time.Millisecond,
			)
			close(returned)
		}()

		await(t, returned, "handler did not stop waiting for the server")

		if !flush.called {
			t.Fatal("telemetry was not flushed after the server timed out")
		}

		if !errors.Is(flush.ctxErr, context.DeadlineExceeded) {
			t.Fatalf("flush context should have been spent, got %v", flush.ctxErr)
		}
	})

	t.Run("returns when the flush fails", func(t *testing.T) {
		stopper := newStopperStub()
		flush := &flushStub{err: errors.New("collector unreachable")}

		HandleGracefulShutdown(
			t.Context(), slog.New(slog.DiscardHandler), stopper, flush.Flush, 5*time.Second,
		)

		await(t, stopper.stopped, "server was not stopped")

		if !flush.called {
			t.Fatal("telemetry was not flushed")
		}
	})
}
