package config

import (
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	BadgerPath     string `mapstructure:"badger_path"`
	GrpcListenAddr string `mapstructure:"grpc_listen_addr"`
	RestListenAddr string `mapstructure:"rest_listen_addr"`
	// Token authenticates outgoing control-plane calls (pgctl, pgway-dp).
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

	// PGWAY_TOKEN etc. override file values
	viper.SetEnvPrefix("pgway")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return err
	}

	c = &cfg
	return nil
}

func Get() *Config {
	return c
}
