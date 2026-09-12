package server

import (
	"log/slog"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"git.sonicoriginal.software/grpc-foundation/config"
)

func TestNew(t *testing.T) {
	t.Run("creates server with default options", func(t *testing.T) {
		server := New(slog.Default())
		if server == nil {
			t.Fatal("expected server to be non-nil")
		}
		server.Stop()
	})

	t.Run("creates server with custom options", func(t *testing.T) {
		customOpt := grpc.MaxRecvMsgSize(2048)
		server := New(slog.Default(), customOpt)
		if server == nil {
			t.Fatal("expected server to be non-nil")
		}
		server.Stop()
	})

	t.Run("creates server with nil logger", func(t *testing.T) {
		server := New(nil)
		if server == nil {
			t.Fatal("expected server to be non-nil")
		}
		server.Stop()
	})

	t.Run("reads all connection settings from environment", func(t *testing.T) {
		t.Setenv(EnvMaxConnectionIdle, "1m")
		t.Setenv(EnvMaxConnectionAge, "2m")
		t.Setenv(EnvMaxConnectionAgeGrace, "3s")
		t.Setenv(config.EnvKeepAliveTime, "4m")
		t.Setenv(config.EnvKeepAliveTimeout, "5s")
		t.Setenv(EnvMaxRecvMsgSize, "1024")
		t.Setenv(EnvMaxSendMsgSize, "2048")

		server := New(slog.Default())
		if server == nil {
			t.Fatal("expected server to be non-nil")
		}
		server.Stop()
	})
}

func TestMakeRecoveryHandler(t *testing.T) {
	t.Run("returns Internal error on panic", func(t *testing.T) {
		recoveryFn := makeRecoveryHandler(slog.Default())
		err := recoveryFn("test panic value")

		st, ok := status.FromError(err)
		if !ok {
			t.Fatal("expected error to be a gRPC status")
		}
		if st.Code() != codes.Internal {
			t.Errorf("expected code %v, got %v", codes.Internal, st.Code())
		}
		if st.Message() != "internal server error" {
			t.Errorf("expected message %q, got %q", "internal server error", st.Message())
		}
	})
}
