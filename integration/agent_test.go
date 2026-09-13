package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/aknEvrnky/pgway/integration/testutil"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

// TestAgentFlow exercises registration → heartbeat → status → deregister →
// delete → token reuse failure over a real TCP gRPC connection.
func TestAgentFlow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	const heartbeatThreshold = 80 * time.Millisecond

	addr, authService := testutil.NewAuthTestServerWithOpts(t, testutil.AuthTestServerOpts{
		HeartbeatThreshold:   heartbeatThreshold,
		AgentTokenTTL:        time.Hour,
		RegistrationTokenTTL: time.Hour,
	})

	anon := newAuthedClient(t, addr, "")
	_, adminToken, err := anon.InitAdmin(ctx, authService.BootstrapToken(), "admin", "password123")
	require.NoError(t, err)
	admin := newAuthedClient(t, addr, adminToken)

	var regToken string
	t.Run("admin creates registration token", func(t *testing.T) {
		regToken, err = admin.CreateRegistrationToken(ctx, time.Hour)
		require.NoError(t, err)
		require.NotEmpty(t, regToken)
		assert.True(t, len(regToken) > 4)
	})

	var agentToken string
	t.Run("register exchanges token for agent credentials", func(t *testing.T) {
		agent, token, err := anon.Register(ctx, regToken, domain.Agent{
			Id:       "edge-1",
			Hostname: "edge-host",
			Version:  "v0.1.0",
			Labels:   map[string]string{"zone": "a"},
		})
		require.NoError(t, err)
		assert.Equal(t, "edge-1", agent.Id)
		require.NotEmpty(t, token)
		agentToken = token
	})

	t.Run("second register with same token fails", func(t *testing.T) {
		_, _, err := anon.Register(ctx, regToken, domain.Agent{Id: "edge-2"})
		assertGrpcCode(t, err, codes.Unauthenticated)
	})

	t.Run("list shows passive before first heartbeat", func(t *testing.T) {
		detailed, err := admin.ListAgentsDetailed(ctx, domain.ListParams{}, domain.AgentFilter{})
		require.NoError(t, err)
		require.Len(t, detailed, 1)
		assert.Equal(t, "passive", detailed[0].Status)
		assert.Equal(t, "edge-1", detailed[0].Info.Name)
	})

	agentClient := newAuthedClient(t, addr, agentToken)

	t.Run("heartbeat marks agent active and extends token", func(t *testing.T) {
		expires, err := agentClient.Heartbeat(ctx)
		require.NoError(t, err)
		assert.True(t, expires.After(time.Now()))

		detailed, err := admin.ListAgentsDetailed(ctx, domain.ListParams{}, domain.AgentFilter{})
		require.NoError(t, err)
		require.Len(t, detailed, 1)
		assert.Equal(t, "active", detailed[0].Status)
	})

	t.Run("agent can read resources but not write", func(t *testing.T) {
		_, err := agentClient.ListProxies(ctx, domain.ListParams{}, domain.ProxyFilter{})
		require.NoError(t, err)

		_, _, err = agentClient.CreateUser(ctx, "x", "password123", domain.RoleMember)
		assertGrpcCode(t, err, codes.PermissionDenied)
	})

	t.Run("user token cannot heartbeat", func(t *testing.T) {
		_, err := admin.Heartbeat(ctx)
		assertGrpcCode(t, err, codes.PermissionDenied)
	})

	t.Run("short threshold yields disconnected", func(t *testing.T) {
		time.Sleep(heartbeatThreshold + 40*time.Millisecond)
		detailed, err := admin.ListAgentsDetailed(ctx, domain.ListParams{}, domain.AgentFilter{})
		require.NoError(t, err)
		require.Len(t, detailed, 1)
		assert.Equal(t, "disconnected", detailed[0].Status)
	})

	t.Run("deregister yields passive", func(t *testing.T) {
		// heartbeat again so the agent is live, then deregister
		_, err := agentClient.Heartbeat(ctx)
		require.NoError(t, err)

		require.NoError(t, agentClient.Deregister(ctx))

		detailed, err := admin.ListAgentsDetailed(ctx, domain.ListParams{}, domain.AgentFilter{})
		require.NoError(t, err)
		require.Len(t, detailed, 1)
		assert.Equal(t, "passive", detailed[0].Status)
	})

	t.Run("delete agent revokes token", func(t *testing.T) {
		require.NoError(t, admin.DeleteAgent(ctx, "edge-1"))

		_, err := agentClient.Heartbeat(ctx)
		assertGrpcCode(t, err, codes.Unauthenticated)

		result, err := admin.ListAgents(ctx, domain.ListParams{}, domain.AgentFilter{})
		require.NoError(t, err)
		assert.Equal(t, 0, result.TotalCount)
	})
}
