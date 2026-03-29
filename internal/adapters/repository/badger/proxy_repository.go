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

func (r *ProxyRepository) List(ctx context.Context, params domain.ListParams, filter domain.ProxyFilter) (domain.ListResult[domain.Proxy], error) {
	predicate := buildProxyPredicate(filter)
	var result domain.ListResult[domain.Proxy]
	err := r.db.View(func(txn *badgerdb.Txn) error {
		var err error
		result, err = listWithCursor(txn, proxyPrefix, params, r.unmarshal, predicate)
		return err
	})
	return result, err
}

func buildProxyPredicate(f domain.ProxyFilter) func(*domain.Proxy) bool {
	if f.Search == "" && f.Protocol == "" && len(f.Labels) == 0 {
		return nil
	}
	return func(p *domain.Proxy) bool {
		if f.Search != "" && !containsFold(p.Id, f.Search) && !containsFold(p.Host, f.Search) {
			return false
		}
		if f.Protocol != "" && string(p.Protocol) != f.Protocol {
			return false
		}
		if len(f.Labels) > 0 && !matchesLabels(p.Labels, f.Labels) {
			return false
		}
		return true
	}
}

func (r *ProxyRepository) Find(ctx context.Context, id string) (*domain.Proxy, error) {
	var proxy *domain.Proxy

	err := r.db.View(func(txn *badgerdb.Txn) error {
		item, err := txn.Get(proxyKey(id))
		if errors.Is(err, badgerdb.ErrKeyNotFound) {
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
	return r.db.Update(func(txn *badgerdb.Txn) error {
		_, err := txn.Get(proxyKey(id))
		if errors.Is(err, badgerdb.ErrKeyNotFound) {
			return fmt.Errorf("proxy %q not found", id)
		}
		if err != nil {
			return err
		}
		return txn.Delete(proxyKey(id))
	})
}

func (r *ProxyRepository) GetByIds(ctx context.Context, ids []string) ([]*domain.Proxy, error) {
	var proxies []*domain.Proxy

	err := r.db.View(func(txn *badgerdb.Txn) error {
		for _, id := range ids {
			item, err := txn.Get(proxyKey(id))
			if errors.Is(err, badgerdb.ErrKeyNotFound) {
				return fmt.Errorf("proxy %q not found", id)
			}
			if err != nil {
				return err
			}

			err = item.Value(func(val []byte) error {
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

func (r *ProxyRepository) FindByLabels(ctx context.Context, labels map[string]string) ([]*domain.Proxy, error) {
	result, err := r.List(ctx, domain.ListParams{}, domain.ProxyFilter{Labels: labels})
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}
