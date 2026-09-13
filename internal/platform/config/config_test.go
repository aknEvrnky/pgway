package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeConfig writes contents to a temp config file and returns its path.
func writeConfig(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yml")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
	return path
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name string
		file string
		env  map[string]string
		want Config
	}{
		{
			name: "reads all fields from file",
			file: `badger_path: /data/pgway
grpc_listen_addr: ":7000"
rest_listen_addr: ":7001"
token: file-token
token_ttl: 1h`,
			want: Config{
				BadgerPath:              "/data/pgway",
				GrpcListenAddr:          ":7000",
				RestListenAddr:          ":7001",
				Token:                   "file-token",
				TokenTTL:                time.Hour,
				RegistrationTokenTTL:    24 * time.Hour,
				AgentTokenTTL:           168 * time.Hour,
				AgentHeartbeatThreshold: 30 * time.Second,
			},
		},
		{
			name: "applies defaults for omitted keys",
			file: `badger_path: /data/pgway`,
			want: Config{
				BadgerPath:              "/data/pgway",
				GrpcListenAddr:          ":9090", // default must carry the leading colon
				RestListenAddr:          ":8081",
				Token:                   "",
				TokenTTL:                720 * time.Hour,
				RegistrationTokenTTL:    24 * time.Hour,
				AgentTokenTTL:           168 * time.Hour,
				AgentHeartbeatThreshold: 30 * time.Second,
			},
		},
		{
			name: "env var overrides file value",
			file: `token: file-token`,
			env:  map[string]string{"PGWAY_TOKEN": "env-token"},
			want: Config{
				BadgerPath:              "/var/pgway/lib",
				GrpcListenAddr:          ":9090",
				RestListenAddr:          ":8081",
				Token:                   "env-token",
				TokenTTL:                720 * time.Hour,
				RegistrationTokenTTL:    24 * time.Hour,
				AgentTokenTTL:           168 * time.Hour,
				AgentHeartbeatThreshold: 30 * time.Second,
			},
		},
		{
			name: "reads agent ttl fields from file",
			file: `badger_path: /data/pgway
registration_token_ttl: 2h
agent_token_ttl: 48h
agent_heartbeat_threshold: 15s`,
			want: Config{
				BadgerPath:              "/data/pgway",
				GrpcListenAddr:          ":9090",
				RestListenAddr:          ":8081",
				Token:                   "",
				TokenTTL:                720 * time.Hour,
				RegistrationTokenTTL:    2 * time.Hour,
				AgentTokenTTL:           48 * time.Hour,
				AgentHeartbeatThreshold: 15 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			require.NoError(t, Load(writeConfig(t, tt.file)))

			cfg := Get()
			require.NotNil(t, cfg)
			assert.Equal(t, tt.want, *cfg)
		})
	}
}

func TestLoadMissingFile(t *testing.T) {
	viper.Reset()
	err := Load(filepath.Join(t.TempDir(), "does-not-exist.yml"))
	assert.Error(t, err)
}
