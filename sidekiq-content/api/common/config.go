package common

import (
	"time"

	"github.com/spf13/viper"
)

// Config api configuration
type Config struct {
	Port            int           `mapstructure:"port"`
	ProxyCount      int           `mapstructure:"proxyCount"`
	MaxContentSize  int64         `mapstructure:"maxContentSize"`
	ReadTimeout     time.Duration `mapstructure:"readTimeout"`
	WriteTimeout    time.Duration `mapstructure:"writeTimeout"`
	CloseTimeout    time.Duration `mapstructure:"closeTimeout"`
	AuthCookieName  string        `mapstructure:"authCookieName"`
	TokenExpiration time.Duration `mapstructure:"tokenExpiration"`
	SignUpAuthName  string        `mapstructure:"signUpAuthName"`
}

// InitConfig initialize api configuration
func InitConfig() (*Config, error) {
	config := &Config{}
	subv := viper.Sub("api")
	err := subv.Unmarshal(&config)
	return config, err
}
