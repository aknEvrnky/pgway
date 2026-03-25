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

func (r *PoolRepository) GetAll(ctx context.Context) ([]*domain.Pool, error) {
	var pools []*domain.Pool

	err := r.db.View(func(txn *badgerdb.Txn) error {
		opts := badgerdb.DefaultIteratorOptions
		opts.Prefix = []byte(poolPrefix)

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			err := it.Item().Value(func(val []byte) error {
				pool, err := r.unmarshal(val)
				if err != nil {
					return err
				}

				pools = append(pools, pool)
				return nil
			})

			if err != nil {
				return err
			}
		}
		return nil
	})

	return pools, err
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
	err := r.db.Update(func(txn *badgerdb.Txn) error {
		return txn.Delete(poolKey(id))
	})

	if errors.Is(err, badgerdb.ErrKeyNotFound) {
		return fmt.Errorf("pool %q not found", id)
	}

	return err
}
