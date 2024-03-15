package note

import (
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/api/common"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/board"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/note"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/model"

	// "github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/notification"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/post"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/profile"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/recent"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/storage"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/thing"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/thingactivity"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/cache"
)

type api struct {
	config               *common.Config
	boardService         board.Service
	postService          post.Service
	noteService          note.Service
	profileService       profile.Service
	recentThingsService  recent.Service
	storageService       storage.Service
	thingService         thing.Service
	thingActivityService thingactivity.Service
	repos                *model.Repos
	cache                *cache.Cache
}

// New creates a new file api
func New(conf *common.Config, boardService board.Service,
	postService post.Service, noteService note.Service, profileService profile.Service,
	recentThingsService recent.Service, storageService storage.Service,
	thingService thing.Service, thingActivityService thingactivity.Service,
	repos *model.Repos) *api {
	return &api{
		config:               conf,
		boardService:         boardService,
		postService:          postService,
		noteService:          noteService,
		profileService:       profileService,
		recentThingsService:  recentThingsService,
		storageService:       storageService,
		thingService:         thingService,
		thingActivityService: thingActivityService,
		repos:                repos,
		cache:                repos.Cache,
	}
}
