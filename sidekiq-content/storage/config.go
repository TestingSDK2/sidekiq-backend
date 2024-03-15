package storage

import "github.com/spf13/viper"

// Config - file storage configuration
type Config struct {
	Type            string `mapstructure:"type"`
	Path            string `mapstructure:"path"`
	Region          string `mapstructure:"region"`
	AccessKeyID     string `mapstructure:"accessKeyID"`
	SecretAccessKey string `mapstructure:"secretAccessKey"`
}

// InitConfig initialize app configuration
func InitConfig() (*Config, error) {
	config := &Config{}
	subv := viper.Sub("fileStorage")
	err := subv.Unmarshal(&config)
	return config, err
}
