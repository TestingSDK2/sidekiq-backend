package config

import (
	"github.com/spf13/viper"
)

// Config app configuration
type Config struct {
	SecretKey       string `mapstructure:"secretKey"`
	JWTKey          string `mapstructure:"jwtKey"`
	VapidPublicKey  string `mapstructure:"vapidPublicKey"`
	VapidPrivateKey string `mapstructure:"vapidPrivateKey"`
}

// InitConfig initialize app configuration
func InitConfig() (*Config, error) {
	config := &Config{}
	subv := viper.Sub("app")
	err := subv.Unmarshal(&config)
	return config, err
}
