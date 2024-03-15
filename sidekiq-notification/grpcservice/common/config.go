package common

import (
	"github.com/spf13/viper"
)

type GrpcInfoConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type GrpcConfig struct {
	Authentication *GrpcInfoConfig `mapstructure:"authentication"`
	People         *GrpcInfoConfig `mapstructure:"people"`
	Notification   *GrpcInfoConfig `mapstructure:"notification"`
	Content        *GrpcInfoConfig `mapstructure:"content"`
	Realtime       *GrpcInfoConfig `mapstructure:"realtime"`
}

// InitConfig initialize api configuration
func InitConfig() (*GrpcConfig, error) {
	config := &GrpcConfig{}
	subv := viper.Sub("grpc")
	err := subv.Unmarshal(&config)
	return config, err
}
