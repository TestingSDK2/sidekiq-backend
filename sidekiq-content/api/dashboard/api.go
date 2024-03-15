package dashboard

import (
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/api/common"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/board"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/profile"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/recent"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/storage"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/cache"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/model"
)

type api struct {
	config *common.Config
	// clientMgr           *notification.ClientManager
	recentThingsService recent.Service
	boardService        board.Service
	profileService      profile.Service
	storageService      storage.Service
	cache               *cache.Cache
}

// New creates a new board api
func New(conf *common.Config, recentThingsService recent.Service,
	boardService board.Service, profileService profile.Service, storageService storage.Service, repos *model.Repos) *api {
	return &api{
		config:              conf,
		recentThingsService: recentThingsService,
		boardService:        boardService,
		profileService:      profileService,
		storageService:      storageService,
		cache:               repos.Cache,
	}
}
