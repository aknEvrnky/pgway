package auth

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- helpers ---

func newTestCredentialService() (*AgentCredentialService, *mockTokenRepo, *mockRegTokenRepo) {
	tokens := newMockTokenRepo()
	regTokens := newMockRegTokenRepo()
	return NewAgentCredentialService(tokens, regTokens), tokens, regTokens
}

// --- tests ---

func TestAgentCredentialService_CreateRegistrationToken(t *testing.T) {
	ctx := context.Background()

	t.Run("returns plaintext and stores only the hash", func(t *testing.T) {
		svc, _, regTokens := newTestCredentialService()

		plaintext, err := svc.CreateRegistrationToken(ctx, time.Hour)
		require.NoError(t, err)
		require.NotEmpty(t, plaintext)

		assert.Contains(t, plaintext, "pgwr_")

		record, ok := regTokens.regTokens[hashToken(plaintext)]
		require.True(t, ok, "record must be keyed by the hash, not the plaintext")
		assert.NotContains(t, regTokens.regTokens, plaintext)

		require.NotNil(t, record.ExpiresAt, "registration tokens must always expire")
		assert.WithinDuration(t, time.Now().Add(time.Hour), *record.ExpiresAt, time.Minute)
	})

	t.Run("non-positive ttl rejected", func(t *testing.T) {
		svc, _, regTokens := newTestCredentialService()

		for _, ttl := range []time.Duration{0, -time.Hour} {
			_, err := svc.CreateRegistrationToken(ctx, ttl)
			assert.Error(t, err, "ttl %v must be rejected", ttl)
		}
		assert.Empty(t, regTokens.regTokens)
	})
}

func TestAgentCredentialService_ConsumeRegistrationToken(t *testing.T) {
	ctx := context.Background()

	t.Run("valid token consumed exactly once", func(t *testing.T) {
		svc, _, regTokens := newTestCredentialService()

		plaintext, err := svc.CreateRegistrationToken(ctx, time.Hour)
		require.NoError(t, err)

		require.NoError(t, svc.ConsumeRegistrationToken(ctx, plaintext))
		assert.Empty(t, regTokens.regTokens, "consumption must burn the record")

		// single-use: the same token can never be consumed again
		err = svc.ConsumeRegistrationToken(ctx, plaintext)
		assert.ErrorIs(t, err, ErrInvalidRegistrationToken)
	})

	t.Run("unknown token", func(t *testing.T) {
		svc, _, _ := newTestCredentialService()
		err := svc.ConsumeRegistrationToken(ctx, "pgw_ghost")
		assert.ErrorIs(t, err, ErrInvalidRegistrationToken)
	})

	t.Run("empty token", func(t *testing.T) {
		svc, _, _ := newTestCredentialService()
		err := svc.ConsumeRegistrationToken(ctx, "")
		assert.ErrorIs(t, err, ErrInvalidRegistrationToken)
	})

	t.Run("expired token rejected and burned", func(t *testing.T) {
		svc, _, regTokens := newTestCredentialService()

		past := time.Now().Add(-time.Minute)
		hash := hashToken("pgw_stale")
		regTokens.regTokens[hash] = &domain.RegistrationToken{Hash: hash, ExpiresAt: &past}

		err := svc.ConsumeRegistrationToken(ctx, "pgw_stale")
		assert.ErrorIs(t, err, ErrInvalidRegistrationToken)
		// fail-clean: the stale record is gone either way
		assert.Empty(t, regTokens.regTokens)
	})
}

func TestAgentCredentialService_IssueAgentToken(t *testing.T) {
	ctx := context.Background()

	t.Run("issues bearer token bound to the agent", func(t *testing.T) {
		svc, tokens, _ := newTestCredentialService()

		plaintext, err := svc.IssueAgentToken(ctx, "edge-1", time.Hour)
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(plaintext, tokenPrefix), "agent tokens are bearer tokens")

		record, ok := tokens.tokens[hashToken(plaintext)]
		require.True(t, ok, "record must be keyed by the hash")
		assert.Equal(t, "edge-1", record.AgentId)
		assert.Empty(t, record.UserId)

		require.NotNil(t, record.ExpiresAt, "agent tokens must always expire (sliding ttl)")
		assert.WithinDuration(t, time.Now().Add(time.Hour), *record.ExpiresAt, time.Minute)
	})

	// regression: an invalid issuance must not leave a half-written record
	t.Run("empty agent id leaves no record", func(t *testing.T) {
		svc, tokens, _ := newTestCredentialService()

		_, err := svc.IssueAgentToken(ctx, "", time.Hour)
		assert.Error(t, err)
		assert.Empty(t, tokens.tokens)
	})

	t.Run("non-positive ttl rejected", func(t *testing.T) {
		svc, tokens, _ := newTestCredentialService()

		_, err := svc.IssueAgentToken(ctx, "edge-1", 0)
		assert.Error(t, err)
		assert.Empty(t, tokens.tokens)
	})
}

