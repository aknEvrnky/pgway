package server

import (
	"testing"

	controlplanev1 "github.com/aknEvrnky/pgway/gen/pgway/controlplane/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProxySpecFromProto_ManualAuthMapsPassword(t *testing.T) {
	spec := proxySpecFromProto(&controlplanev1.ProxySpecV1{
		Protocol: "http",
		Host:     "10.0.0.1",
		Port:     3128,
		Auth: &controlplanev1.AuthSpec{
			User: "alice",
			Pass: "s3cret",
		},
	})

	require.NotNil(t, spec.Auth)
	assert.Equal(t, "alice", spec.Auth.User)
	assert.Equal(t, "s3cret", spec.Auth.Pass)
}
