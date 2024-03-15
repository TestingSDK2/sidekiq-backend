package post

import (
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/api/common"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/board"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/collection"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/note"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/post"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/profile"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/recent"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/storage"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/task"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/thing"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/thingactivity"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/cache"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/model"
)

type api struct {
	config               *common.Config
	boardService         board.Service
	postService          post.Service
	noteService          note.Service
	taskService          task.Service
	profileService       profile.Service
	recentThingsService  recent.Service
	storageService       storage.Service
	thingService         thing.Service
	thingActivityService thingactivity.Service
	cache                *cache.Cache
	repos                *model.Repos
	collectionService    collection.Service
}

func New(conf *common.Config, boardService board.Service, postService post.Service, noteService note.Service,
	taskService task.Service, profileService profile.Service, recentThingsService recent.Service, storageService storage.Service,
	thingService thing.Service, thingActivityService thingactivity.Service,
	repos *model.Repos, collectionService collection.Service) *api {
	return &api{
		config:               conf,
		boardService:         boardService,
		postService:          postService,
		noteService:          noteService,
		taskService:          taskService,
		profileService:       profileService,
		recentThingsService:  recentThingsService,
		storageService:       storageService,
		thingService:         thingService,
		thingActivityService: thingActivityService,
		cache:                repos.Cache,
		repos:                repos,
		collectionService:    collectionService,
	}
}
