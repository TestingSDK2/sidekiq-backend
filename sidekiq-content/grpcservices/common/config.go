package contentgrpc

import (
	"github.com/spf13/viper"
)

type GrpcInfoConfig struct {
	Host string `mapstructure:"host"`
	Port string `mapstructure:"port"`
}

type GrpcConfig struct {
	Search         *GrpcInfoConfig `mapstructure:"search"`
	Content        *GrpcInfoConfig `mapstructure:"content"`
	People         *GrpcInfoConfig `mapstructure:"people"`
	Authentication *GrpcInfoConfig `mapstructure:"authentication"`
	Notification   *GrpcInfoConfig `mapstructure:"notification"`
}

// InitConfig initialize grpc configuration
func InitConfig() (*GrpcConfig, error) {
	config := &GrpcConfig{}
	subv := viper.Sub("grpc")
	err := subv.Unmarshal(&config)
	return config, err
}
