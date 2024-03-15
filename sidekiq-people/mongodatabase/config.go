package mongodatabase

import (
	"github.com/spf13/viper"
)

// DBConfig configuration for db
type DBConfig struct {
	Host   string `mapstructure:"host"`
	DBName string `mapstructure:"dbName"`
}

// InitConfig initialize app configuration
func InitConfig() (*DBConfig, error) {
	dbconfig := &DBConfig{}
	subv := viper.Sub("mongodatabase")
	err := subv.Unmarshal(&dbconfig)
	return dbconfig, err
}
