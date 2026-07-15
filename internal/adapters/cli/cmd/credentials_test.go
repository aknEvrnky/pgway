package cmd

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCredentialsPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path, err := credentialsPath()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, ".pgctl", "credentials"), path)
}

func TestCredentialsRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// nothing stored yet
	assert.Empty(t, readCredentials())

	require.NoError(t, writeCredentials("secret-token"))
	assert.Equal(t, "secret-token", readCredentials())

	require.NoError(t, removeCredentials())
	assert.Empty(t, readCredentials())

	// removing again is a no-op
	assert.NoError(t, removeCredentials())
}
