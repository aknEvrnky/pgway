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
	path := filepath.Join(t.TempDir(), "config.toml")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
	return path
}

func hostname(t *testing.T) string {
	t.Helper()
	h, err := os.Hostname()
	require.NoError(t, err)
	return h
}

func defaultWant(host string) Config {
	return Config{
		LogLevel: "info",
		Token:    "",
		Badger: BadgerConfig{
			Path:       "/var/pgway/lib",
			GCInterval: 5 * time.Minute,
		},
		GRPC: GRPCConfig{
			ListenAddr:        ":9090",
			KeepaliveInterval: time.Minute,
			KeepaliveTimeout:  20 * time.Second,
			RateLimitRPS:      100,
			RateLimitBurst:    200,
		},
		Rest: RestConfig{ListenAddr: ":8081"},
		Probes: ProbesConfig{
			Enabled:    false,
			ListenAddr: ":8082",
		},
		Auth: AuthConfig{
			TokenTTL:             720 * time.Hour,
			RegistrationTokenTTL: 24 * time.Hour,
			AgentTokenTTL:        168 * time.Hour,
		},
		Agent: AgentConfig{
			Name:               host,
			Labels:             map[string]string{},
			StatePath:          "/var/lib/pgway/agent.json",
			HeartbeatInterval:  10 * time.Second,
			HeartbeatThreshold: 30 * time.Second,
			RegistrationToken:  "",
		},
		Proxy: ProxyConfig{
			MaxRequestBodyBytes: ByteSize(10 << 20),
			MaxIdleConns:        1024,
			MaxIdleConnsPerHost: 128,
			IdleConnTimeout:     90 * time.Second,
			DialTimeout:         10 * time.Second,
			DNSCache: DNSCacheConfig{
				Enabled: true,
				TTL:     5 * time.Minute,
			},
		},
		Dataplane: DataplaneConfig{
			EventCoalesceWindow:              100 * time.Millisecond,
			EventCoalesceMaxBuffer:           256,
			CPDisconnectStrategy:             "fail_open",
			CPDisconnectUnreachableThreshold: 30 * time.Second,
			CPDisconnectRecoverThreshold:     0,
			EventResyncInterval:              5 * time.Minute,
		},
		Otel: OtelConfig{
			Enabled:        false,
			Endpoint:       "localhost:4317",
			Insecure:       false,
			ServiceName:    "",
			ExportInterval: 15 * time.Second,
		},
	}
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
			file: `
token = "file-token"

[badger]
path = "/data/pgway"

[grpc]
listen_addr = ":7000"

[rest]
listen_addr = ":7001"

[auth]
token_ttl = "1h"
`,
			want: func() Config {
				w := defaultWant(host)
				w.Token = "file-token"
				w.Badger.Path = "/data/pgway"
				w.GRPC.ListenAddr = ":7000"
				w.Rest.ListenAddr = ":7001"
				w.Auth.TokenTTL = time.Hour
				return w
			}(),
		},
		{
			name: "applies defaults for omitted keys",
			file: `
[badger]
path = "/data/pgway"
`,
			want: func() Config {
				w := defaultWant(host)
				w.Badger.Path = "/data/pgway"
				return w
			}(),
		},
		{
			name: "env var overrides file value",
			file: `token = "file-token"`,
			env:  map[string]string{"PGWAY_TOKEN": "env-token"},
			want: func() Config {
				w := defaultWant(host)
				w.Token = "env-token"
				return w
			}(),
		},
		{
			name: "reads agent ttl fields from file",
			file: `
[badger]
path = "/data/pgway"

[auth]
registration_token_ttl = "2h"
agent_token_ttl = "48h"

[agent]
heartbeat_threshold = "15s"
`,
			want: func() Config {
				w := defaultWant(host)
				w.Badger.Path = "/data/pgway"
				w.Auth.RegistrationTokenTTL = 2 * time.Hour
				w.Auth.AgentTokenTTL = 48 * time.Hour
				w.Agent.HeartbeatThreshold = 15 * time.Second
				return w
			}(),
		},
		{
			name: "reads dp agent fields from file",
			file: `
[badger]
path = "/data/pgway"

[agent]
name = "edge-1"
labels = { region = "us-east" }
state_path = "/tmp/pgway/agent.json"
heartbeat_interval = "5s"
registration_token = "reg-secret"
`,
			want: func() Config {
				w := defaultWant(host)
				w.Badger.Path = "/data/pgway"
				w.Agent.Name = "edge-1"
				w.Agent.Labels = map[string]string{"region": "us-east"}
				w.Agent.StatePath = "/tmp/pgway/agent.json"
				w.Agent.HeartbeatInterval = 5 * time.Second
				w.Agent.RegistrationToken = "reg-secret"
				return w
			}(),
		},
		{
			name: "registration token env overrides file",
			file: `
[badger]
path = "/data/pgway"

[agent]
registration_token = "file-reg"
`,
			env: map[string]string{"PGWAY_AGENT_REGISTRATION_TOKEN": "env-reg"},
			want: func() Config {
				w := defaultWant(host)
				w.Badger.Path = "/data/pgway"
				w.Agent.RegistrationToken = "env-reg"
				return w
			}(),
		},
		{
			name: "log_level env overrides file",
			file: `
log_level = "warn"

[badger]
path = "/data/pgway"
`,
			env: map[string]string{"PGWAY_LOG_LEVEL": "debug"},
			want: func() Config {
				w := defaultWant(host)
				w.LogLevel = "debug"
				w.Badger.Path = "/data/pgway"
				return w
			}(),
		},
		{
			name: "max_request_body_bytes from file and env override",
			file: `
[badger]
path = "/data/pgway"

[proxy]
max_request_body_bytes = "2KiB"
`,
			env: map[string]string{"PGWAY_PROXY_MAX_REQUEST_BODY_BYTES": "4KiB"},
			want: func() Config {
				w := defaultWant(host)
				w.Badger.Path = "/data/pgway"
				w.Proxy.MaxRequestBodyBytes = ByteSize(4 << 10)
				return w
			}(),
		},
		{
			name: "max_request_body_bytes accepts 10MiB",
			file: `
[badger]
path = "/data/pgway"

[proxy]
max_request_body_bytes = "10MiB"
`,
			want: func() Config {
				w := defaultWant(host)
				w.Badger.Path = "/data/pgway"
				w.Proxy.MaxRequestBodyBytes = ByteSize(10 << 20)
				return w
			}(),
		},
		{
			name: "reads grpc keepalive from file",
			file: `
[badger]
path = "/data/pgway"

[grpc]
keepalive_interval = "30s"
keepalive_timeout = "10s"
`,
			want: func() Config {
				w := defaultWant(host)
				w.Badger.Path = "/data/pgway"
				w.GRPC.KeepaliveInterval = 30 * time.Second
				w.GRPC.KeepaliveTimeout = 10 * time.Second
				return w
			}(),
		},
		{
			name: "grpc keepalive interval 0 disables",
			file: `
[badger]
path = "/data/pgway"

[grpc]
keepalive_interval = "0"
`,
			want: func() Config {
				w := defaultWant(host)
				w.Badger.Path = "/data/pgway"
				w.GRPC.KeepaliveInterval = 0
				return w
			}(),
		},
		{
			name: "reads grpc rate limit from file",
			file: `
[badger]
path = "/data/pgway"

[grpc]
rate_limit_rps = 50
rate_limit_burst = 75
`,
			want: func() Config {
				w := defaultWant(host)
				w.Badger.Path = "/data/pgway"
				w.GRPC.RateLimitRPS = 50
				w.GRPC.RateLimitBurst = 75
				return w
			}(),
		},
		{
			name: "grpc rate limit rps 0 disables",
			file: `
[badger]
path = "/data/pgway"

[grpc]
rate_limit_rps = 0
`,
			want: func() Config {
				w := defaultWant(host)
				w.Badger.Path = "/data/pgway"
				w.GRPC.RateLimitRPS = 0
				return w
			}(),
		},
		{
			name: "reads dataplane event coalesce from file",
			file: `
[badger]
path = "/data/pgway"

[dataplane]
event_coalesce_window = "50ms"
event_coalesce_max_buffer = 128
`,
			want: func() Config {
				w := defaultWant(host)
				w.Badger.Path = "/data/pgway"
				w.Dataplane.EventCoalesceWindow = 50 * time.Millisecond
				w.Dataplane.EventCoalesceMaxBuffer = 128
				return w
			}(),
		},
		{
			name: "dataplane event coalesce window 0 disables",
			file: `
[badger]
path = "/data/pgway"

[dataplane]
event_coalesce_window = "0"
`,
			want: func() Config {
				w := defaultWant(host)
				w.Badger.Path = "/data/pgway"
				w.Dataplane.EventCoalesceWindow = 0
				return w
			}(),
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
	err := Load(filepath.Join(t.TempDir(), "does-not-exist.toml"))
	assert.Error(t, err)
}

