package thing

import (
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/api/common"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/board"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/collection"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/file"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/note"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/model"

	// "github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/notification"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/post"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/profile"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/recent"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/storage"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/task"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/thing"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/cache"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/mongodatabase"
)

type api struct {
	config *common.Config
	// clientMgr           *notification.ClientManager
	thingService        thing.Service
	profileService      profile.Service
	boardService        board.Service
	taskService         task.Service
	noteService         note.Service
	fileService         file.Service
	storageService      storage.Service
	postService         post.Service
	recentThingsService recent.Service
	db                  *mongodatabase.DBConfig
	// notificationService notification.Service
	cache             *cache.Cache
	collectionService collection.Service
	repos             *model.Repos
}

func New(conf *common.Config, thingService thing.Service, profileService profile.Service, boardService board.Service, taskService task.Service, noteService note.Service, fileService file.Service, storageService storage.Service,
	recentThingsService recent.Service, db *mongodatabase.DBConfig, repos *model.Repos, postService post.Service,
	// notificationService notification.Service,
	collectionService collection.Service) *api {
	return &api{
		config: conf,
		// clientMgr:           clientMgr,
		thingService:        thingService,
		profileService:      profileService,
		boardService:        boardService,
		taskService:         taskService,
		noteService:         noteService,
		fileService:         fileService,
		storageService:      storageService,
		recentThingsService: recentThingsService,
		postService:         postService,
		db:                  db,
		cache:               repos.Cache,
		// notificationService: notificationService,
		collectionService: collectionService,
		repos:             repos,
	}
}
