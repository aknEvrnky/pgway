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
	poolStorageVersion = "v1"
	poolKind           = "Pool"
	poolPrefix         = "pools:"
)

type PoolRepository struct {
	db *badgerdb.DB
}

func NewPoolRepository(db *badgerdb.DB) *PoolRepository {
	return &PoolRepository{db: db}
}

func poolKey(id string) []byte {
	return []byte(poolPrefix + id)
}

func (r *PoolRepository) marshal(pool *domain.Pool) ([]byte, error) {
	return json.Marshal(StoredResource[domain.Pool]{
		StorageVersion: poolStorageVersion,
		Kind:           poolKind,
		Spec:           *pool,
	})
}

func (r *PoolRepository) unmarshal(data []byte) (*domain.Pool, error) {
	stored, err := unmarshal[domain.Pool](data)
	if err != nil {
		return nil, err
	}
	return &stored.Spec, nil
}

func (r *PoolRepository) List(ctx context.Context, params domain.ListParams, filter domain.PoolFilter) (domain.ListResult[domain.Pool], error) {
	predicate := buildPoolPredicate(filter)
	var result domain.ListResult[domain.Pool]
	err := r.db.View(func(txn *badgerdb.Txn) error {
		var err error
		result, err = listWithCursor(txn, poolPrefix, params, r.unmarshal, predicate)
		return err
	})
	return result, err
}

func buildPoolPredicate(f domain.PoolFilter) func(*domain.Pool) bool {
	if f.Search == "" && f.Type == "" {
		return nil
	}
	return func(p *domain.Pool) bool {
		if f.Search != "" && !containsFold(p.Id, f.Search) && !containsFold(p.Title, f.Search) {
			return false
		}
		if f.Type != "" && string(p.Type) != f.Type {
			return false
		}
		return true
	}
}

func (r *PoolRepository) Find(ctx context.Context, id string) (*domain.Pool, error) {
	var pool *domain.Pool

	err := r.db.View(func(txn *badgerdb.Txn) error {
		item, err := txn.Get(poolKey(id))
		if errors.Is(err, badgerdb.ErrKeyNotFound) {
			return fmt.Errorf("pool %q not found", id)
		}

		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			pool, err = r.unmarshal(val)
			return err
		})
	})

	if err != nil {
		return nil, err
	}

	return pool, nil
}

func (r *PoolRepository) Save(ctx context.Context, pool *domain.Pool) error {
	data, err := r.marshal(pool)
	if err != nil {
		return fmt.Errorf("marshal pool %q: %w", pool.Id, err)
	}

	return r.db.Update(func(txn *badgerdb.Txn) error {
		return txn.Set(poolKey(pool.Id), data)
	})
}

func (r *PoolRepository) Delete(ctx context.Context, id string) error {
	return r.db.Update(func(txn *badgerdb.Txn) error {
		_, err := txn.Get(poolKey(id))
		if errors.Is(err, badgerdb.ErrKeyNotFound) {
			return fmt.Errorf("pool %q not found", id)
		}
		if err != nil {
			return err
		}
		return txn.Delete(poolKey(id))
	})
}
