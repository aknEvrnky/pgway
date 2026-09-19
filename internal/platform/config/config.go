package config

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

type Config struct {
	// LogLevel sets the global zap log level (debug|info|warn|error).
	LogLevel string `mapstructure:"log_level"`
	// Token authenticates outgoing control-plane calls (pgctl).
	// pgway-dp uses an agent token from the state file after registration.
	Token string `mapstructure:"token"`

	Badger    BadgerConfig    `mapstructure:"badger"`
	GRPC      GRPCConfig      `mapstructure:"grpc"`
	Rest      RestConfig      `mapstructure:"rest"`
	Auth      AuthConfig      `mapstructure:"auth"`
	Agent     AgentConfig     `mapstructure:"agent"`
	Proxy     ProxyConfig     `mapstructure:"proxy"`
	Dataplane DataplaneConfig `mapstructure:"dataplane"`
}

type BadgerConfig struct {
	Path string `mapstructure:"path"`
	// GCInterval is how often value log GC runs (pgway, pgway-cp).
	// <= 0 disables background GC.
	GCInterval time.Duration `mapstructure:"gc_interval"`
}

type GRPCConfig struct {
	ListenAddr string `mapstructure:"listen_addr"`
	// DialAddr is the CP address clients dial (pgctl, pgway-dp). When empty,
	// DialTarget falls back to ListenAddr (convenient for all-in-one / local).
	DialAddr string `mapstructure:"dial_addr"`
	// KeepaliveInterval is how often idle gRPC connections send pings.
	// <= 0 disables keepalive on both server and client.
	KeepaliveInterval time.Duration `mapstructure:"keepalive_interval"`
	// KeepaliveTimeout is how long to wait for a keepalive ping ACK.
	// Required to be > 0 when KeepaliveInterval is enabled.
	KeepaliveTimeout time.Duration `mapstructure:"keepalive_timeout"`
	// RateLimitRPS is the per-client token-bucket refill rate for unary CP RPCs.
	// <= 0 disables rate limiting.
	RateLimitRPS float64 `mapstructure:"rate_limit_rps"`
	// RateLimitBurst is the token-bucket capacity. Required to be >= 1 when
	// RateLimitRPS is enabled.
	RateLimitBurst int `mapstructure:"rate_limit_burst"`
}

// DialTarget returns DialAddr if set, otherwise ListenAddr.
func (c GRPCConfig) DialTarget() string {
	if c.DialAddr != "" {
		return c.DialAddr
	}
	return c.ListenAddr
}

type RestConfig struct {
	ListenAddr string `mapstructure:"listen_addr"`
}

type AuthConfig struct {
	// TokenTTL is the default lifetime of login-issued tokens.
	TokenTTL time.Duration `mapstructure:"token_ttl"`
	// RegistrationTokenTTL is the default lifetime of single-use agent
	// registration tokens when --ttl is omitted.
	RegistrationTokenTTL time.Duration `mapstructure:"registration_token_ttl"`
	// AgentTokenTTL is the sliding window granted at agent token issue and
	// every heartbeat.
	AgentTokenTTL time.Duration `mapstructure:"agent_token_ttl"`
}

type AgentConfig struct {
	// Name is the unique agent identity registered with the CP (DP).
	// Empty defaults to the host name.
	Name string `mapstructure:"name"`
	// Labels are operator-declared labels advertised at Register (DP).
	Labels map[string]string `mapstructure:"labels"`
	// StatePath is the DP credentials file (agent_id + agent_token).
	StatePath string `mapstructure:"state_path"`
	// HeartbeatInterval is how often the DP sends Heartbeat RPCs.
	HeartbeatInterval time.Duration `mapstructure:"heartbeat_interval"`
	// HeartbeatThreshold is the active/disconnected boundary used when
	// deriving agent status at read time (≈3× heartbeat interval).
	HeartbeatThreshold time.Duration `mapstructure:"heartbeat_threshold"`
	// RegistrationToken is the single-use bootstrap secret for first Register.
	// Prefer PGWAY_AGENT_REGISTRATION_TOKEN; never commit this value.
	RegistrationToken string `mapstructure:"registration_token"`
}

