//revive:disable:package-comments
package client

import (
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	"git.sonicoriginal.software/grpc-foundation/config"
)

// New creates a gRPC client connection with standardized options.
//
// Standard options include:
//   - Insecure credentials (for internal service mesh communication)
//   - OpenTelemetry trace propagation (automatic distributed tracing)
//   - Keepalive pings, read from GRPC_KEEPALIVE_TIME and GRPC_KEEPALIVE_TIMEOUT,
//     sent whether or not an RPC is active so a peer that went silent is
//     noticed on a held connection and not only on the next call
//
// Additional options can be passed to customize the connection:
//
//	conn, err := client.New("service:50051", nil, nil,
//	    grpc.WithBlock(),
//	    grpc.WithTimeout(5*time.Second),
//	)
//
// For external services requiring TLS, pass custom credentials:
//
//	tlsConfig := &tls.Config{...}
//	creds := credentials.NewTLS(tlsConfig)
//	conn, err := client.New("external.service.com:443", nil, nil,
//	    grpc.WithTransportCredentials(creds), // Overrides insecure default
//	)
func New(
	address string,
	unaryInterceptors []grpc.UnaryClientInterceptor,
	streamInterceptors []grpc.StreamClientInterceptor,
	opts ...grpc.DialOption,
) (*grpc.ClientConn, error) {
	standardOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                config.KeepAliveTime(),
			Timeout:             config.KeepAliveTimeout(),
			PermitWithoutStream: true,
		}),
		grpc.WithChainUnaryInterceptor(unaryInterceptors...),
		grpc.WithChainStreamInterceptor(streamInterceptors...),
	}

	standardOpts = append(standardOpts, opts...)

	return grpc.NewClient(address, standardOpts...)
}
