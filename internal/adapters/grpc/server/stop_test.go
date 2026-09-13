package server

import (
	"context"
	"net"
	"testing"
	"time"

	controlplanev1 "github.com/aknEvrnky/pgway/gen/pgway/controlplane/v1"
	"github.com/aknEvrnky/pgway/internal/adapters/pubsub/memory"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func TestGracefulStopCompletesWithOpenWatch(t *testing.T) {
	lis := bufconn.Listen(1 << 20)
	bus := memory.NewPubSub(10)

	shutdown, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := grpc.NewServer()
	RegisterChange(s, NewChangeServer(bus, shutdown))
	go s.Serve(lis) //nolint:errcheck

	conn, err := grpc.NewClient("passthrough:///buf",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	client := controlplanev1.NewChangeServiceClient(conn)
	stream, err := client.Watch(context.Background(), &controlplanev1.WatchRequest{})
	require.NoError(t, err)
	_ = stream

	time.Sleep(50 * time.Millisecond)

	// Cancel process shutdown — Watch must exit so GracefulStop can finish.
	cancel()

	done := make(chan struct{})
	go func() {
		GracefulStopWithTimeout(s, time.Second)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("GracefulStop did not complete with an open Watch stream after shutdown cancel")
	}
}
