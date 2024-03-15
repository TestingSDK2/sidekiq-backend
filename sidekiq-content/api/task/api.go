package task

import (
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/api/common"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/board"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/model"

	// "github.com/ProImaging/sidekiq-backend/sidekiq-content/app/notification"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/post"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/profile"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/recent"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/storage"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/task"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/thing"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/thingactivity"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/cache"
)

type api struct {
	config *common.Config
	// clientMgr            *notification.ClientManager
	boardService         board.Service
	taskService          task.Service
	profileService       profile.Service
	recentThingsService  recent.Service
	storageService       storage.Service
	thingService         thing.Service
	thingActivityService thingactivity.Service
	cache                *cache.Cache
	// notificationService  notification.Service
	repos       *model.Repos
	postService post.Service
}

// New creates a new board api
func New(conf *common.Config, boardService board.Service, taskService task.Service,
	profileService profile.Service, recentThingsService recent.Service, storageService storage.Service, thingService thing.Service,
	thingActivityService thingactivity.Service, repos *model.Repos,
	// notificationService notification.Service,
	postService post.Service) *api {
	return &api{
		config: conf,
		// clientMgr:            clientMgr,
		boardService:         boardService,
		taskService:          taskService,
		profileService:       profileService,
		recentThingsService:  recentThingsService,
		storageService:       storageService,
		thingService:         thingService,
		thingActivityService: thingActivityService,
		cache:                repos.Cache,
		// notificationService:  notificationService,
		repos:       repos,
		postService: postService,
	}
}
