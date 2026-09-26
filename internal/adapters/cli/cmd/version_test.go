package cmd_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/aknEvrnky/pgway/internal/adapters/cli/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionCmd_NoDial(t *testing.T) {
	dialed := false
	root := cmd.NewRootCmd(func(addr, token string) (cmd.Client, error) {
		dialed = true
		return nil, assert.AnError
	})
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"version"})
	require.NoError(t, root.Execute())
	assert.False(t, dialed, "version must not dial the control plane")
	assert.True(t, strings.Contains(buf.String(), "pgctl version"))
}

func TestVersionFlag_NoDial(t *testing.T) {
	dialed := false
	root := cmd.NewRootCmd(func(addr, token string) (cmd.Client, error) {
		dialed = true
		return nil, assert.AnError
	})
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"--version"})
	require.NoError(t, root.Execute())
	assert.False(t, dialed, "--version must not dial the control plane")
	assert.True(t, strings.Contains(buf.String(), "pgctl version"))
}
