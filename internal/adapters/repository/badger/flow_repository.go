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
	flowStorageVersion = "v1"
	flowKind           = "Flow"
	flowPrefix         = "flows:"
)

type FlowRepository struct {
	db *badgerdb.DB
}

func NewFlowRepository(db *badgerdb.DB) *FlowRepository {
	return &FlowRepository{db: db}
}

func flowKey(id string) []byte {
	return []byte(flowPrefix + id)
}

func (r *FlowRepository) marshal(flow *domain.Flow) ([]byte, error) {
	return json.Marshal(StoredResource[domain.Flow]{
		StorageVersion: flowStorageVersion,
		Kind:           flowKind,
		Spec:           *flow,
	})
}

func (r *FlowRepository) unmarshal(data []byte) (*domain.Flow, error) {
	stored, err := unmarshal[domain.Flow](data)
	if err != nil {
		return nil, err
	}
	return &stored.Spec, nil
}

func (r *FlowRepository) List(ctx context.Context, params domain.ListParams) (domain.ListResult[domain.Flow], error) {
	var result domain.ListResult[domain.Flow]
	err := r.db.View(func(txn *badgerdb.Txn) error {
		var err error
		result, err = listWithCursor(txn, flowPrefix, params, r.unmarshal)
		return err
	})
	return result, err
}

func (r *FlowRepository) Find(ctx context.Context, id string) (*domain.Flow, error) {
	var flow *domain.Flow

	err := r.db.View(func(txn *badgerdb.Txn) error {
		item, err := txn.Get(flowKey(id))
		if errors.Is(err, badgerdb.ErrKeyNotFound) {
			return fmt.Errorf("flow %q not found", id)
		}
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			flow, err = r.unmarshal(val)
			return err
		})
	})

	if err != nil {
		return nil, err
	}

	return flow, nil
}

func (r *FlowRepository) Save(ctx context.Context, flow *domain.Flow) error {
	data, err := r.marshal(flow)
	if err != nil {
		return fmt.Errorf("marshal flow %q: %w", flow.Id, err)
	}

	return r.db.Update(func(txn *badgerdb.Txn) error {
		return txn.Set(flowKey(flow.Id), data)
	})
}

func (r *FlowRepository) Delete(ctx context.Context, id string) error {
	return r.db.Update(func(txn *badgerdb.Txn) error {
		_, err := txn.Get(flowKey(id))
		if errors.Is(err, badgerdb.ErrKeyNotFound) {
			return fmt.Errorf("flow %q not found", id)
		}
		if err != nil {
			return err
		}
		return txn.Delete(flowKey(id))
	})
}
