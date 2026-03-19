package badger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	badgerdb "github.com/dgraph-io/badger/v4"
)

const (
	proxyStorageVersion = "v1"
	proxyKind           = "Proxy"
	proxyPrefix         = "proxies:"
)

type ProxyRepository struct {
	db *badgerdb.DB
}

func NewProxyRepository(db *badgerdb.DB) *ProxyRepository {
	return &ProxyRepository{db: db}
}

func proxyKey(id string) []byte {
	return []byte(proxyPrefix + id)
}

func (r *ProxyRepository) marshal(proxy *domain.Proxy) ([]byte, error) {
	return json.Marshal(StoredResource[domain.Proxy]{
		StorageVersion: proxyStorageVersion,
		Kind:           proxyKind,
		UpdatedAt:      time.Now(),
		Spec:           *proxy,
	})
}

func (r *ProxyRepository) unmarshal(data []byte) (*domain.Proxy, error) {
	stored, err := unmarshal[domain.Proxy](data)
	if err != nil {
		return nil, err
	}
	return &stored.Spec, nil
}

func (r *ProxyRepository) GetAll(ctx context.Context) ([]*domain.Proxy, error) {
	var proxies []*domain.Proxy

	err := r.db.View(func(txn *badgerdb.Txn) error {
		opts := badgerdb.DefaultIteratorOptions
		opts.Prefix = []byte(proxyPrefix)

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			err := it.Item().Value(func(val []byte) error {
				proxy, err := r.unmarshal(val)
				if err != nil {
					return err
				}

				proxies = append(proxies, proxy)
				return nil
			})

			if err != nil {
				return err
			}
		}
		return nil
	})

	return proxies, err
}

func (r *ProxyRepository) Find(ctx context.Context, id string) (*domain.Proxy, error) {
	var proxy *domain.Proxy

	err := r.db.View(func(txn *badgerdb.Txn) error {
		item, err := txn.Get(proxyKey(id))
		if !errors.Is(err, badgerdb.ErrKeyNotFound) {
			return fmt.Errorf("proxy %q not found", id)
		}

		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			proxy, err = r.unmarshal(val)
			return err
		})
	})

	if err != nil {
		return nil, err
	}

	return proxy, nil
}

func (r *ProxyRepository) Save(ctx context.Context, proxy *domain.Proxy) error {
	data, err := r.marshal(proxy)
	if err != nil {
		return fmt.Errorf("marshall proxy %q: %w", proxy.Id, err)
	}

	return r.db.Update(func(txn *badgerdb.Txn) error {
		return txn.Set(proxyKey(proxy.Id), data)
	})
}

func (r *ProxyRepository) Delete(ctx context.Context, id string) error {
	err := r.db.Update(func(txn *badgerdb.Txn) error {
		return txn.Delete(proxyKey(id))
	})

	if errors.Is(err, badgerdb.ErrKeyNotFound) {
		return fmt.Errorf("proxy %q not found", id)
	}

	return err
}
