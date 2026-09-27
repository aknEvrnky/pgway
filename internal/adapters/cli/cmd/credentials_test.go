package cmd

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultTokenPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path, err := defaultTokenPath()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, ".pgctl", "credentials"), path)
}

func TestTokenFileRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "token")

	// nothing stored yet
	assert.Empty(t, readTokenFile(path))

	require.NoError(t, writeTokenFile(path, "secret-token\n"))
	assert.Equal(t, "secret-token", readTokenFile(path))

	require.NoError(t, removeTokenFile(path))
	assert.Empty(t, readTokenFile(path))

	// removing again is a no-op
	assert.NoError(t, removeTokenFile(path))
}
