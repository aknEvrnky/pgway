package testutil

import (
	"context"
	"net"
	"testing"

	controlplanev1 "github.com/aknEvrnky/pgway/gen/pgway/controlplane/v1"
	grpcserver "github.com/aknEvrnky/pgway/internal/adapters/grpc/server"
	"github.com/aknEvrnky/pgway/internal/ports"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

// NewTestGrpcServer starts an in-memory gRPC server with all 6 control plane
// services registered. It returns a client connection backed by bufconn.
// Both the server and connection are cleaned up when the test finishes.
func NewTestGrpcServer(t *testing.T, cp ports.ControlPlane) *grpc.ClientConn {
	t.Helper()

	lis := bufconn.Listen(bufSize)
	t.Cleanup(func() { lis.Close() }) // registered first → runs last (LIFO)

	s := grpc.NewServer()
	cpServer := grpcserver.NewControlPlaneServer(cp)
	controlplanev1.RegisterProxyServiceServer(s, cpServer)
	controlplanev1.RegisterPoolServiceServer(s, cpServer)
	controlplanev1.RegisterRouterServiceServer(s, cpServer)
	controlplanev1.RegisterBalancerServiceServer(s, cpServer)
	controlplanev1.RegisterEntrypointServiceServer(s, cpServer)
	controlplanev1.RegisterFlowServiceServer(s, cpServer)

	go s.Serve(lis) //nolint:errcheck

	t.Cleanup(s.Stop) // registered after lis.Close → runs first (LIFO)

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("create grpc client: %v", err)
	}

	t.Cleanup(func() { conn.Close() })

	return conn
}
