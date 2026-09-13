package badger_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	badgerutil "github.com/aknEvrnky/pgway/integration/testutil/badger"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

func newTestAgent(id, hostname string, labels map[string]string) *domain.Agent {
	return &domain.Agent{Id: id, Hostname: hostname, Labels: labels}
}

func TestAgentRepository(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Create and Find", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		agent := newTestAgent("agent-1", "host-a", nil)
		require.NoError(t, store.Agents.Create(ctx, agent))

		got, err := store.Agents.Find(ctx, "agent-1")
		require.NoError(t, err)
		assert.Equal(t, agent, got)
	})

	t.Run("Create duplicate id", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		require.NoError(t, store.Agents.Create(ctx, newTestAgent("agent-1", "host-a", nil)))

		err := store.Agents.Create(ctx, newTestAgent("agent-1", "host-b", nil))
		assert.ErrorIs(t, err, domain.ErrAgentExists)
	})

	t.Run("Find missing agent", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		_, err := store.Agents.Find(ctx, "ghost")
		assert.ErrorContains(t, err, `agent "ghost" not found`)
	})

	t.Run("List returns all", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		require.NoError(t, store.Agents.Create(ctx, newTestAgent("agent-1", "web-01", nil)))
		require.NoError(t, store.Agents.Create(ctx, newTestAgent("agent-2", "db-01", nil)))

		result, err := store.Agents.List(ctx, domain.ListParams{}, domain.AgentFilter{})
		require.NoError(t, err)
		assert.Len(t, result.Items, 2)
		assert.Equal(t, 2, result.TotalCount)
	})

	t.Run("List with search filter matches id and hostname", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		require.NoError(t, store.Agents.Create(ctx, newTestAgent("agent-1", "web-01", nil)))
		require.NoError(t, store.Agents.Create(ctx, newTestAgent("agent-2", "db-01", nil)))

		result, err := store.Agents.List(ctx, domain.ListParams{}, domain.AgentFilter{Search: "AGENT-1"})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		assert.Equal(t, "agent-1", result.Items[0].Id)

		result, err = store.Agents.List(ctx, domain.ListParams{}, domain.AgentFilter{Search: "db-"})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		assert.Equal(t, "agent-2", result.Items[0].Id)
	})

	t.Run("List with labels filter", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		require.NoError(t, store.Agents.Create(ctx, newTestAgent("agent-1", "web-01", map[string]string{"region": "eu"})))
		require.NoError(t, store.Agents.Create(ctx, newTestAgent("agent-2", "db-01", map[string]string{"region": "us"})))

		result, err := store.Agents.List(ctx, domain.ListParams{}, domain.AgentFilter{Labels: map[string]string{"region": "eu"}})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		assert.Equal(t, "agent-1", result.Items[0].Id)
	})

	t.Run("Save overwrite", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		agent := newTestAgent("agent-1", "host-a", nil)
		require.NoError(t, store.Agents.Create(ctx, agent))

		agent.Hostname = "host-b"
		require.NoError(t, store.Agents.Save(ctx, agent))

		got, err := store.Agents.Find(ctx, "agent-1")
		require.NoError(t, err)
		assert.Equal(t, "host-b", got.Hostname)
	})

	t.Run("Delete", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		require.NoError(t, store.Agents.Create(ctx, newTestAgent("agent-1", "host-a", nil)))
		require.NoError(t, store.Agents.Delete(ctx, "agent-1"))

		_, err := store.Agents.Find(ctx, "agent-1")
		assert.Error(t, err)
	})

	t.Run("Delete missing agent", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		assert.ErrorContains(t, store.Agents.Delete(ctx, "ghost"), `agent "ghost" not found`)
	})

	t.Run("Delete does not touch same-id user", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		require.NoError(t, store.Users.Save(ctx, newTestUser("shared-id", domain.RoleMember)))
		require.NoError(t, store.Agents.Create(ctx, newTestAgent("shared-id", "host-a", nil)))

		require.NoError(t, store.Agents.Delete(ctx, "shared-id"))

		_, err := store.Agents.Find(ctx, "shared-id")
		assert.ErrorContains(t, err, "not found")
		_, err = store.Users.Find(ctx, "shared-id")
		assert.NoError(t, err)
	})
}
