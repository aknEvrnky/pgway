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

func hostname(t *testing.T) string {
	t.Helper()
	h, err := os.Hostname()
	require.NoError(t, err)
	return h
}

func TestLoad(t *testing.T) {
	host := hostname(t)

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
				BadgerGCInterval:       5 * time.Minute,
				GrpcListenAddr:          ":7000",
				GRPCKeepaliveInterval:   time.Minute,
				GRPCKeepaliveTimeout:    20 * time.Second,
				RestListenAddr:          ":7001",
				Token:                   "file-token",
				TokenTTL:                time.Hour,
				RegistrationTokenTTL:    24 * time.Hour,
				AgentTokenTTL:           168 * time.Hour,
				AgentHeartbeatThreshold: 30 * time.Second,
				AgentName:               host,
				AgentLabels:             map[string]string{},
				AgentStatePath:          "/var/lib/pgway/agent.json",
				HeartbeatInterval:       10 * time.Second,
				RegistrationToken:       "",
				MaxRequestBodyBytes:     ByteSize(10 << 20),
				ProxyMaxIdleConns:       1024,
				ProxyMaxIdleConnsPerHost: 128,
				ProxyIdleConnTimeout:    90 * time.Second,
				ProxyDialTimeout:        10 * time.Second,
				LogLevel:                "info",
			},
		},
		{
			name: "applies defaults for omitted keys",
			file: `badger_path: /data/pgway`,
			want: Config{
				BadgerPath:              "/data/pgway",
				BadgerGCInterval:       5 * time.Minute,
				GrpcListenAddr:          ":9090", // default must carry the leading colon
				GRPCKeepaliveInterval:   time.Minute,
				GRPCKeepaliveTimeout:    20 * time.Second,
				RestListenAddr:          ":8081",
				Token:                   "",
				TokenTTL:                720 * time.Hour,
				RegistrationTokenTTL:    24 * time.Hour,
				AgentTokenTTL:           168 * time.Hour,
				AgentHeartbeatThreshold: 30 * time.Second,
				AgentName:               host,
				AgentLabels:             map[string]string{},
				AgentStatePath:          "/var/lib/pgway/agent.json",
				HeartbeatInterval:       10 * time.Second,
				RegistrationToken:       "",
				MaxRequestBodyBytes:     ByteSize(10 << 20),
				ProxyMaxIdleConns:       1024,
				ProxyMaxIdleConnsPerHost: 128,
				ProxyIdleConnTimeout:    90 * time.Second,
				ProxyDialTimeout:        10 * time.Second,
				LogLevel:                "info",
			},
		},
		{
			name: "env var overrides file value",
			file: `token: file-token`,
			env:  map[string]string{"PGWAY_TOKEN": "env-token"},
			want: Config{
				BadgerPath:              "/var/pgway/lib",
				BadgerGCInterval:       5 * time.Minute,
				GrpcListenAddr:          ":9090",
				GRPCKeepaliveInterval:   time.Minute,
				GRPCKeepaliveTimeout:    20 * time.Second,
				RestListenAddr:          ":8081",
				Token:                   "env-token",
				TokenTTL:                720 * time.Hour,
				RegistrationTokenTTL:    24 * time.Hour,
				AgentTokenTTL:           168 * time.Hour,
				AgentHeartbeatThreshold: 30 * time.Second,
				AgentName:               host,
				AgentLabels:             map[string]string{},
				AgentStatePath:          "/var/lib/pgway/agent.json",
				HeartbeatInterval:       10 * time.Second,
				RegistrationToken:       "",
				MaxRequestBodyBytes:     ByteSize(10 << 20),
				ProxyMaxIdleConns:       1024,
				ProxyMaxIdleConnsPerHost: 128,
				ProxyIdleConnTimeout:    90 * time.Second,
				ProxyDialTimeout:        10 * time.Second,
				LogLevel:                "info",
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
				BadgerGCInterval:       5 * time.Minute,
				GrpcListenAddr:          ":9090",
				GRPCKeepaliveInterval:   time.Minute,
				GRPCKeepaliveTimeout:    20 * time.Second,
				RestListenAddr:          ":8081",
				Token:                   "",
				TokenTTL:                720 * time.Hour,
				RegistrationTokenTTL:    2 * time.Hour,
				AgentTokenTTL:           48 * time.Hour,
				AgentHeartbeatThreshold: 15 * time.Second,
				AgentName:               host,
				AgentLabels:             map[string]string{},
				AgentStatePath:          "/var/lib/pgway/agent.json",
				HeartbeatInterval:       10 * time.Second,
				RegistrationToken:       "",
				MaxRequestBodyBytes:     ByteSize(10 << 20),
				ProxyMaxIdleConns:       1024,
				ProxyMaxIdleConnsPerHost: 128,
				ProxyIdleConnTimeout:    90 * time.Second,
				ProxyDialTimeout:        10 * time.Second,
				LogLevel:                "info",
			},
		},
		{
			name: "reads dp agent fields from file",
			file: `badger_path: /data/pgway
agent_name: edge-1
agent_labels:
  region: us-east
agent_state_path: /tmp/pgway/agent.json
heartbeat_interval: 5s
registration_token: reg-secret`,
			want: Config{
				BadgerPath:              "/data/pgway",
				BadgerGCInterval:       5 * time.Minute,
				GrpcListenAddr:          ":9090",
				GRPCKeepaliveInterval:   time.Minute,
				GRPCKeepaliveTimeout:    20 * time.Second,
				RestListenAddr:          ":8081",
				Token:                   "",
				TokenTTL:                720 * time.Hour,
				RegistrationTokenTTL:    24 * time.Hour,
				AgentTokenTTL:           168 * time.Hour,
				AgentHeartbeatThreshold: 30 * time.Second,
				AgentName:               "edge-1",
				AgentLabels:             map[string]string{"region": "us-east"},
				AgentStatePath:          "/tmp/pgway/agent.json",
				HeartbeatInterval:       5 * time.Second,
				RegistrationToken:       "reg-secret",
				MaxRequestBodyBytes:     ByteSize(10 << 20),
				ProxyMaxIdleConns:       1024,
				ProxyMaxIdleConnsPerHost: 128,
				ProxyIdleConnTimeout:    90 * time.Second,
				ProxyDialTimeout:        10 * time.Second,
				LogLevel:                "info",
			},
		},
		{
			name: "registration token env overrides file",
			file: `badger_path: /data/pgway
registration_token: file-reg`,
			env: map[string]string{"PGWAY_REGISTRATION_TOKEN": "env-reg"},
			want: Config{
				BadgerPath:              "/data/pgway",
				BadgerGCInterval:       5 * time.Minute,
				GrpcListenAddr:          ":9090",
				GRPCKeepaliveInterval:   time.Minute,
				GRPCKeepaliveTimeout:    20 * time.Second,
				RestListenAddr:          ":8081",
				Token:                   "",
				TokenTTL:                720 * time.Hour,
				RegistrationTokenTTL:    24 * time.Hour,
				AgentTokenTTL:           168 * time.Hour,
				AgentHeartbeatThreshold: 30 * time.Second,
				AgentName:               host,
				AgentLabels:             map[string]string{},
				AgentStatePath:          "/var/lib/pgway/agent.json",
				HeartbeatInterval:       10 * time.Second,
				RegistrationToken:       "env-reg",
				MaxRequestBodyBytes:     ByteSize(10 << 20),
				ProxyMaxIdleConns:       1024,
				ProxyMaxIdleConnsPerHost: 128,
				ProxyIdleConnTimeout:    90 * time.Second,
				ProxyDialTimeout:        10 * time.Second,
				LogLevel:                "info",
			},
		},
		{
			name: "log_level env overrides file",
			file: `badger_path: /data/pgway
log_level: warn`,
			env: map[string]string{"PGWAY_LOG_LEVEL": "debug"},
			want: Config{
				BadgerPath:              "/data/pgway",
				BadgerGCInterval:       5 * time.Minute,
				GrpcListenAddr:          ":9090",
				GRPCKeepaliveInterval:   time.Minute,
				GRPCKeepaliveTimeout:    20 * time.Second,
				RestListenAddr:          ":8081",
				Token:                   "",
				TokenTTL:                720 * time.Hour,
				RegistrationTokenTTL:    24 * time.Hour,
				AgentTokenTTL:           168 * time.Hour,
				AgentHeartbeatThreshold: 30 * time.Second,
				AgentName:               host,
				AgentLabels:             map[string]string{},
				AgentStatePath:          "/var/lib/pgway/agent.json",
				HeartbeatInterval:       10 * time.Second,
				RegistrationToken:       "",
				MaxRequestBodyBytes:     ByteSize(10 << 20),
				ProxyMaxIdleConns:       1024,
				ProxyMaxIdleConnsPerHost: 128,
				ProxyIdleConnTimeout:    90 * time.Second,
				ProxyDialTimeout:        10 * time.Second,
				LogLevel:                "debug",
			},
		},
		{
			name: "max_request_body_bytes from file and env override",
			file: `badger_path: /data/pgway
max_request_body_bytes: 2KiB`,
			env: map[string]string{"PGWAY_MAX_REQUEST_BODY_BYTES": "4KiB"},
			want: Config{
				BadgerPath:              "/data/pgway",
				BadgerGCInterval:       5 * time.Minute,
				GrpcListenAddr:          ":9090",
				GRPCKeepaliveInterval:   time.Minute,
				GRPCKeepaliveTimeout:    20 * time.Second,
				RestListenAddr:          ":8081",
				Token:                   "",
				TokenTTL:                720 * time.Hour,
				RegistrationTokenTTL:    24 * time.Hour,
				AgentTokenTTL:           168 * time.Hour,
				AgentHeartbeatThreshold: 30 * time.Second,
				AgentName:               host,
				AgentLabels:             map[string]string{},
				AgentStatePath:          "/var/lib/pgway/agent.json",
				HeartbeatInterval:       10 * time.Second,
				RegistrationToken:       "",
				MaxRequestBodyBytes:     ByteSize(4 << 10),
				ProxyMaxIdleConns:       1024,
				ProxyMaxIdleConnsPerHost: 128,
				ProxyIdleConnTimeout:    90 * time.Second,
				ProxyDialTimeout:        10 * time.Second,
				LogLevel:                "info",
			},
		},
		{
			name: "max_request_body_bytes accepts 10MiB",
			file: `badger_path: /data/pgway
max_request_body_bytes: 10MiB`,
			want: Config{
				BadgerPath:              "/data/pgway",
				BadgerGCInterval:       5 * time.Minute,
				GrpcListenAddr:          ":9090",
				GRPCKeepaliveInterval:   time.Minute,
				GRPCKeepaliveTimeout:    20 * time.Second,
				RestListenAddr:          ":8081",
				Token:                   "",
				TokenTTL:                720 * time.Hour,
				RegistrationTokenTTL:    24 * time.Hour,
				AgentTokenTTL:           168 * time.Hour,
				AgentHeartbeatThreshold: 30 * time.Second,
				AgentName:               host,
				AgentLabels:             map[string]string{},
				AgentStatePath:          "/var/lib/pgway/agent.json",
				HeartbeatInterval:       10 * time.Second,
				RegistrationToken:       "",
				MaxRequestBodyBytes:     ByteSize(10 << 20),
				ProxyMaxIdleConns:       1024,
				ProxyMaxIdleConnsPerHost: 128,
				ProxyIdleConnTimeout:    90 * time.Second,
				ProxyDialTimeout:        10 * time.Second,
				LogLevel:                "info",
			},
		},
		{
			name: "reads grpc keepalive from file",
			file: `badger_path: /data/pgway
grpc_keepalive_interval: 30s
grpc_keepalive_timeout: 10s`,
			want: Config{
				BadgerPath:              "/data/pgway",
				BadgerGCInterval:        5 * time.Minute,
				GrpcListenAddr:          ":9090",
				GRPCKeepaliveInterval:   30 * time.Second,
				GRPCKeepaliveTimeout:    10 * time.Second,
				RestListenAddr:          ":8081",
				Token:                   "",
				TokenTTL:                720 * time.Hour,
				RegistrationTokenTTL:    24 * time.Hour,
				AgentTokenTTL:           168 * time.Hour,
				AgentHeartbeatThreshold: 30 * time.Second,
				AgentName:               host,
				AgentLabels:             map[string]string{},
				AgentStatePath:          "/var/lib/pgway/agent.json",
				HeartbeatInterval:       10 * time.Second,
				RegistrationToken:       "",
				MaxRequestBodyBytes:     ByteSize(10 << 20),
				ProxyMaxIdleConns:       1024,
				ProxyMaxIdleConnsPerHost: 128,
				ProxyIdleConnTimeout:    90 * time.Second,
				ProxyDialTimeout:        10 * time.Second,
				LogLevel:                "info",
			},
		},
		{
			name: "grpc keepalive interval 0 disables",
			file: `badger_path: /data/pgway
grpc_keepalive_interval: 0`,
			want: Config{
				BadgerPath:              "/data/pgway",
				BadgerGCInterval:        5 * time.Minute,
				GrpcListenAddr:          ":9090",
				GRPCKeepaliveInterval:   0,
				GRPCKeepaliveTimeout:    20 * time.Second,
				RestListenAddr:          ":8081",
				Token:                   "",
				TokenTTL:                720 * time.Hour,
				RegistrationTokenTTL:    24 * time.Hour,
				AgentTokenTTL:           168 * time.Hour,
				AgentHeartbeatThreshold: 30 * time.Second,
				AgentName:               host,
				AgentLabels:             map[string]string{},
				AgentStatePath:          "/var/lib/pgway/agent.json",
				HeartbeatInterval:       10 * time.Second,
				RegistrationToken:       "",
				MaxRequestBodyBytes:     ByteSize(10 << 20),
				ProxyMaxIdleConns:       1024,
				ProxyMaxIdleConnsPerHost: 128,
				ProxyIdleConnTimeout:    90 * time.Second,
				ProxyDialTimeout:        10 * time.Second,
				LogLevel:                "info",
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

func TestLoad_KeepaliveTimeoutRequiredWhenEnabled(t *testing.T) {
	viper.Reset()
	err := Load(writeConfig(t, `badger_path: /data/pgway
grpc_keepalive_interval: 1m
grpc_keepalive_timeout: 0`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "grpc_keepalive_timeout")
}

func TestLoad_ProxyTransportValidation(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		wantErr string
	}{
		{
			name: "per_host zero rejected",
			file: `badger_path: /data/pgway
proxy_max_idle_conns_per_host: 0`,
			wantErr: "proxy_max_idle_conns_per_host",
		},
		{
			name: "dial timeout zero rejected",
			file: `badger_path: /data/pgway
proxy_dial_timeout: 0`,
			wantErr: "proxy_dial_timeout",
		},
		{
			name: "negative max idle rejected",
			file: `badger_path: /data/pgway
proxy_max_idle_conns: -1`,
			wantErr: "proxy_max_idle_conns",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()
			err := Load(writeConfig(t, tt.file))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestLoad_ProxyTransportFromFile(t *testing.T) {
	viper.Reset()
	require.NoError(t, Load(writeConfig(t, `badger_path: /data/pgway
proxy_max_idle_conns: 512
proxy_max_idle_conns_per_host: 64
proxy_idle_conn_timeout: 30s
proxy_dial_timeout: 5s`)))

	cfg := Get()
	require.NotNil(t, cfg)
	assert.Equal(t, 512, cfg.ProxyMaxIdleConns)
	assert.Equal(t, 64, cfg.ProxyMaxIdleConnsPerHost)
	assert.Equal(t, 30*time.Second, cfg.ProxyIdleConnTimeout)
	assert.Equal(t, 5*time.Second, cfg.ProxyDialTimeout)
}

