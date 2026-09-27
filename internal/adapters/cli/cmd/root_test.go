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
			name:        "bare IPv6 host is bracketed by JoinHostPort",
			credentials: "cred-token",
			args:        []string{"noop", "-H", "::1"},
			wantAddr:    "[::1]:9090",
			wantToken:   "cred-token",
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

// TestRootCmdFlagValidation pins fail-fast flag checking: an invalid port or
// a host that smuggles a port / brackets must error before dialing, instead
// of surfacing as a deferred generic Unavailable at the first RPC.
func TestRootCmdFlagValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "port zero", args: []string{"--port", "0"}, wantErr: "--port"},
		{name: "port negative", args: []string{"--port", "-1"}, wantErr: "--port"},
		{name: "port too large", args: []string{"--port", "65536"}, wantErr: "--port"},
		{name: "host with port", args: []string{"--host", "cp:9090"}, wantErr: "--host"},
		{name: "bracketed IPv6 host", args: []string{"--host", "[::1]"}, wantErr: "--host"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dialed := false
			root := newTestRoot(func(addr, token string) (Client, error) {
				dialed = true
				return fakeClient{}, nil
			}, "noop")
			root.SetArgs(append([]string{"noop"}, tt.args...))

			err := root.Execute()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
			assert.False(t, dialed, "invalid flags must fail before dialing")
		})
	}
}

// TestRootCmdTokenRequired pins token fail-fast semantics: commands whose
// RPCs all require auth error with a pointer to the token file (missing or
// empty) instead of a server-side Unauthenticated, while commands marked
// tokenless (login/init — their RPCs are exempt) dial without a token file.
func TestRootCmdTokenRequired(t *testing.T) {
	tests := []struct {
		name        string
		leaf        string
		credentials string // "" = no file, "empty" = present but blank
		wantErr     string
		wantToken   string
	}{
		{
			name:    "missing token file errors for authed commands",
			leaf:    "noop",
			wantErr: "no token file at",
		},
		{
			name:        "empty token file errors for authed commands",
			leaf:        "noop",
			credentials: "empty",
			wantErr:     "is empty",
		},
		{
			name:      "tokenless command dials without a token file",
			leaf:      "exempt",
			wantToken: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			if tt.credentials == "empty" {
				require.NoError(t, os.MkdirAll(filepath.Join(home, ".pgctl"), 0o700))
				require.NoError(t, os.WriteFile(filepath.Join(home, ".pgctl", "credentials"), []byte("   \n"), 0o600))
			}

			var gotToken string
			root := newTestRoot(func(addr, token string) (Client, error) {
				gotToken = token
				return fakeClient{}, nil
			}, "noop", "exempt")
			root.SetArgs([]string{tt.leaf})

			err := root.Execute()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantToken, gotToken)
		})
	}
}

func newTestRoot(connect ConnectFunc, leaves ...string) *cobra.Command {
	root := NewRootCmd(connect)
	for _, name := range leaves {
		leaf := &cobra.Command{
			Use:  name,
			RunE: func(*cobra.Command, []string) error { return nil },
		}
		if name == "exempt" {
			// mirrors the login/init marker: RPCs are auth-exempt
			leaf.Annotations = map[string]string{tokenlessAnnotation: "true"}
		}
		root.AddCommand(leaf)
	}
	return root
}
