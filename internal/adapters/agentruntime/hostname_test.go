package agentruntime

import (
	"os"
	"testing"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureHostname(t *testing.T) {
	host, err := os.Hostname()
	require.NoError(t, err)

	a := &domain.Agent{}
	require.NoError(t, EnsureHostname(a))
	assert.Equal(t, host, a.Hostname)

	a.Hostname = "fixed"
	require.NoError(t, EnsureHostname(a))
	assert.Equal(t, "fixed", a.Hostname)
}
