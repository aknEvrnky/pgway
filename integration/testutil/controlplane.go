package testutil

import (
	"testing"

	badgerutil "github.com/aknEvrnky/pgway/integration/testutil/badger"
	"github.com/aknEvrnky/pgway/internal/application/controlplane"
	"github.com/aknEvrnky/pgway/internal/ports"
)

// NewSvcWithPublisher is a helper that sets up a fresh BadgerDB store and a ControlPlane service.
func NewSvcWithPublisher(t *testing.T) (*controlplane.Service, *SpyPublisher) {
	t.Helper()
	publisher := &SpyPublisher{}

	return NewSvc(t, publisher), publisher
}

func NewSvc(t *testing.T, publisher ports.EventPublisherPort) *controlplane.Service {
	t.Helper()
	store := badgerutil.NewBadgerStore(t)

	return controlplane.NewService(
		store.Proxies,
		store.Pools,
		store.LBs,
		store.Routers,
		store.Flows,
		store.EPs,
		publisher,
	)
}
