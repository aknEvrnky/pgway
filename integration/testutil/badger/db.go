package badgerutil

import (
	"os"
	"testing"

	badger "github.com/dgraph-io/badger/v4"

	badgerrepo "github.com/aknEvrnky/pgway/internal/adapters/repository/badger"
	"github.com/aknEvrnky/pgway/internal/ports"
)

// TestStore holds an open BadgerDB instance and all 6 repository adapters.
type TestStore struct {
	DB      *badger.DB
	Proxies ports.ProxyRepositoryPort
	Pools   ports.PoolRepositoryPort
	LBs     ports.LoadBalancerRepositoryPort
	Routers ports.RouterRepositoryPort
	Flows   ports.FlowRepositoryPort
	EPs     ports.EntryPointRepositoryPort
}

// NewBadgerDB opens a BadgerDB in a temp directory. The database is closed
// and the directory is removed when the test finishes.
func NewBadgerDB(t *testing.T) *badger.DB {
	t.Helper()

	dir, err := os.MkdirTemp("", "pgway-badger-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	db, err := badger.Open(badger.DefaultOptions(dir).WithLogger(nil))
	if err != nil {
		os.RemoveAll(dir)
		t.Fatalf("open badger: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
		os.RemoveAll(dir)
	})

	return db
}

// NewBadgerStore opens a BadgerDB and initialises all 6 repository adapters.
func NewBadgerStore(t *testing.T) *TestStore {
	t.Helper()

	db := NewBadgerDB(t)

	return &TestStore{
		DB:      db,
		Proxies: badgerrepo.NewProxyRepository(db),
		Pools:   badgerrepo.NewPoolRepository(db),
		LBs:     badgerrepo.NewBalancerRepository(db),
		Routers: badgerrepo.NewRouterRepository(db),
		Flows:   badgerrepo.NewFlowRepository(db),
		EPs:     badgerrepo.NewEntrypointRepository(db),
	}
}
