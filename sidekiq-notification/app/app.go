package app

import (
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/ProImaging/sidekiq-backend/sidekiq-notification/app/config"
	"github.com/ProImaging/sidekiq-backend/sidekiq-notification/app/notification"
	"github.com/ProImaging/sidekiq-backend/sidekiq-notification/model"

	"github.com/ProImaging/sidekiq-backend/sidekiq-notification/cache"

	"github.com/ProImaging/sidekiq-backend/sidekiq-notification/database"

	repo "github.com/ProImaging/sidekiq-backend/sidekiq-notification/model"
	"github.com/ProImaging/sidekiq-backend/sidekiq-notification/mongodatabase"
	accountProtobuf "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"

	realtimeProtobuf "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-realtime/v1"
)

// App our application
type App struct {
	Config              *config.Config
	Repos               *repo.Repos
	NotificationService notification.Service
}

// NewContext create new request context
func (a *App) NewContext() *Context {
	return &Context{
		Logger: logrus.StandardLogger(),
	}
}

// New create a new app
func New(peopleServiceClient, realtimeServiceClient *grpc.ClientConn) (app *App, err error) {

	appConf, err := config.InitConfig()
	if err != nil {
		return nil, err
	}

	dbConf, err := database.InitConfig()
	if err != nil {
		return nil, err
	}

	cacheConf, err := cache.InitConfig()
	if err != nil {
		return nil, err
	}

	masterDB, err := database.New(dbConf.Master)
	if err != nil {
		return nil, err
	}

	replicaDB, err := database.New(dbConf.Replica)
	if err != nil {
		return nil, err
	}

	mongoDBConf, err := mongodatabase.InitConfig()
	if err != nil {
		return nil, err
	}

	repos := &model.Repos{
		MasterDB:              masterDB,
		ReplicaDB:             replicaDB,
		Cache:                 cache.New(cacheConf),
		MongoDB:               mongoDBConf,
		PeopleServiceClient:   accountProtobuf.NewAccountServiceClient(peopleServiceClient),
		RealtimeServiceClient: realtimeProtobuf.NewDeliveryServiceClient(realtimeServiceClient),
	}

	return &App{
		Config:              appConf,
		Repos:               repos,
		NotificationService: notification.NewService(repos, appConf),
	}, nil
}

// Close closes application handles and connections
func (a *App) Close() {
	logrus.Info("Closing Connection to database")

	err := a.Repos.MasterDB.Close()
	if err != nil {
		logrus.Error("unable to close connection to master database", err)
	}
	err = a.Repos.ReplicaDB.Close()
	if err != nil {
		logrus.Error("unable to close connection to replica database", err)
	}
	err = a.Repos.Cache.Close()
	if err != nil {
		logrus.Error("unable to close connection to cache", err)
	}

	return
}

// ValidationError error when inputs are invalid
type ValidationError struct {
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return e.Message
}

// UserError when user is disallowed from resource
type UserError struct {
	Message    string `json:"message"`
	StatusCode int    `json:"-"`
}

func (e *UserError) Error() string {
	return e.Message
}