func TestAgentCredentialService_ExtendAgentToken(t *testing.T) {
	ctx := context.Background()

	t.Run("pushes expiry forward without changing the token value", func(t *testing.T) {
		svc, tokens, _ := newTestCredentialService()

		plaintext, err := svc.IssueAgentToken(ctx, "edge-1", time.Hour)
		require.NoError(t, err)
		hash := hashToken(plaintext)
		originalExpiry := *tokens.tokens[hash].ExpiresAt

		require.NoError(t, svc.ExtendAgentToken(ctx, plaintext, 24*time.Hour))

		record, ok := tokens.tokens[hash]
		require.True(t, ok, "sliding ttl must keep the same hash — the value never rotates")
		assert.True(t, record.ExpiresAt.After(originalExpiry))
		assert.WithinDuration(t, time.Now().Add(24*time.Hour), *record.ExpiresAt, time.Minute)
		assert.WithinDuration(t, time.Now(), record.UpdatedAt, time.Minute)
	})

	// regression: the sliding mechanism must never grant a user token immortality
	t.Run("user token cannot be extended", func(t *testing.T) {
		svc, tokens, _ := newTestCredentialService()
		hash := seedToken(tokens, "pgw_user", domain.Token{UserId: "admin"})

		err := svc.ExtendAgentToken(ctx, "pgw_user", 24*time.Hour)
		assert.ErrorIs(t, err, ErrInvalidToken)
		assert.Nil(t, tokens.tokens[hash].ExpiresAt, "user token must stay untouched")
	})

	t.Run("expired agent token cannot be resurrected", func(t *testing.T) {
		svc, tokens, _ := newTestCredentialService()
		past := time.Now().Add(-time.Minute)
		seedToken(tokens, "pgw_dead", domain.Token{AgentId: "edge-1", ExpiresAt: &past})

		err := svc.ExtendAgentToken(ctx, "pgw_dead", 24*time.Hour)
		assert.ErrorIs(t, err, ErrInvalidToken)
	})

	t.Run("unknown token", func(t *testing.T) {
		svc, _, _ := newTestCredentialService()
		err := svc.ExtendAgentToken(ctx, "pgw_ghost", 24*time.Hour)
		assert.ErrorIs(t, err, ErrInvalidToken)
	})

	t.Run("non-positive ttl rejected", func(t *testing.T) {
		svc, _, _ := newTestCredentialService()
		plaintext, err := svc.IssueAgentToken(ctx, "edge-1", time.Hour)
		require.NoError(t, err)

		assert.Error(t, svc.ExtendAgentToken(ctx, plaintext, 0))
	})
}

func TestAgentCredentialService_RevokeAgentTokens(t *testing.T) {
	ctx := context.Background()

	t.Run("revokes only that agent's tokens", func(t *testing.T) {
		svc, tokens, _ := newTestCredentialService()
		aHash1 := seedToken(tokens, "pgw_a1", domain.Token{AgentId: "agent-a"})
		aHash2 := seedToken(tokens, "pgw_a2", domain.Token{AgentId: "agent-a"})
		bHash := seedToken(tokens, "pgw_b1", domain.Token{AgentId: "agent-b"})
		userHash := seedToken(tokens, "pgw_u1", domain.Token{UserId: "admin"})

		require.NoError(t, svc.RevokeAgentTokens(ctx, "agent-a"))

		assert.NotContains(t, tokens.tokens, aHash1)
		assert.NotContains(t, tokens.tokens, aHash2)
		assert.Contains(t, tokens.tokens, bHash)
		assert.Contains(t, tokens.tokens, userHash)
	})

	t.Run("empty agent id rejected", func(t *testing.T) {
		svc, _, _ := newTestCredentialService()
		assert.Error(t, svc.RevokeAgentTokens(ctx, ""))
	})
}

// bridge test: a token minted by the credential service authenticates as an
// agent principal through the Authenticator when both share the same stores.
func TestAgentCredentialService_IssuedTokenAuthenticates(t *testing.T) {
	ctx := context.Background()

	authr, _, agents, tokens := newTestAuthenticator()
	svc := NewAgentCredentialService(tokens, newMockRegTokenRepo())

	agents.agents["edge-1"] = &domain.Agent{Id: "edge-1", Hostname: "edge-host"}

	plaintext, err := svc.IssueAgentToken(ctx, "edge-1", time.Hour)
	require.NoError(t, err)

	principal, err := authr.Authenticate(ctx, plaintext)
	require.NoError(t, err)
	assert.Equal(t, domain.PrincipalKindAgent, principal.Kind())
	assert.Equal(t, "edge-1", principal.Agent.Id)

	// and revocation severs exactly that access
	require.NoError(t, svc.RevokeAgentTokens(ctx, "edge-1"))
	_, err = authr.Authenticate(ctx, plaintext)
	assert.ErrorIs(t, err, ErrInvalidToken)
}
