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
	balancerStorageVersion = "v1"
	balancerKind           = "LoadBalancer"
	balancerPrefix         = "balancers:"
)

type BalancerRepository struct {
	db *badgerdb.DB
}

func NewBalancerRepository(db *badgerdb.DB) *BalancerRepository {
	return &BalancerRepository{db: db}
}

func balancerKey(id string) []byte {
	return []byte(balancerPrefix + id)
}

func (r *BalancerRepository) marshal(lb *domain.LoadBalancer) ([]byte, error) {
	return json.Marshal(StoredResource[domain.LoadBalancer]{
		StorageVersion: balancerStorageVersion,
		Kind:           balancerKind,
		Spec:           *lb,
	})
}

func (r *BalancerRepository) unmarshal(data []byte) (*domain.LoadBalancer, error) {
	stored, err := unmarshal[domain.LoadBalancer](data)
	if err != nil {
		return nil, err
	}
	return &stored.Spec, nil
}

func (r *BalancerRepository) List(ctx context.Context, params domain.ListParams) (domain.ListResult[domain.LoadBalancer], error) {
	var result domain.ListResult[domain.LoadBalancer]
	err := r.db.View(func(txn *badgerdb.Txn) error {
		var err error
		result, err = listWithCursor(txn, balancerPrefix, params, r.unmarshal)
		return err
	})
	return result, err
}

func (r *BalancerRepository) Find(ctx context.Context, id string) (*domain.LoadBalancer, error) {
	var lb *domain.LoadBalancer

	err := r.db.View(func(txn *badgerdb.Txn) error {
		item, err := txn.Get(balancerKey(id))
		if errors.Is(err, badgerdb.ErrKeyNotFound) {
			return fmt.Errorf("balancer %q not found", id)
		}
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			lb, err = r.unmarshal(val)
			return err
		})
	})

	if err != nil {
		return nil, err
	}

	return lb, nil
}

func (r *BalancerRepository) Save(ctx context.Context, lb *domain.LoadBalancer) error {
	data, err := r.marshal(lb)
	if err != nil {
		return fmt.Errorf("marshal balancer %q: %w", lb.Id, err)
	}

	return r.db.Update(func(txn *badgerdb.Txn) error {
		return txn.Set(balancerKey(lb.Id), data)
	})
}

func (r *BalancerRepository) Delete(ctx context.Context, id string) error {
	return r.db.Update(func(txn *badgerdb.Txn) error {
		_, err := txn.Get(balancerKey(id))
		if errors.Is(err, badgerdb.ErrKeyNotFound) {
			return fmt.Errorf("balancer %q not found", id)
		}
		if err != nil {
			return err
		}
		return txn.Delete(balancerKey(id))
	})
}
