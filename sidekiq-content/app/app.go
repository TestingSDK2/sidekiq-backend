package app

import (
	"github.com/sirupsen/logrus"

	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/board"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/collection"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/config"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/email"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/file"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/message"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/note"
	repo "github.com/ProImaging/sidekiq-backend/sidekiq-content/model"
	"google.golang.org/grpc"

	// "github.com/ProImaging/sidekiq-backend/sidekiq-content/app/notification"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/post"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/profile"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/recent"
	appStorage "github.com/ProImaging/sidekiq-backend/sidekiq-content/app/storage"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/task"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/thing"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/thingactivity"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/user"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/cache"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/database"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/mongodatabase"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/storage"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/thingsqs"
	authrpc "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-auth-server/v1"
	notfrpc "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-notification/v1"
	peoplerpc "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"
	searchrpc "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-search/v1"
)

// App our application
type App struct {
	Config *config.Config
	Repos  *repo.Repos
	// ClientManager        *notification.ClientManager
	UserService          user.Service
	StorageService       appStorage.Service
	BoardService         board.Service
	PostService          post.Service
	FileService          file.Service
	NoteService          note.Service
	TaskService          task.Service
	RecentThingsService  recent.Service
	ProfileService       profile.Service
	CollectionService    collection.Service
	ThingService         thing.Service
	EmailService         email.Service
	MessageService       message.Service
	ThingActivityService thingactivity.Service
	// NotifcationService   notification.Service
}

// NewContext create new request context
func (a *App) NewContext() *Context {
	return &Context{
		Logger:         logrus.StandardLogger(),
		UserService:    a.UserService,
		ProfileService: a.ProfileService,
		StorageService: a.StorageService,
	}
}

// New create a new app
func New(authGrpcClient, searchgrpcClient, peoplegrpcClient, notfgrpcClient *grpc.ClientConn) (app *App, err error) {
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

	sqsConn, err := thingsqs.New()
	if err != nil {
		return nil, err
	}

	repos := &repo.Repos{
		MasterDB:                      masterDB,
		ReplicaDB:                     replicaDB,
		Cache:                         cache.New(cacheConf),
		Storage:                       fileStore,
		TmpStorage:                    tmpStore,
		MongoDB:                       mongoDBConf,
		ThingSQS:                      sqsConn,
		AuthServiceClient:             authrpc.NewAuthServiceClient(authGrpcClient),
		SearchGrpcServiceClient:       searchrpc.NewSearchServiceClient(searchgrpcClient),
		PeopleGrpcServiceClient:       peoplerpc.NewAccountServiceClient(peoplegrpcClient),
		NotificationGrpcServiceClient: notfrpc.NewNotificationServiceClient(notfgrpcClient),
	}

	return &App{
		Config: appConf,
		Repos:  repos,
		// ClientManager:        notification.NewClientManager(repos.Cache),
		UserService:          user.NewService(repos, appConf),
		StorageService:       appStorage.NewService(repos, appConf),
		BoardService:         board.NewService(repos, appConf),
		PostService:          post.NewService(repos, appConf),
		FileService:          file.NewService(repos, appConf),
		NoteService:          note.NewService(repos, appConf),
		TaskService:          task.NewService(repos, appConf),
		RecentThingsService:  recent.NewService(repos, appConf),
		ProfileService:       profile.NewService(repos, appConf),
		CollectionService:    collection.NewService(repos, appConf),
		ThingService:         thing.NewService(repos, appConf),
		EmailService:         email.NewService(),
		MessageService:       message.NewService(repos, appConf),
		ThingActivityService: thingactivity.NewService(repos, appConf),
		// NotifcationService:   notification.NewService(repos, appConf),
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
