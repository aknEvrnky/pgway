package cmd

import (
	"os"
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

func TestRequireTokenFile(t *testing.T) {
	t.Run("missing file errors with a pointer to the path", func(t *testing.T) {
		_, err := requireTokenFile(filepath.Join(t.TempDir(), "absent"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no token file at")
	})

	t.Run("blank file errors instead of yielding an empty token", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "blank-token")
		require.NoError(t, os.WriteFile(path, []byte("  \n"), 0o600))
		_, err := requireTokenFile(path)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "is empty")
	})

	t.Run("valid file returns the trimmed token", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "token")
		require.NoError(t, os.WriteFile(path, []byte("tok\n"), 0o600))
		token, err := requireTokenFile(path)
		require.NoError(t, err)
		assert.Equal(t, "tok", token)
	})
}
