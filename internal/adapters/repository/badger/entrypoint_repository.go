package badger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	badgerdb "github.com/dgraph-io/badger/v4"
)

const (
	entrypointStorageVersion = "v1"
	entrypointKind           = "Entrypoint"
	entrypointPrefix         = "entrypoints:"
)

type EntrypointRepository struct {
	db *badgerdb.DB
}

func NewEntrypointRepository(db *badgerdb.DB) *EntrypointRepository {
	return &EntrypointRepository{db: db}
}

func entrypointKey(id string) []byte {
	return []byte(entrypointPrefix + id)
}

func (r *EntrypointRepository) marshal(ep *domain.Entrypoint) ([]byte, error) {
	return json.Marshal(StoredResource[domain.Entrypoint]{
		StorageVersion: entrypointStorageVersion,
		Kind:           entrypointKind,
		Spec:           *ep,
	})
}

func (r *EntrypointRepository) unmarshal(data []byte) (*domain.Entrypoint, error) {
	stored, err := unmarshal[domain.Entrypoint](data)
	if err != nil {
		return nil, err
	}
	return &stored.Spec, nil
}

func (r *EntrypointRepository) List(ctx context.Context, params domain.ListParams) (domain.ListResult[domain.Entrypoint], error) {
	var result domain.ListResult[domain.Entrypoint]
	err := r.db.View(func(txn *badgerdb.Txn) error {
		var err error
		result, err = listWithCursor(txn, entrypointPrefix, params, r.unmarshal)
		return err
	})
	return result, err
}

func (r *EntrypointRepository) Find(ctx context.Context, id string) (*domain.Entrypoint, error) {
	var ep *domain.Entrypoint

	err := r.db.View(func(txn *badgerdb.Txn) error {
		item, err := txn.Get(entrypointKey(id))
		if errors.Is(err, badgerdb.ErrKeyNotFound) {
			return fmt.Errorf("entrypoint %q not found", id)
		}
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			ep, err = r.unmarshal(val)
			return err
		})
	})

	if err != nil {
		return nil, err
	}

	return ep, nil
}

func (r *EntrypointRepository) Save(ctx context.Context, ep *domain.Entrypoint) error {
	data, err := r.marshal(ep)
	if err != nil {
		return fmt.Errorf("marshal entrypoint %q: %w", ep.Id, err)
	}

	return r.db.Update(func(txn *badgerdb.Txn) error {
		return txn.Set(entrypointKey(ep.Id), data)
	})
}

func (r *EntrypointRepository) Delete(ctx context.Context, id string) error {
	return r.db.Update(func(txn *badgerdb.Txn) error {
		_, err := txn.Get(entrypointKey(id))
		if errors.Is(err, badgerdb.ErrKeyNotFound) {
			return fmt.Errorf("entrypoint %q not found", id)
		}
		if err != nil {
			return err
		}
		return txn.Delete(entrypointKey(id))
	})
}
