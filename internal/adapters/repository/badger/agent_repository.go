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
	agentStorageVersion = "v1"
	agentKind           = "Agent"
	agentPrefix         = "agents:"
)

type AgentRepository struct {
	db *badgerdb.DB
}

func NewAgentRepository(db *badgerdb.DB) *AgentRepository {
	return &AgentRepository{db: db}
}

func agentKey(id string) []byte {
	return []byte(agentPrefix + id)
}

func (r *AgentRepository) marshal(agent *domain.Agent) ([]byte, error) {
	return json.Marshal(StoredResource[domain.Agent]{
		StorageVersion: agentStorageVersion,
		Kind:           agentKind,
		Spec:           *agent,
	})
}

func (r *AgentRepository) unmarshal(data []byte) (*domain.Agent, error) {
	stored, err := unmarshal[domain.Agent](data)
	if err != nil {
		return nil, err
	}
	return &stored.Spec, nil
}

func (r *AgentRepository) List(ctx context.Context, params domain.ListParams, filter domain.AgentFilter) (domain.ListResult[domain.Agent], error) {
	predicate := buildAgentPredicate(filter)
	var result domain.ListResult[domain.Agent]
	err := r.db.View(func(txn *badgerdb.Txn) error {
		var err error
		result, err = listWithCursor(txn, agentPrefix, params, r.unmarshal, predicate)
		return err
	})
	return result, err
}

func buildAgentPredicate(f domain.AgentFilter) func(*domain.Agent) bool {
	if f.Search == "" && len(f.Labels) == 0 {
		return nil
	}
	return func(a *domain.Agent) bool {
		if f.Search != "" && !containsFold(a.Id, f.Search) && !containsFold(a.Hostname, f.Search) {
			return false
		}
		if len(f.Labels) > 0 && !matchesLabels(a.Labels, f.Labels) {
			return false
		}
		return true
	}
}

func (r *AgentRepository) Find(ctx context.Context, id string) (*domain.Agent, error) {
	var agent *domain.Agent

	err := r.db.View(func(txn *badgerdb.Txn) error {
		item, err := txn.Get(agentKey(id))
		if errors.Is(err, badgerdb.ErrKeyNotFound) {
			return fmt.Errorf("agent %q not found", id)
		}

		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			agent, err = r.unmarshal(val)
			return err
		})
	})

	if err != nil {
		return nil, err
	}

	return agent, nil
}

func (r *AgentRepository) Create(ctx context.Context, agent *domain.Agent) error {
	data, err := r.marshal(agent)
	if err != nil {
		return fmt.Errorf("marshall agent %q: %w", agent.Id, err)
	}

	// The check-then-set runs inside a single serializable transaction, so
	// concurrent creates for the same id cannot both succeed.
	return r.db.Update(func(txn *badgerdb.Txn) error {
		_, err := txn.Get(agentKey(agent.Id))
		if err == nil {
			return domain.ErrAgentExists
		}
		if !errors.Is(err, badgerdb.ErrKeyNotFound) {
			return err
		}
		return txn.Set(agentKey(agent.Id), data)
	})
}

func (r *AgentRepository) Save(ctx context.Context, agent *domain.Agent) error {
	data, err := r.marshal(agent)
	if err != nil {
		return fmt.Errorf("marshall agent %q: %w", agent.Id, err)
	}

	return r.db.Update(func(txn *badgerdb.Txn) error {
		return txn.Set(agentKey(agent.Id), data)
	})
}

func (r *AgentRepository) Delete(ctx context.Context, id string) error {
	return r.db.Update(func(txn *badgerdb.Txn) error {
		_, err := txn.Get(agentKey(id))
		if errors.Is(err, badgerdb.ErrKeyNotFound) {
			return fmt.Errorf("agent %q not found", id)
		}
		if err != nil {
			return err
		}
		return txn.Delete(agentKey(id))
	})
}
