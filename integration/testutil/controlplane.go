package testutil

import (
	"testing"

	badgerutil "github.com/aknEvrnky/pgway/integration/testutil/badger"
	"github.com/aknEvrnky/pgway/internal/application/controlplane/api"
	"github.com/aknEvrnky/pgway/internal/ports"
)

// NewSvcWithPublisher is a helper that sets up a fresh BadgerDB store and a ControlPlane service.
func NewSvcWithPublisher(t *testing.T) (*api.Service, *SpyPublisher) {
	t.Helper()
	publisher := &SpyPublisher{}

	return NewSvc(t, publisher), publisher
}

func NewSvc(t *testing.T, publisher ports.EventPublisherPort) *api.Service {
	t.Helper()
	store := badgerutil.NewBadgerStore(t)

	return api.NewService(
		store.Proxies,
		store.Pools,
		store.LBs,
		store.Routers,
		store.Flows,
		store.EPs,
		publisher,
	)
}
