package config

import (
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

type Config struct {
	BadgerPath     string `mapstructure:"badger_path"`
	GrpcListenAddr string `mapstructure:"grpc_listen_addr"`
	RestListenAddr string `mapstructure:"rest_listen_addr"`
	// Token authenticates outgoing control-plane calls (pgctl).
	// pgway-dp uses an agent token from the state file after registration.
	Token string `mapstructure:"token"`
	// TokenTTL is the default lifetime of login-issued tokens.
	TokenTTL time.Duration `mapstructure:"token_ttl"`
	// RegistrationTokenTTL is the default lifetime of single-use agent
	// registration tokens when --ttl is omitted.
	RegistrationTokenTTL time.Duration `mapstructure:"registration_token_ttl"`
	// AgentTokenTTL is the sliding window granted at agent token issue and
	// every heartbeat.
	AgentTokenTTL time.Duration `mapstructure:"agent_token_ttl"`
	// AgentHeartbeatThreshold is the active/disconnected boundary used when
	// deriving agent status at read time (≈3× heartbeat interval).
	AgentHeartbeatThreshold time.Duration `mapstructure:"agent_heartbeat_threshold"`
	// AgentName is the unique agent identity registered with the CP (DP).
	// Empty defaults to the host name.
	AgentName string `mapstructure:"agent_name"`
	// AgentLabels are operator-declared labels advertised at Register (DP).
	AgentLabels map[string]string `mapstructure:"agent_labels"`
	// AgentStatePath is the DP credentials file (agent_id + agent_token).
	AgentStatePath string `mapstructure:"agent_state_path"`
	// HeartbeatInterval is how often the DP sends Heartbeat RPCs.
	HeartbeatInterval time.Duration `mapstructure:"heartbeat_interval"`
	// RegistrationToken is the single-use bootstrap secret for first Register.
	// Prefer PGWAY_REGISTRATION_TOKEN; never commit this value.
	RegistrationToken string `mapstructure:"registration_token"`
	// MaxRequestBodyBytes caps non-CONNECT proxy request bodies.
	// Accepts bare integers or human sizes (e.g. "10MiB"). 0 disables the limit.
	MaxRequestBodyBytes ByteSize `mapstructure:"max_request_body_bytes"`
	// LogLevel sets the global zap log level (debug|info|warn|error).
	LogLevel string `mapstructure:"log_level"`
}

var c *Config

func Load(path string) error {
	if path != "" {
		viper.SetConfigFile(path)
	} else {
		viper.SetConfigName("config")
		viper.AddConfigPath("/etc/pgway/")
		viper.AddConfigPath("$HOME/.pgway")
		viper.AddConfigPath(".")
	}

	viper.SetDefault("badger_path", "/var/pgway/lib")
	viper.SetDefault("grpc_listen_addr", ":9090")
	viper.SetDefault("rest_listen_addr", ":8081")
	viper.SetDefault("token", "")
	viper.SetDefault("token_ttl", 720*time.Hour)
	viper.SetDefault("registration_token_ttl", 24*time.Hour)
	viper.SetDefault("agent_token_ttl", 168*time.Hour)
	viper.SetDefault("agent_heartbeat_threshold", 30*time.Second)
	viper.SetDefault("agent_name", "")
	viper.SetDefault("agent_labels", map[string]string{})
	viper.SetDefault("agent_state_path", "/var/lib/pgway/agent.json")
	viper.SetDefault("heartbeat_interval", 10*time.Second)
	viper.SetDefault("registration_token", "")
	viper.SetDefault("max_request_body_bytes", "10MiB")
	viper.SetDefault("log_level", "info")

	// PGWAY_TOKEN etc. override file values
	viper.SetEnvPrefix("pgway")
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

	if cfg.AgentName == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return err
		}
		cfg.AgentName = hostname
	}
	if cfg.AgentLabels == nil {
		cfg.AgentLabels = map[string]string{}
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
