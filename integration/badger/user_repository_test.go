package badger_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	badgerutil "github.com/aknEvrnky/pgway/integration/testutil/badger"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

func newTestUser(id string, role domain.Role) *domain.User {
	return &domain.User{Id: id, PasswordHash: "$2a$10$hash", Role: role}
}

func TestUserRepository(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Save and Find", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		user := newTestUser("alice", domain.RoleAdmin)
		require.NoError(t, store.Users.Save(ctx, user))

		got, err := store.Users.Find(ctx, "alice")
		require.NoError(t, err)
		assert.Equal(t, user, got)
	})

	t.Run("Find missing user", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		_, err := store.Users.Find(ctx, "ghost")
		assert.ErrorContains(t, err, `user "ghost" not found`)
	})

	t.Run("Count", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)

		count, err := store.Users.Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, 0, count)

		require.NoError(t, store.Users.Save(ctx, newTestUser("alice", domain.RoleAdmin)))
		require.NoError(t, store.Users.Save(ctx, newTestUser("bob", domain.RoleMember)))

		count, err = store.Users.Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("List with role filter", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		require.NoError(t, store.Users.Save(ctx, newTestUser("alice", domain.RoleAdmin)))
		require.NoError(t, store.Users.Save(ctx, newTestUser("bob", domain.RoleMember)))

		result, err := store.Users.List(ctx, domain.ListParams{}, domain.UserFilter{Role: "admin"})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		assert.Equal(t, "alice", result.Items[0].Id)
	})

	t.Run("List with search filter", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		require.NoError(t, store.Users.Save(ctx, newTestUser("alice", domain.RoleAdmin)))
		require.NoError(t, store.Users.Save(ctx, newTestUser("bob", domain.RoleMember)))

		result, err := store.Users.List(ctx, domain.ListParams{}, domain.UserFilter{Search: "ALI"})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		assert.Equal(t, "alice", result.Items[0].Id)
	})

	t.Run("Delete", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		require.NoError(t, store.Users.Save(ctx, newTestUser("alice", domain.RoleAdmin)))
		require.NoError(t, store.Users.Delete(ctx, "alice"))

		_, err := store.Users.Find(ctx, "alice")
		assert.Error(t, err)
	})

	t.Run("Delete missing user", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		assert.ErrorContains(t, store.Users.Delete(ctx, "ghost"), `user "ghost" not found`)
	})
}

func TestTokenRepository(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Save and Find", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		token := &domain.Token{Hash: "hash-1", UserId: "alice"}
		require.NoError(t, store.Tokens.Save(ctx, token))

		got, err := store.Tokens.Find(ctx, "hash-1")
		require.NoError(t, err)
		assert.Equal(t, token, got)
	})

	t.Run("Find missing token", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		_, err := store.Tokens.Find(ctx, "ghost")
		assert.ErrorContains(t, err, "token not found")
	})

	t.Run("Save already expired token fails", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		past := time.Now().Add(-time.Hour)
		err := store.Tokens.Save(ctx, &domain.Token{Hash: "h", UserId: "alice", ExpiresAt: &past})
		assert.ErrorContains(t, err, "token already expired")
	})

	t.Run("Save with future expiry is readable", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		future := time.Now().Add(time.Hour)
		require.NoError(t, store.Tokens.Save(ctx, &domain.Token{Hash: "h", UserId: "alice", ExpiresAt: &future}))

		got, err := store.Tokens.Find(ctx, "h")
		require.NoError(t, err)
		assert.Equal(t, "alice", got.UserId)
	})

	t.Run("Delete", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		require.NoError(t, store.Tokens.Save(ctx, &domain.Token{Hash: "h", UserId: "alice"}))
		require.NoError(t, store.Tokens.Delete(ctx, "h"))

		_, err := store.Tokens.Find(ctx, "h")
		assert.Error(t, err)
	})

	t.Run("DeleteByUserId removes only that user's tokens", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		require.NoError(t, store.Tokens.Save(ctx, &domain.Token{Hash: "a1", UserId: "alice"}))
		require.NoError(t, store.Tokens.Save(ctx, &domain.Token{Hash: "a2", UserId: "alice"}))
		require.NoError(t, store.Tokens.Save(ctx, &domain.Token{Hash: "b1", UserId: "bob"}))

		require.NoError(t, store.Tokens.DeleteByUserId(ctx, "alice"))

		_, err := store.Tokens.Find(ctx, "a1")
		assert.Error(t, err)
		_, err = store.Tokens.Find(ctx, "a2")
		assert.Error(t, err)

		got, err := store.Tokens.Find(ctx, "b1")
		require.NoError(t, err)
		assert.Equal(t, "bob", got.UserId)
	})

	t.Run("DeleteByAgentId removes only that agent's tokens", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		require.NoError(t, store.Tokens.Save(ctx, &domain.Token{Hash: "a1", AgentId: "agent-alpha"}))
		require.NoError(t, store.Tokens.Save(ctx, &domain.Token{Hash: "f1", AgentId: "agent-alpha"}))
		require.NoError(t, store.Tokens.Save(ctx, &domain.Token{Hash: "b1", AgentId: "agent-beta"}))

		require.NoError(t, store.Tokens.DeleteByAgentId(ctx, "agent-alpha"))

		_, err := store.Tokens.Find(ctx, "a1")
		assert.Error(t, err)
		_, err = store.Tokens.Find(ctx, "f1")
		assert.Error(t, err)

		got, err := store.Tokens.Find(ctx, "b1")
		require.NoError(t, err)
		assert.Equal(t, "agent-beta", got.AgentId)
	})
}
