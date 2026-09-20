package badger

import (
	"context"

	"github.com/aknEvrnky/pgway/internal/ports"
	badgerdb "github.com/dgraph-io/badger/v4"
)

type storagePinger struct {
	db *badgerdb.DB
}

// NewStoragePinger returns a readiness Ping over an open Badger DB.
func NewStoragePinger(db *badgerdb.DB) ports.StoragePinger {
	return &storagePinger{db: db}
}

func (p *storagePinger) Ping(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return p.db.View(func(txn *badgerdb.Txn) error {
		return nil
	})
}
