package server

import (
	"log/slog"
	"time"

	"git.sonicoriginal.software/logger"

	"git.sonicoriginal.software/grpc-foundation/config"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
)

// New creates a gRPC server with production-grade defaults.
// The provided logger is used for recovery logging.
// Additional server options can be provided to customize behavior.
func New(log *slog.Logger, opts ...grpc.ServerOption) *grpc.Server {
	if log == nil {
		log = logger.NewNullLogger()
	}

	recoveryHandler := recovery.WithRecoveryHandler(makeRecoveryHandler(log))

	standardOpts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			recovery.UnaryServerInterceptor(recoveryHandler),
			makeLoggerUnaryInterceptor(log),
		),
		grpc.ChainStreamInterceptor(
			recovery.StreamServerInterceptor(recoveryHandler),
			makeLoggerStreamInterceptor(log),
		),
		// OpenTelemetry tracing via stats handler
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		// Connection management
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     config.DurationOrDefault(EnvMaxConnectionIdle, DefaultMaxConnectionIdle),
			MaxConnectionAge:      config.DurationOrDefault(EnvMaxConnectionAge, DefaultMaxConnectionAge),
			MaxConnectionAgeGrace: config.DurationOrDefault(EnvMaxConnectionAgeGrace, DefaultMaxConnectionAgeGrace),
			Time:                  config.KeepAliveTime(),
			Timeout:               config.KeepAliveTimeout(),
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             1 * time.Minute,
			PermitWithoutStream: true,
		}),
		// Message size limits
		grpc.MaxRecvMsgSize(config.IntOrDefault(EnvMaxRecvMsgSize, DefaultMaxRecvMsgSize)),
		grpc.MaxSendMsgSize(config.IntOrDefault(EnvMaxSendMsgSize, DefaultMaxSendMsgSize)),
	}

	// Append custom options (can override standard options if needed)
	allOpts := append(standardOpts, opts...)
	return grpc.NewServer(allOpts...)
}

// makeRecoveryHandler creates a panic recovery handler that logs and returns Internal error.
func makeRecoveryHandler(log *slog.Logger) func(p any) error {
	return func(p any) error {
		log.Error("Recovered from panic", slog.Any("panic", p))
		return status.Errorf(codes.Internal, "internal server error")
	}
}
