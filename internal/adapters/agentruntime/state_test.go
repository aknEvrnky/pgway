package agentruntime

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveLoadState(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "pgway")
	path := filepath.Join(dir, "agent.json")

	require.NoError(t, SaveState(path, &State{AgentID: "edge-1", AgentToken: "tok-abc"}))

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	dirInfo, err := os.Stat(dir)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), dirInfo.Mode().Perm())

	got, err := LoadState(path)
	require.NoError(t, err)
	assert.Equal(t, "edge-1", got.AgentID)
	assert.Equal(t, "tok-abc", got.AgentToken)
}

func TestLoadStateMissing(t *testing.T) {
	_, err := LoadState(filepath.Join(t.TempDir(), "missing.json"))
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestLoadStateRejectsIncomplete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"agent_id":"only"}`), 0o600))

	_, err := LoadState(path)
	assert.Error(t, err)
}

func TestSaveStateRejectsEmpty(t *testing.T) {
	err := SaveState(filepath.Join(t.TempDir(), "agent.json"), &State{})
	assert.Error(t, err)
}

func TestLockPath(t *testing.T) {
	assert.Equal(t, "/var/lib/pgway/agent.lock", LockPath("/var/lib/pgway/agent.json"))
}

func TestAcquireLockExclusive(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "agent.lock")

	first, err := AcquireLock(lockPath)
	require.NoError(t, err)
	defer first.Close()

	_, err = AcquireLock(lockPath)
	assert.Error(t, err)

	require.NoError(t, first.Close())

	second, err := AcquireLock(lockPath)
	require.NoError(t, err)
	require.NoError(t, second.Close())
}
