package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	PoolPath   string `mapstructure:"pool_path"`
	BadgerPath string `mapstructure:"badger_path"`
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

	//viper.SetDefault("server.port", 50100)

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
