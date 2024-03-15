package storage

import (
	"errors"

	"github.com/ProImaging/sidekiq-backend/sidekiq-models/model"

	"github.com/spf13/viper"
)

// New create new DB
func New(conf *Config) (model.FileStorage, error) {
	var s model.FileStorage
	var err error
	switch conf.Type {
	case "local":
		s, err = NewLocalStorage(conf.Path)
	case "wasabi":
		s, err = NewWasabiStorage(conf.Path, conf.Region, conf.AccessKeyID, conf.SecretAccessKey)
	}
	if err != nil {
		return nil, errors.New("failed to create storage adapter")
	}
	return s, nil
}

func NewTmp() (model.FileStorage, error) {
	config := &Config{}
	subv := viper.Sub("tmpFileStorage")
	err := subv.Unmarshal(&config)
	if err != nil {
		return nil, err
	}
	return New(config)
}
