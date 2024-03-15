package file

import (
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/api/common"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/board"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/file"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/model"

	// "github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/notification"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/post"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/profile"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/storage"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/thing"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/thingactivity"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/cache"
)

type api struct {
	config *common.Config
	// clientMgr            *notification.ClientManager
	fileService          file.Service
	boardService         board.Service
	profileService       profile.Service
	storageService       storage.Service
	thingActivityService thingactivity.Service
	postService          post.Service
	thingService         thing.Service
	repos                *model.Repos
	cache                *cache.Cache
}

// New creates a new file api
func New(conf *common.Config, fileService file.Service,
	boardService board.Service, profileService profile.Service, storageService storage.Service,
	thingActivityService thingactivity.Service, postService post.Service, repos *model.Repos, thingService thing.Service) *api {
	return &api{
		config: conf,
		// clientMgr:            clientMgr,
		fileService:          fileService,
		boardService:         boardService,
		profileService:       profileService,
		storageService:       storageService,
		thingActivityService: thingActivityService,
		thingService:         thingService,
		postService:          postService,
		repos:                repos,
		cache:                repos.Cache,
	}
}
