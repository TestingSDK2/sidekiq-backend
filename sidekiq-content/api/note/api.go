package note

import (
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/api/common"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/board"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/note"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/model"

	// "github.com/ProImaging/sidekiq-backend/sidekiq-content/app/notification"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/post"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/profile"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/recent"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/storage"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/thing"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/thingactivity"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/cache"
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