type ProxyConfig struct {
	// MaxRequestBodyBytes caps non-CONNECT proxy request bodies.
	// Accepts bare integers or human sizes (e.g. "10MiB"). 0 disables the limit.
	MaxRequestBodyBytes ByteSize `mapstructure:"max_request_body_bytes"`
	// MaxIdleConns is the global idle connection limit across all hosts
	// on each per-proxy http.Transport. 0 means unlimited.
	MaxIdleConns int `mapstructure:"max_idle_conns"`
	// MaxIdleConnsPerHost caps idle connections per upstream host.
	// Must be > 0 (Go treats 0 as DefaultMaxIdleConnsPerHost=2).
	MaxIdleConnsPerHost int `mapstructure:"max_idle_conns_per_host"`
	// IdleConnTimeout is how long an idle connection stays in the pool.
	IdleConnTimeout time.Duration `mapstructure:"idle_conn_timeout"`
	// DialTimeout is the TCP dial timeout for upstream proxy connections.
	DialTimeout time.Duration  `mapstructure:"dial_timeout"`
	DNSCache    DNSCacheConfig `mapstructure:"dns_cache"`
}

// DNSCacheConfig controls fixed-TTL caching of upstream proxy hostname lookups.
type DNSCacheConfig struct {
	// Enabled turns on the DNS cache. When false, dials use the default resolver.
	Enabled bool `mapstructure:"enabled"`
	// TTL is how long successful lookups are reused. Must be > 0 when Enabled.
	TTL time.Duration `mapstructure:"ttl"`
}

// DataplaneConfig controls data-plane event handling and related process knobs.
type DataplaneConfig struct {
	// EventCoalesceWindow is the setTimeout-style flush delay for change events.
	// <= 0 disables coalescing (immediate dispatch).
	EventCoalesceWindow time.Duration `mapstructure:"event_coalesce_window"`
	// EventCoalesceMaxBuffer triggers an early flush when this many events
	// accumulate for one coalesce key before the window elapses.
	// Required to be >= 1 when EventCoalesceWindow is enabled.
	EventCoalesceMaxBuffer int `mapstructure:"event_coalesce_max_buffer"`

	// CPDisconnectStrategy is fail_open | fail_closed for distributed DP when
	// the control plane is unreachable. All-in-one ignores this (in-process CP).
	CPDisconnectStrategy string `mapstructure:"cp_disconnect_strategy"`
	// CPDisconnectUnreachableThreshold is how long both heartbeat and watch
	// proofs must be stale before entering unreachable.
	CPDisconnectUnreachableThreshold time.Duration `mapstructure:"cp_disconnect_unreachable_threshold"`
	// CPDisconnectRecoverThreshold is hysteresis before leaving unreachable
	// once proofs are fresh again. 0 = immediate recover.
	CPDisconnectRecoverThreshold time.Duration `mapstructure:"cp_disconnect_recover_threshold"`
	// EventResyncInterval is how often all-in-one pgway runs a full Resync
	// (Bootstrap + listener reconcile) as a missed-event safety net.
	// <= 0 disables. Ignored by pgway-dp (Watch reconnect is primary).
	EventResyncInterval time.Duration `mapstructure:"event_resync_interval"`
}

var c *Config

