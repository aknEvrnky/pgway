package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	EntryPoints []EntrypointConfig `mapstructure:"entry_points"`
	Flows       []FlowConfig       `mapstructure:"flows"`
	Routers     []RouterConfig     `mapstructure:"routers"`
}

type EntrypointConfig struct {
	Id       string `mapstructure:"id"`
	Title    string `mapstructure:"title"`
	Protocol string `mapstructure:"protocol"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Flow     string `mapstructure:"flow"`
}

type RouterConfig struct {
	Id          string             `mapstructure:"id"`
	Title       string             `mapstructure:"title"`
	Description string             `mapstructure:"description,omitempty"`
	Rules       []RouterRuleConfig `mapstructure:"rules"`
}

type RouterRuleConfig struct {
	Id     string            `mapstructure:"id"`
	Match  RouterMatchConfig `mapstructure:"match"`
	Target string            `mapstructure:"target"`
}

type RouterConditionConfig struct {
	Type  string `mapstructure:"type"`
	Value string `mapstructure:"value"`
}

type RouterMatchConfig struct {
	All []RouterConditionConfig `mapstructure:"all,omitempty"`
	Any []RouterConditionConfig `mapstructure:"any,omitempty"`
	Not *RouterConditionConfig  `mapstructure:"not,omitempty"`

	Type  string `mapstructure:"type,omitempty"`
	Value string `mapstructure:"value,omitempty"`
}

type FlowConfig struct {
	Id         string `mapstructure:"string"`
	RouterId   string `mapstructure:"router,omitempty"`
	BalancerId string `mapstructure:"balancer"`
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
