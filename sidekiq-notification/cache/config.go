package cache

import (
	"github.com/spf13/viper"
)

// Config redis cache configuration
type Config struct {
	Type     string `mapstructure:"type"`
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Password string `mapstructure:"password"`
}

// InitConfig initialize app configuration
func InitConfig() (*Config, error) {
	config := &Config{}
	subv := viper.Sub("cache")
	err := subv.Unmarshal(&config)
	return config, err
}
