package storage

import (
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/api/common"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/board"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/collection"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/file"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/note"
	model "github.com/TestingSDK2/sidekiq-backend/sidekiq-models"

	// "github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/notification"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/post"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/profile"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/recent"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/storage"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/task"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/thing"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/thingactivity"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/cache"
	repo "github.com/TestingSDK2/sidekiq-backend/sidekiq-content/model"
)

type api struct {
	config *common.Config
	// clientMgr           *notification.ClientManager
	fileService         file.Service
	storageService      storage.Service
	taskService         task.Service
	noteService         note.Service
	profileService      profile.Service
	boardService        board.Service
	collectionService   collection.Service
	recentThingsService recent.Service
	fileStore           model.FileStorage
	thingActivity       thingactivity.Service
	tmpFileStore        model.FileStorage
	postService         post.Service
	cache               *cache.Cache
	thingService        thing.Service
	repos               *repo.Repos
}

// New creates a new storage api
func New(conf *common.Config, fileService file.Service, storageService storage.Service,
	profileService profile.Service, recentThingsService recent.Service, boardService board.Service, collectionService collection.Service, taskService task.Service,
	noteService note.Service, thingActivity thingactivity.Service, repos *repo.Repos, postService post.Service, thingService thing.Service,
) *api {
	return &api{
		config: conf,
		// clientMgr:           clientMgr,
		fileService:         fileService,
		storageService:      storageService,
		taskService:         taskService,
		noteService:         noteService,
		profileService:      profileService,
		boardService:        boardService,
		postService:         postService,
		collectionService:   collectionService,
		recentThingsService: recentThingsService,
		thingActivity:       thingActivity,
		fileStore:           repos.Storage,
		tmpFileStore:        repos.TmpStorage,
		cache:               repos.Cache,
		thingService:        thingService,
		repos:               repos,
	}
}
