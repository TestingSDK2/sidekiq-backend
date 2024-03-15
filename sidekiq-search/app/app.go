package app

import (
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/ProImaging/sidekiq-backend/sidekiq-search/app/config"
	"github.com/ProImaging/sidekiq-backend/sidekiq-search/app/email"

	authrpc "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-auth-server/v1"
	contentrpc "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-content/v1"
	peoplerpc "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"

	"github.com/ProImaging/sidekiq-backend/sidekiq-search/app/search"

	"github.com/ProImaging/sidekiq-backend/sidekiq-search/cache"
	"github.com/ProImaging/sidekiq-backend/sidekiq-search/database"
	repo "github.com/ProImaging/sidekiq-backend/sidekiq-search/model"
	"github.com/ProImaging/sidekiq-backend/sidekiq-search/mongodatabase"
	"github.com/ProImaging/sidekiq-backend/sidekiq-search/thingsqs"
)

// App our application
type App struct {
	Config *config.Config
	Repos  *repo.Repos
	// torageService appStorage.Service
	SearchService search.Service
	EmailService  email.Service
}

// NewContext create new request context
func (a *App) NewContext() *Context {
	return &Context{
		Logger: logrus.StandardLogger(),
	}
}

// New create a new app
func New(contentGrpcClient, peopleGrpcClient, authGrpcClient *grpc.ClientConn) (app *App, err error) {
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

	sqsConn, err := thingsqs.New()
	if err != nil {
		return nil, err
	}

	repos := &repo.Repos{
		MasterDB:                 masterDB,
		ReplicaDB:                replicaDB,
		Cache:                    cache.New(cacheConf),
		MongoDB:                  mongoDBConf,
		ThingSQS:                 sqsConn,
		ContentGrpcServiceClient: contentrpc.NewBoardServiceClient(contentGrpcClient),
		PeopleServiceClient:      peoplerpc.NewAccountServiceClient(peopleGrpcClient),
		AuthServiceClient:        authrpc.NewAuthServiceClient(authGrpcClient),
	}

	return &App{
		Config:        appConf,
		Repos:         repos,
		SearchService: search.NewService(repos, appConf),
		EmailService:  email.NewService(),
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
