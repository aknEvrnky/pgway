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
	routerStorageVersion = "v1"
	routerKind           = "Router"
	routerPrefix         = "routers:"
)

type RouterRepository struct {
	db *badgerdb.DB
}

func NewRouterRepository(db *badgerdb.DB) *RouterRepository {
	return &RouterRepository{db: db}
}

func routerKey(id string) []byte {
	return []byte(routerPrefix + id)
}

func (r *RouterRepository) marshal(router *domain.Router) ([]byte, error) {
	return json.Marshal(StoredResource[domain.Router]{
		StorageVersion: routerStorageVersion,
		Kind:           routerKind,
		Spec:           *router,
	})
}

func (r *RouterRepository) unmarshal(data []byte) (*domain.Router, error) {
	stored, err := unmarshal[domain.Router](data)
	if err != nil {
		return nil, err
	}
	return &stored.Spec, nil
}

func (r *RouterRepository) GetAll(ctx context.Context) ([]*domain.Router, error) {
	var routers []*domain.Router

	err := r.db.View(func(txn *badgerdb.Txn) error {
		opts := badgerdb.DefaultIteratorOptions
		opts.Prefix = []byte(routerPrefix)

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			err := it.Item().Value(func(val []byte) error {
				router, err := r.unmarshal(val)
				if err != nil {
					return err
				}
				routers = append(routers, router)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})

	return routers, err
}

func (r *RouterRepository) Find(ctx context.Context, id string) (*domain.Router, error) {
	var router *domain.Router

	err := r.db.View(func(txn *badgerdb.Txn) error {
		item, err := txn.Get(routerKey(id))
		if errors.Is(err, badgerdb.ErrKeyNotFound) {
			return fmt.Errorf("router %q not found", id)
		}
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			router, err = r.unmarshal(val)
			return err
		})
	})

	if err != nil {
		return nil, err
	}

	return router, nil
}

func (r *RouterRepository) Save(ctx context.Context, router *domain.Router) error {
	data, err := r.marshal(router)
	if err != nil {
		return fmt.Errorf("marshal router %q: %w", router.Id, err)
	}

	return r.db.Update(func(txn *badgerdb.Txn) error {
		return txn.Set(routerKey(router.Id), data)
	})
}

func (r *RouterRepository) Delete(ctx context.Context, id string) error {
	return r.db.Update(func(txn *badgerdb.Txn) error {
		_, err := txn.Get(routerKey(id))
		if errors.Is(err, badgerdb.ErrKeyNotFound) {
			return fmt.Errorf("router %q not found", id)
		}
		if err != nil {
			return err
		}
		return txn.Delete(routerKey(id))
	})
}
