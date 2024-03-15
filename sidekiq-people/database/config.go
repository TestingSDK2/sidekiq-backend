package database

import (
	"time"

	"github.com/spf13/viper"
)

// Config database configuration
type Config struct {
	Master  *DBConfig `mapstructure:"master"`
	Replica *DBConfig `mapstructure:"replica"`
}

// DBConfig configuration for db
type DBConfig struct {
	Type         string        `mapstructure:"type"`
	Host         string        `mapstructure:"host"`
	Port         string        `mapstructure:"port"`
	DBName       string        `mapstructure:"dbName"`
	UserName     string        `mapstructure:"userName"`
	Password     string        `mapstructure:"password"`
	MaxLifetime  time.Duration `mapstructure:"maxLifetime"`
	MaxOpenConns int           `mapstructure:"maxOpenConns"`
	MaxIdleConns int           `mapstructure:"maxIdleConns"`
}

// InitConfig initialize app configuration
func InitConfig() (*Config, error) {
	config := &Config{}
	subv := viper.Sub("database")
	err := subv.Unmarshal(&config)
	return config, err
}
