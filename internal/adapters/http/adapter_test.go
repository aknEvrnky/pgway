package http

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/application/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeAPI is a minimal ports.Application for adapter tests. Only EntryPoints
// carries behaviour; the data plane methods are never exercised here.
type fakeAPI struct {
	mu  sync.Mutex
	eps []*domain.Entrypoint
}

func (f *fakeAPI) setEntrypoints(eps ...*domain.Entrypoint) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.eps = eps
}

func (f *fakeAPI) EntryPoints(_ context.Context) ([]*domain.Entrypoint, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.eps, nil
}

func (f *fakeAPI) Bootstrap(_ context.Context) error { return nil }

func (f *fakeAPI) ExecuteFlow(_ context.Context, _ string, _ *http.Request) (*domain.Proxy, string, error) {
	return nil, "", fmt.Errorf("not implemented")
}

func (f *fakeAPI) Release(_ context.Context, _ string, _ domain.BalancerResult) error { return nil }

func (f *fakeAPI) HandleEvent(_ context.Context, _ event.ChangeEvent) error { return nil }

func testEntrypoint(id string, port uint16) *domain.Entrypoint {
	return &domain.Entrypoint{
		Id:       id,
		Title:    "test",
		Protocol: "http",
		Host:     "127.0.0.1",
		Port:     port,
		FlowId:   "flow-1",
	}
}

func freePort(t *testing.T) uint16 {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := lis.Addr().(*net.TCPAddr).Port
	require.NoError(t, lis.Close())
	return uint16(port)
}

func waitListening(t *testing.T, addr string) {
	t.Helper()
	require.Eventually(t, func() bool {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	}, 2*time.Second, 20*time.Millisecond, "expected %s to accept connections", addr)
}

func waitNotListening(t *testing.T, addr string) {
	t.Helper()
	require.Eventually(t, func() bool {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err != nil {
			return true
		}
		_ = conn.Close()
		return false
	}, 2*time.Second, 20*time.Millisecond, "expected %s to refuse connections", addr)
}

func savedEvent(id string) event.ChangeEvent {
	return event.ChangeEvent{ID: id, ResourceType: event.ResourceTypeEntrypoint, ChangeKind: event.ChangeKindSaved}
}

func deletedEvent(id string) event.ChangeEvent {
	return event.ChangeEvent{ID: id, ResourceType: event.ResourceTypeEntrypoint, ChangeKind: event.ChangeKindDeleted}
}

// --- HandleEvent -----------------------------------------------------------

func TestAdapter_HandleEvent_SavedStartsNewServer(t *testing.T) {
	api := &fakeAPI{}
	adapter, err := NewHttpAdapter(context.Background(), api, nil)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = adapter.Run(ctx) }()

	// a brand-new entrypoint appears after startup
	ep := testEntrypoint("ep-new", freePort(t))
	api.setEntrypoints(ep)

	require.NoError(t, adapter.HandleEvent(ctx, savedEvent(ep.Id)))
	waitListening(t, ep.ListenAddr())
}

func TestAdapter_HandleEvent_SavedRestartsExistingServer(t *testing.T) {
	oldPort, newPort := freePort(t), freePort(t)
	ep := testEntrypoint("ep-1", oldPort)

	api := &fakeAPI{}
	api.setEntrypoints(ep)

	adapter, err := NewHttpAdapter(context.Background(), api, nil)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = adapter.Run(ctx) }()
	waitListening(t, ep.ListenAddr())

	// entrypoint moves to a new port
	updated := testEntrypoint("ep-1", newPort)
	api.setEntrypoints(updated)

	require.NoError(t, adapter.HandleEvent(ctx, savedEvent("ep-1")))
	waitNotListening(t, ep.ListenAddr())
	waitListening(t, updated.ListenAddr())
}

func TestAdapter_HandleEvent_DeletedStopsServer(t *testing.T) {
	ep := testEntrypoint("ep-1", freePort(t))

	api := &fakeAPI{}
	api.setEntrypoints(ep)

	adapter, err := NewHttpAdapter(context.Background(), api, nil)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runDone := make(chan struct{})
	go func() {
		_ = adapter.Run(ctx)
		close(runDone)
	}()
	waitListening(t, ep.ListenAddr())

	api.setEntrypoints() // deleted upstream
	require.NoError(t, adapter.HandleEvent(ctx, deletedEvent("ep-1")))
	waitNotListening(t, ep.ListenAddr())

	// Run must survive losing its last server: the adapter's lifetime is
	// bound to ctx, not to the number of live servers.
	assert.Never(t, func() bool {
		select {
		case <-runDone:
			return true
		default:
			return false
		}
	}, 300*time.Millisecond, 50*time.Millisecond, "Run must not return when all servers are gone")
}

func TestAdapter_HandleEvent_IgnoresOtherResourceTypes(t *testing.T) {
	api := &fakeAPI{}
	adapter, err := NewHttpAdapter(context.Background(), api, nil)
	require.NoError(t, err)

	e := event.ChangeEvent{ID: "p-1", ResourceType: event.ResourceTypeProxy, ChangeKind: event.ChangeKindSaved}
	require.NoError(t, adapter.HandleEvent(context.Background(), e))

	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	assert.Empty(t, adapter.servers)
}

func TestAdapter_HandleEvent_SavedUnknownEntrypointReturnsError(t *testing.T) {
	api := &fakeAPI{}
	adapter, err := NewHttpAdapter(context.Background(), api, nil)
	require.NoError(t, err)

	err = adapter.HandleEvent(context.Background(), savedEvent("ghost"))
	assert.ErrorContains(t, err, "not found")
}

// --- Run / Shutdown --------------------------------------------------------

func TestAdapter_Run_ReturnsOnContextCancel(t *testing.T) {
	ep := testEntrypoint("ep-1", freePort(t))

	api := &fakeAPI{}
	api.setEntrypoints(ep)

	adapter, err := NewHttpAdapter(context.Background(), api, nil)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	runDone := make(chan struct{})
	go func() {
		_ = adapter.Run(ctx)
		close(runDone)
	}()
	waitListening(t, ep.ListenAddr())

	cancel()
	select {
	case <-runDone:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after ctx cancel")
	}
}

func TestAdapter_Shutdown_StopsAllServers(t *testing.T) {
	ep1 := testEntrypoint("ep-1", freePort(t))
	ep2 := testEntrypoint("ep-2", freePort(t))

	api := &fakeAPI{}
	api.setEntrypoints(ep1, ep2)

	adapter, err := NewHttpAdapter(context.Background(), api, nil)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = adapter.Run(ctx) }()
	waitListening(t, ep1.ListenAddr())
	waitListening(t, ep2.ListenAddr())

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	require.NoError(t, adapter.Shutdown(shutdownCtx))

	waitNotListening(t, ep1.ListenAddr())
	waitNotListening(t, ep2.ListenAddr())
}
