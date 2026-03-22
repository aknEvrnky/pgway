package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	BadgerPath     string `mapstructure:"badger_path"`
	GrpcListenAddr string `mapstructure:"grpc_listen_addr"`
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
	viper.SetDefault("grpc_listen_addr", "9090")

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