func TestLoad_KeepaliveTimeoutRequiredWhenEnabled(t *testing.T) {
	viper.Reset()
	err := Load(writeConfig(t, `
[badger]
path = "/data/pgway"

[grpc]
keepalive_interval = "1m"
keepalive_timeout = "0"
`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "grpc.keepalive_timeout")
}

func TestLoad_RateLimitBurstRequiredWhenEnabled(t *testing.T) {
	viper.Reset()
	err := Load(writeConfig(t, `
[badger]
path = "/data/pgway"

[grpc]
rate_limit_rps = 10
rate_limit_burst = 0
`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "grpc.rate_limit_burst")
}

func TestLoad_EventCoalesceMaxBufferRequiredWhenEnabled(t *testing.T) {
	viper.Reset()
	err := Load(writeConfig(t, `
[badger]
path = "/data/pgway"

[dataplane]
event_coalesce_window = "100ms"
event_coalesce_max_buffer = 0
`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dataplane.event_coalesce_max_buffer")
}

func TestLoad_CPDisconnectValidation(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		wantErr string
	}{
		{
			name: "bad strategy",
			file: `
[badger]
path = "/data/pgway"

[dataplane]
cp_disconnect_strategy = "panic"
`,
			wantErr: "cp_disconnect_strategy",
		},
		{
			name: "negative unreachable threshold",
			file: `
[badger]
path = "/data/pgway"

[dataplane]
cp_disconnect_unreachable_threshold = "-1s"
`,
			wantErr: "cp_disconnect_unreachable_threshold",
		},
		{
			name: "negative recover threshold",
			file: `
[badger]
path = "/data/pgway"

[dataplane]
cp_disconnect_recover_threshold = "-5s"
`,
			wantErr: "cp_disconnect_recover_threshold",
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

func TestLoad_ProbesValidation(t *testing.T) {
	tests := []struct {
		name string
		file string
	}{
		{
			name: "empty listen_addr",
			file: `
[badger]
path = "/data/pgway"

[probes]
enabled = true
listen_addr = ""
`,
		},
		{
			name: "whitespace-only listen_addr",
			file: `
[badger]
path = "/data/pgway"

[probes]
enabled = true
listen_addr = "   "
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()
			err := Load(writeConfig(t, tt.file))
			require.Error(t, err)
			assert.Contains(t, err.Error(), "probes.listen_addr")
		})
	}
}

func TestLoad_ProbesEnabled(t *testing.T) {
	viper.Reset()
	require.NoError(t, Load(writeConfig(t, `
[badger]
path = "/data/pgway"

[probes]
enabled = true
listen_addr = "127.0.0.1:18082"
`)))
	cfg := Get()
	require.NotNil(t, cfg)
	assert.True(t, cfg.Probes.Enabled)
	assert.Equal(t, "127.0.0.1:18082", cfg.Probes.ListenAddr)
}

func TestLoad_OtelValidation(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		wantErr string
	}{
		{
			name: "empty endpoint when enabled",
			file: `
[badger]
path = "/data/pgway"

[otel]
enabled = true
endpoint = ""
`,
			wantErr: "otel.endpoint",
		},
		{
			name: "zero export interval when enabled",
			file: `
[badger]
path = "/data/pgway"

[otel]
enabled = true
endpoint = "localhost:4317"
export_interval = "0s"
`,
			wantErr: "otel.export_interval",
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

func TestLoad_OtelEnabled(t *testing.T) {
	viper.Reset()
	require.NoError(t, Load(writeConfig(t, `
[badger]
path = "/data/pgway"

[otel]
enabled = true
endpoint = "collector:4317"
service_name = "pgway-test"
export_interval = "30s"
`)))
	cfg := Get()
	require.NotNil(t, cfg)
	assert.True(t, cfg.Otel.Enabled)
	assert.Equal(t, "collector:4317", cfg.Otel.Endpoint)
	assert.Equal(t, "pgway-test", cfg.Otel.ServiceName)
	assert.Equal(t, 30*time.Second, cfg.Otel.ExportInterval)
}

func TestLoad_OtelInsecure(t *testing.T) {
	viper.Reset()
	require.NoError(t, Load(writeConfig(t, `
[badger]
path = "/data/pgway"

[otel]
enabled = true
endpoint = "collector:4317"
insecure = true
`)))
	cfg := Get()
	require.NotNil(t, cfg)
	assert.True(t, cfg.Otel.Insecure)
}

func TestLoad_ProxyTransportValidation(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		wantErr string
	}{
		{
			name: "per_host zero rejected",
			file: `
[badger]
path = "/data/pgway"

[proxy]
max_idle_conns_per_host = 0
`,
			wantErr: "proxy.max_idle_conns_per_host",
		},
		{
			name: "dial timeout zero rejected",
			file: `
[badger]
path = "/data/pgway"

[proxy]
dial_timeout = "0"
`,
			wantErr: "proxy.dial_timeout",
		},
		{
			name: "negative max idle rejected",
			file: `
[badger]
path = "/data/pgway"

[proxy]
max_idle_conns = -1
`,
			wantErr: "proxy.max_idle_conns",
		},
		{
			name: "dns cache ttl zero rejected when enabled",
			file: `
[badger]
path = "/data/pgway"

[proxy.dns_cache]
enabled = true
ttl = "0"
`,
			wantErr: "proxy.dns_cache.ttl",
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
	require.NoError(t, Load(writeConfig(t, `
[badger]
path = "/data/pgway"

[proxy]
max_idle_conns = 512
max_idle_conns_per_host = 64
idle_conn_timeout = "30s"
dial_timeout = "5s"
`)))

	cfg := Get()
	require.NotNil(t, cfg)
	assert.Equal(t, 512, cfg.Proxy.MaxIdleConns)
	assert.Equal(t, 64, cfg.Proxy.MaxIdleConnsPerHost)
	assert.Equal(t, 30*time.Second, cfg.Proxy.IdleConnTimeout)
	assert.Equal(t, 5*time.Second, cfg.Proxy.DialTimeout)
}

func TestLoad_DNSCacheFromFileAndEnv(t *testing.T) {
	viper.Reset()
	t.Setenv("PGWAY_PROXY_DNS_CACHE_TTL", "2m")
	require.NoError(t, Load(writeConfig(t, `
[badger]
path = "/data/pgway"

[proxy.dns_cache]
enabled = false
ttl = "10m"
`)))

	cfg := Get()
	require.NotNil(t, cfg)
	assert.False(t, cfg.Proxy.DNSCache.Enabled)
	assert.Equal(t, 2*time.Minute, cfg.Proxy.DNSCache.TTL)
}

func TestLoad_DNSCacheDisabledIgnoresZeroTTL(t *testing.T) {
	viper.Reset()
	require.NoError(t, Load(writeConfig(t, `
[badger]
path = "/data/pgway"

[proxy.dns_cache]
enabled = false
ttl = "0"
`)))

	cfg := Get()
	require.NotNil(t, cfg)
	assert.False(t, cfg.Proxy.DNSCache.Enabled)
	assert.Equal(t, time.Duration(0), cfg.Proxy.DNSCache.TTL)
}
