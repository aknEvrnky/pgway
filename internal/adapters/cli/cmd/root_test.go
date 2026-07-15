package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeClient satisfies the Client interface; only Close is exercised by the
// root command's PersistentPostRun. Any other call would panic (nil embed).
type fakeClient struct {
	Client
}

func (fakeClient) Close() error { return nil }

// TestRootCmdTokenResolution verifies the PersistentPreRunE resolution order
// (--token flag > config/env token > credentials file) and that --config
// selects the config file that supplies grpc_listen_addr and the token.
func TestRootCmdTokenResolution(t *testing.T) {
	tests := []struct {
		name        string
		configBody  string
		credentials string // written to ~/.pgctl/credentials when non-empty
		tokenFlag   string // passed as --token when non-empty
		wantToken   string
		wantAddr    string
	}{
		{
			name:        "flag wins over config and credentials",
			configBody:  "grpc_listen_addr: \":7001\"\ntoken: config-token",
			credentials: "cred-token",
			tokenFlag:   "flag-token",
			wantToken:   "flag-token",
			wantAddr:    ":7001",
		},
		{
			name:        "config token used when no flag",
			configBody:  "grpc_listen_addr: \":7002\"\ntoken: config-token",
			credentials: "cred-token",
			wantToken:   "config-token",
			wantAddr:    ":7002",
		},
		{
			name:        "credentials file used when no flag or config token",
			configBody:  "grpc_listen_addr: \":7003\"",
			credentials: "cred-token",
			wantToken:   "cred-token",
			wantAddr:    ":7003",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()

			home := t.TempDir()
			t.Setenv("HOME", home)
			if tt.credentials != "" {
				require.NoError(t, os.MkdirAll(filepath.Join(home, ".pgctl"), 0o700))
				require.NoError(t, os.WriteFile(filepath.Join(home, ".pgctl", "credentials"), []byte(tt.credentials), 0o600))
			}

			configPath := filepath.Join(t.TempDir(), "config.yml")
			require.NoError(t, os.WriteFile(configPath, []byte(tt.configBody), 0o600))

			var gotAddr, gotToken string
			connect := func(addr, token string) (Client, error) {
				gotAddr, gotToken = addr, token
				return fakeClient{}, nil
			}

			root := NewRootCmd(connect)
			// a benign leaf so PersistentPreRunE runs without hitting a real server
			root.AddCommand(&cobra.Command{
				Use:  "noop",
				RunE: func(*cobra.Command, []string) error { return nil },
			})

			args := []string{"noop", "--config", configPath}
			if tt.tokenFlag != "" {
				args = append(args, "--token", tt.tokenFlag)
			}
			root.SetArgs(args)

			require.NoError(t, root.Execute())
			assert.Equal(t, tt.wantToken, gotToken)
			assert.Equal(t, tt.wantAddr, gotAddr)
		})
	}
}

func TestRootCmdConfigLoadError(t *testing.T) {
	viper.Reset()
	t.Setenv("HOME", t.TempDir())

	root := NewRootCmd(func(string, string) (Client, error) { return fakeClient{}, nil })
	root.AddCommand(&cobra.Command{Use: "noop", RunE: func(*cobra.Command, []string) error { return nil }})
	root.SetArgs([]string{"noop", "--config", filepath.Join(t.TempDir(), "missing.yml")})

	assert.Error(t, root.Execute())
}
