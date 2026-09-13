package agentstate

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/aknEvrnky/pgway/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreSaveLoad(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "pgway")
	path := filepath.Join(dir, "agent.json")
	store := NewStore(path)

	require.NoError(t, store.Save(context.Background(), &ports.AgentHostCredentials{
		AgentID: "edge-1", AgentToken: "tok-abc",
	}))

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	dirInfo, err := os.Stat(dir)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), dirInfo.Mode().Perm())

	got, err := store.Load(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "edge-1", got.AgentID)
	assert.Equal(t, "tok-abc", got.AgentToken)
}

func TestStoreLoadMissing(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "missing.json"))
	_, err := store.Load(context.Background())
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestStoreRejectsIncomplete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"agent_id":"only"}`), 0o600))

	_, err := NewStore(path).Load(context.Background())
	assert.Error(t, err)
}

func TestStoreSaveRejectsEmpty(t *testing.T) {
	err := NewStore(filepath.Join(t.TempDir(), "agent.json")).Save(context.Background(), &ports.AgentHostCredentials{})
	assert.Error(t, err)
}

func TestLockPath(t *testing.T) {
	assert.Equal(t, "/var/lib/pgway/agent.lock", LockPath("/var/lib/pgway/agent.json"))
}

func TestAcquireLockExclusive(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "agent.json")
	lock := NewLock(statePath)

	first, err := lock.Acquire()
	require.NoError(t, err)
	defer first.Close()

	_, err = NewLock(statePath).Acquire()
	assert.Error(t, err)

	require.NoError(t, first.Close())

	second, err := NewLock(statePath).Acquire()
	require.NoError(t, err)
	require.NoError(t, second.Close())
}
