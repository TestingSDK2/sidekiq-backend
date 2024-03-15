package env

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	EnvFile = "ENV_FILE"
)

type Env struct {
	AppEnv               string `mapstructure:"APP_ENV" default:"dev"`
	AppPort              string `mapstructure:"APP_PORT"`
	MongoDbConnectionUrl string `mapstructure:"MONGODB_CONNECTION_URL" required:"true"`
	DbName               string `mapstructure:"DB_NAME"`

	RealtimeGrpcHost string `mapstructure:"REALTIME_GRPC_HOST"`
	AuthGrpcHost     string `mapstructure:"AUTH_GRPC_HOST"`
	BoardGrpcHost    string `mapstructure:"BOARD_GRPC_HOST"`
	AuthCookieName   string `mapstructure:"authCookieName"`
}

// TODO add validator for required env
func NewEnv(fileName string) (*Env, error) {
	env := Env{}

	viper.SetConfigFile(fileName)

	err := viper.ReadInConfig()
	if err != nil {
		logrus.Error("Env file not found")
		return nil, err
	}

	err = viper.Unmarshal(&env)
	if err != nil {
		logrus.Errorf("Unable to load environment from: %s", fileName)
		return nil, err
	}

	return &env, nil
}
