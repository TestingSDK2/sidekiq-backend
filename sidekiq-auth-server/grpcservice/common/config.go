package common

import "github.com/spf13/viper"

type GrpcInfoConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type GrpcConfig struct {
	Authentication *GrpcInfoConfig `mapstructure:"authentication"`
	People         *GrpcInfoConfig `mapstructure:"people"`
}

// InitConfig initialize grpc configuration
func InitConfig() (*GrpcConfig, error) {
	config := &GrpcConfig{}
	subv := viper.Sub("grpc")
	err := subv.Unmarshal(&config)
	return config, err
}
