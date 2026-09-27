package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeClient satisfies the Client interface; only Close is exercised by the
// root command's PersistentPostRun. Any other call would panic (nil embed).
type fakeClient struct {
	Client
}

func (fakeClient) Close() error { return nil }

// TestRootCmdConnectionResolution verifies the post-config-decoupling
// connection model: the address comes from --host/--port flags (defaults
// 127.0.0.1:9090) and the bearer token comes only from the token file
// (--token-path, default $HOME/.pgctl/credentials). Each case runs with a
// decoy config file and PGWAY_* env vars planted on the old resolution
// paths — the decoupling regression test.
func TestRootCmdConnectionResolution(t *testing.T) {
	tests := []struct {
		name        string
		credentials string // written to $HOME/.pgctl/credentials when non-empty
		customToken string // written to a temp file passed via --token-path
		args        []string
		wantAddr    string
		wantToken   string
	}{
		{
			name:        "defaults dial loopback 9090 and read default token file",
			credentials: "cred-token",
			args:        []string{"noop"},
			wantAddr:    "127.0.0.1:9090",
			wantToken:   "cred-token",
		},
		{
			name:        "long flags override host and port",
			credentials: "cred-token",
			args:        []string{"noop", "--host", "cp.internal", "--port", "7001"},
			wantAddr:    "cp.internal:7001",
			wantToken:   "cred-token",
		},
		{
			name:        "shorthand flags -H and -P",
			credentials: "cred-token",
			args:        []string{"noop", "-H", "10.0.0.5", "-P", "7002"},
			wantAddr:    "10.0.0.5:7002",
			wantToken:   "cred-token",
		},
		{
			name:        "custom token path",
			customToken: "vault-token",
			args:        []string{"noop"},
			wantAddr:    "127.0.0.1:9090",
			wantToken:   "vault-token",
		},
		{
			name:     "missing token file means empty token",
			args:     []string{"noop"},
			wantAddr: "127.0.0.1:9090",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)

			// Decoy config file and env vars on the old resolution paths: if
			// pgctl ever regains config/env support, these values leak into
			// the assertions below instead of the flag/file-derived ones.
			require.NoError(t, os.MkdirAll(filepath.Join(home, ".pgway"), 0o700))
			decoy := filepath.Join(home, ".pgway", "config.toml")
			require.NoError(t, os.WriteFile(decoy, []byte("token = \"config-token\"\n\n[grpc]\nlisten_addr = \":6666\"\ndial_addr = \"config-host:6667\"\n"), 0o600))
			t.Setenv("PGWAY_TOKEN", "env-token")
			t.Setenv("PGWAY_GRPC_LISTEN_ADDR", "env-host:6668")
			t.Setenv("PGWAY_GRPC_DIAL_ADDR", "env-host:6669")

			if tt.credentials != "" {
				require.NoError(t, os.MkdirAll(filepath.Join(home, ".pgctl"), 0o700))
				require.NoError(t, os.WriteFile(filepath.Join(home, ".pgctl", "credentials"), []byte(tt.credentials), 0o600))
			}

			args := append([]string(nil), tt.args...)
			if tt.customToken != "" {
				p := filepath.Join(t.TempDir(), "custom-token")
				require.NoError(t, os.WriteFile(p, []byte(tt.customToken), 0o600))
				args = append(args, "--token-path", p)
			}

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
			root.SetArgs(args)

			require.NoError(t, root.Execute())
			assert.Equal(t, tt.wantAddr, gotAddr)
			assert.Equal(t, tt.wantToken, gotToken)
		})
	}
}
