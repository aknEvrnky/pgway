package badger_test

import (
	"context"
	"testing"

	badgerrepo "github.com/aknEvrnky/pgway/internal/adapters/repository/badger"
	"github.com/dgraph-io/badger/v4"
	"github.com/stretchr/testify/require"
)

func TestStoragePinger_Ping(t *testing.T) {
	dir := t.TempDir()
	db, err := badger.Open(badger.DefaultOptions(dir).WithLogger(nil))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	p := badgerrepo.NewStoragePinger(db)
	require.NoError(t, p.Ping(context.Background()))

	require.NoError(t, db.Close())
	require.Error(t, p.Ping(context.Background()))
}
