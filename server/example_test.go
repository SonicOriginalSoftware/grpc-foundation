package server_test

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"

	"git.sonicoriginal.software/logger"

	"git.sonicoriginal.software/grpc-foundation/otel"
	"git.sonicoriginal.software/grpc-foundation/server"
)

const cleanupTimeout = 5 * time.Second

// registerExampleService stands in for the caller's own service registration.
func registerExampleService(s grpc.ServiceRegistrar) {
	s.RegisterService(&grpc.ServiceDesc{
		ServiceName: "example.ExampleService",
		HandlerType: (*any)(nil),
		Methods:     []grpc.MethodDesc{{MethodName: "Create"}},
	}, struct{}{})
}

// Example shows a service starting up and shutting down. It has no Output
// comment, so it is compiled but never run — its job is to keep this sequence
// type-checked against the real API.
func Example() {
	// Registered first so it runs last, after the teardown below has flushed.
	// Returning rather than calling os.Exit directly is what lets the defers run
	// at all.
	exitCode := 0
	defer func() { os.Exit(exitCode) }()

	// The process context. Everything the shutdown needs is built on this one,
	// which is why the signal cancels a child of it rather than this.
	ctx := context.Background()

	serveCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverName := server.Name("example")

	log, flush, err := otel.Init(ctx, serverName, server.Version())
	if err != nil {
		slog.Default().Error("Failed to initialize telemetry", slog.Any("error", err))
		exitCode = 1
		return
	}

	ctx = logger.ContextWithLogger(ctx, log)

	// New installs an interceptor that puts this logger into every request
	// context, so it has to be built after the logger is complete.
	srv := server.New(log)

	// Deferred before anything else can fail, so every path out of here stops
	// the server and exports what it logged on the way.
	defer server.HandleGracefulShutdown(ctx, log, srv, flush, cleanupTimeout)

	registerExampleService(srv)

	lis, err := server.Listen()
	if err != nil {
		log.Error("Failed to create listener", slog.Any("error", err))
		exitCode = 1
		return
	}

	log = log.With(slog.String("address", lis.Addr().String()))

	// Serve blocks, and a deferred teardown cannot run while it does, so it goes
	// to a goroutine and the select below decides when this returns.
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(lis) }()

	log.Info("gRPC server listening")

	select {
	case err := <-serveErr:
		if err != nil {
			log.Error("Failed to serve", slog.Any("error", err))
			exitCode = 1
		}
	case <-serveCtx.Done():
	}
}
