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
