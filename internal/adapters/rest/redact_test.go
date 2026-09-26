package rest

import (
	"testing"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedactProxy_ClearsPassword(t *testing.T) {
	in := &domain.Proxy{
		Id: "p",
		Auth: &domain.BasicAuth{
			User: "u",
			Pass: "secret",
		},
	}
	out := redactProxy(in)
	require.NotNil(t, out.Auth)
	assert.Equal(t, "u", out.Auth.User)
	assert.Empty(t, out.Auth.Pass)
	assert.Equal(t, "secret", in.Auth.Pass, "original must not be mutated")
}
