package server

import (
	"context"
	"log/slog"
	"time"

	"git.sonicoriginal.software/grpc-foundation/lifecycle"
)

// GracefulStopper stops a gRPC server once its in-flight requests finish
type GracefulStopper interface {
	GracefulStop()
}

// HandleGracefulShutdown stops the gRPC server and then flushes the telemetry
// providers, both against one deadline built from cleanupTimeout. Whatever
// stopping the server spends, the flush does not get.
//
// The flush runs second because the requests that drain during GracefulStop
// produce the spans and logs it exports.
//
// Callers defer it, so it runs on every path out of main: a listener that fails
// to bind flushes the error explaining that failure the same way a signal
// flushes the requests that drained.
//
// ctx is the process context. Passing the context that was cancelled to trigger
// the shutdown leaves nothing for the deadline to be built on and the flush
// exports nothing.
func HandleGracefulShutdown(
	ctx context.Context,
	log *slog.Logger,
	grpcServer GracefulStopper,
	flush lifecycle.ShutdownFunc,
	cleanupTimeout time.Duration,
) {
	shutdownLog := log.With(slog.String("component", "shutdown-handler"))
	shutdownLog.Info("Initiating graceful shutdown")

	shutdownCtx, cancel := context.WithTimeout(ctx, cleanupTimeout)
	defer cancel()

	shutdownLog.Info("Stopping gRPC server")

	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		shutdownLog.Info("gRPC server stopped")
	case <-shutdownCtx.Done():
		shutdownLog.Warn("Timed out waiting for in-flight requests",
			slog.Duration("timeout", cleanupTimeout))
	}

	shutdownLog.Info("Flushing telemetry")

	if err := flush(shutdownCtx); err != nil {
		shutdownLog.Warn("Telemetry flush incomplete", slog.Any("error", err))
		return
	}

	shutdownLog.Info("Telemetry flushed")
}
