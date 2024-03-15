package app

import (
	"github.com/ProImaging/sidekiq-backend/sidekiq-people/app/profile"
	appStorage "github.com/ProImaging/sidekiq-backend/sidekiq-people/app/storage"
	"github.com/ProImaging/sidekiq-backend/sidekiq-people/model"
	"github.com/ProImaging/sidekiq-backend/sidekiq-people/storage"
	"google.golang.org/grpc"

	"github.com/sirupsen/logrus"

	"github.com/ProImaging/sidekiq-backend/sidekiq-people/app/account"
	"github.com/ProImaging/sidekiq-backend/sidekiq-people/app/config"

	"github.com/ProImaging/sidekiq-backend/sidekiq-people/cache"

	"github.com/ProImaging/sidekiq-backend/sidekiq-people/database"

	repo "github.com/ProImaging/sidekiq-backend/sidekiq-people/model"

	"github.com/ProImaging/sidekiq-backend/sidekiq-people/mongodatabase"
	authProtobuf "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-auth-server/v1"
	notiProtobuf "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-notification/v1"
)

// App our application
type App struct {
	Config         *config.Config
	Repos          *repo.Repos
	AccountService account.Service
	ProfileService profile.Service
	StorageService appStorage.Service
}

// NewContext create new request context
func (a *App) NewContext() *Context {
	return &Context{
		Logger:         logrus.StandardLogger(),
		StorageService: a.StorageService,
	}
}

// New create a new app
func New(authGrpcClient, notfGrpcClient *grpc.ClientConn) (app *App, err error) {
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

	storageConf, err := storage.InitConfig()
	if err != nil {
		return nil, err
	}

	fileStore, err := storage.New(storageConf)
	if err != nil {
		return nil, err
	}

	tmpStore, err := storage.NewTmp()
	if err != nil {
		return nil, err
	}

	repos := &model.Repos{
		MasterDB:                  masterDB,
		ReplicaDB:                 replicaDB,
		Storage:                   fileStore,
		TmpStorage:                tmpStore,
		Cache:                     cache.New(cacheConf),
		MongoDB:                   mongoDBConf,
		AuthServiceClient:         authProtobuf.NewAuthServiceClient(authGrpcClient),
		NotificationServiceClient: notiProtobuf.NewNotificationServiceClient(notfGrpcClient),
	}

	return &App{
		Config:         appConf,
		Repos:          repos,
		AccountService: account.NewService(repos, appConf),
		ProfileService: profile.NewService(repos, appConf),
		StorageService: appStorage.NewService(repos, appConf),
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