func Load(path string) error {
	if path != "" {
		viper.SetConfigFile(path)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("toml")
		viper.AddConfigPath("/etc/pgway/")
		viper.AddConfigPath("$HOME/.pgway")
		viper.AddConfigPath(".")
	}

	viper.SetDefault("log_level", "info")
	viper.SetDefault("token", "")
	viper.SetDefault("badger.path", "/var/pgway/lib")
	viper.SetDefault("badger.gc_interval", 5*time.Minute)
	viper.SetDefault("grpc.listen_addr", ":9090")
	viper.SetDefault("grpc.keepalive_interval", time.Minute)
	viper.SetDefault("grpc.keepalive_timeout", 20*time.Second)
	viper.SetDefault("grpc.rate_limit_rps", 100.0)
	viper.SetDefault("grpc.rate_limit_burst", 200)
	viper.SetDefault("rest.listen_addr", ":8081")
	viper.SetDefault("auth.token_ttl", 720*time.Hour)
	viper.SetDefault("auth.registration_token_ttl", 24*time.Hour)
	viper.SetDefault("auth.agent_token_ttl", 168*time.Hour)
	viper.SetDefault("agent.name", "")
	viper.SetDefault("agent.labels", map[string]string{})
	viper.SetDefault("agent.state_path", "/var/lib/pgway/agent.json")
	viper.SetDefault("agent.heartbeat_interval", 10*time.Second)
	viper.SetDefault("agent.heartbeat_threshold", 30*time.Second)
	viper.SetDefault("agent.registration_token", "")
	viper.SetDefault("proxy.max_request_body_bytes", "10MiB")
	viper.SetDefault("proxy.max_idle_conns", 1024)
	viper.SetDefault("proxy.max_idle_conns_per_host", 128)
	viper.SetDefault("proxy.idle_conn_timeout", 90*time.Second)
	viper.SetDefault("proxy.dial_timeout", 10*time.Second)
	viper.SetDefault("proxy.dns_cache.enabled", true)
	viper.SetDefault("proxy.dns_cache.ttl", 5*time.Minute)
	viper.SetDefault("dataplane.event_coalesce_window", 100*time.Millisecond)
	viper.SetDefault("dataplane.event_coalesce_max_buffer", 256)
	viper.SetDefault("dataplane.cp_disconnect_strategy", "fail_open")
	viper.SetDefault("dataplane.cp_disconnect_unreachable_threshold", 30*time.Second)
	viper.SetDefault("dataplane.cp_disconnect_recover_threshold", time.Duration(0))
	viper.SetDefault("dataplane.event_resync_interval", 5*time.Minute)

	// PGWAY_TOKEN, PGWAY_BADGER_PATH, PGWAY_AGENT_REGISTRATION_TOKEN, etc.
	viper.SetEnvPrefix("pgway")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	var cfg Config
	// Custom DecodeHook replaces viper defaults — keep Duration/slice hooks.
	if err := viper.Unmarshal(&cfg, viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		byteSizeDecodeHook(),
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToSliceHookFunc(","),
		mapstructure.TextUnmarshallerHookFunc(),
	))); err != nil {
		return err
	}

	if cfg.Agent.Name == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return err
		}
		cfg.Agent.Name = hostname
	}
	if cfg.Agent.Labels == nil {
		cfg.Agent.Labels = map[string]string{}
	}
	if cfg.GRPC.KeepaliveInterval > 0 && cfg.GRPC.KeepaliveTimeout <= 0 {
		return fmt.Errorf("grpc.keepalive_timeout must be > 0 when grpc.keepalive_interval is enabled")
	}
	if cfg.GRPC.RateLimitRPS > 0 && cfg.GRPC.RateLimitBurst < 1 {
		return fmt.Errorf("grpc.rate_limit_burst must be >= 1 when grpc.rate_limit_rps is enabled")
	}
	if cfg.Proxy.MaxIdleConns < 0 {
		return fmt.Errorf("proxy.max_idle_conns must be >= 0")
	}
	if cfg.Proxy.MaxIdleConnsPerHost <= 0 {
		return fmt.Errorf("proxy.max_idle_conns_per_host must be > 0")
	}
	if cfg.Proxy.IdleConnTimeout < 0 {
		return fmt.Errorf("proxy.idle_conn_timeout must be >= 0")
	}
	if cfg.Proxy.DialTimeout <= 0 {
		return fmt.Errorf("proxy.dial_timeout must be > 0")
	}
	if cfg.Proxy.DNSCache.Enabled && cfg.Proxy.DNSCache.TTL <= 0 {
		return fmt.Errorf("proxy.dns_cache.ttl must be > 0 when proxy.dns_cache.enabled is true")
	}
	if cfg.Dataplane.EventCoalesceWindow > 0 && cfg.Dataplane.EventCoalesceMaxBuffer < 1 {
		return fmt.Errorf("dataplane.event_coalesce_max_buffer must be >= 1 when dataplane.event_coalesce_window is enabled")
	}
	switch cfg.Dataplane.CPDisconnectStrategy {
	case "fail_open", "fail_closed":
	default:
		return fmt.Errorf("dataplane.cp_disconnect_strategy must be fail_open or fail_closed")
	}
	if cfg.Dataplane.CPDisconnectUnreachableThreshold < 0 {
		return fmt.Errorf("dataplane.cp_disconnect_unreachable_threshold must be >= 0")
	}
	if cfg.Dataplane.CPDisconnectRecoverThreshold < 0 {
		return fmt.Errorf("dataplane.cp_disconnect_recover_threshold must be >= 0")
	}

	c = &cfg
	return nil
}

func Get() *Config {
	return c
}

func byteSizeDecodeHook() mapstructure.DecodeHookFunc {
	return func(from, to reflect.Type, data any) (any, error) {
		if to != reflect.TypeFor[ByteSize]() {
			return data, nil
		}
		switch v := data.(type) {
		case string:
			return ParseByteSize(v)
		case int:
			return ByteSize(v), nil
		case int64:
			return ByteSize(v), nil
		case uint64:
			return ByteSize(v), nil
		case float64:
			return ByteSize(v), nil
		case ByteSize:
			return v, nil
		default:
			return nil, fmt.Errorf("byte size: unsupported type %T", data)
		}
	}
}
