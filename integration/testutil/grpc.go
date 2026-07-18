package testutil

import (
	"context"
	"net"
	"testing"
	"time"

	badgerutil "github.com/aknEvrnky/pgway/integration/testutil/badger"
	grpcserver "github.com/aknEvrnky/pgway/internal/adapters/grpc/server"
	"github.com/aknEvrnky/pgway/internal/adapters/pubsub/memory"
	"github.com/aknEvrnky/pgway/internal/application/auth"
	"github.com/aknEvrnky/pgway/internal/application/controlplane"
	"github.com/aknEvrnky/pgway/internal/ports"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

// NewTestGrpcServer starts an in-memory gRPC server with all 6 control plane
// services registered. It returns a client connection backed by bufconn.
// Both the server and connection are cleaned up when the test finishes.
func NewTestGrpcServer(t *testing.T, cp ports.ControlPlane, resolver ports.ProxyResolver) *grpc.ClientConn {
	t.Helper()

	lis := bufconn.Listen(bufSize)
	t.Cleanup(func() { lis.Close() }) // registered first → runs last (LIFO)

	s := grpc.NewServer()
	grpcserver.RegisterControlPlane(s, grpcserver.NewControlPlaneServer(cp, resolver))

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

// NewAuthTestServer starts the production gRPC server wiring (control plane +
// auth, interceptors enabled, via grpcserver.New) on an ephemeral localhost
// port. It returns the listen address and the auth service so tests can read
// the bootstrap token.
func NewAuthTestServer(t *testing.T) (string, *auth.Service) {
	t.Helper()

	store := badgerutil.NewBadgerStore(t)
	pubsub := memory.NewPubSub(10)
	cpService := controlplane.NewService(
		store.Proxies,
		store.Pools,
		store.LBs,
		store.Routers,
		store.Flows,
		store.EPs,
		pubsub,
	)
	authService := auth.NewService(store.Users, store.Tokens, time.Hour)
	authenticator := auth.NewAuthenticator(store.Users, store.Agents, store.Tokens)

	if err := authService.Bootstrap(context.Background()); err != nil {
		t.Fatalf("auth bootstrap: %v", err)
	}

	s := grpcserver.New(cpService, cpService, authService, authService, authenticator)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	go s.Serve(lis) //nolint:errcheck
	t.Cleanup(s.Stop)

	return lis.Addr().String(), authService
}
