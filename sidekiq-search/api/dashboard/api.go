package dashboard

import (
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-search/api/common"

	// "github.com/TestingSDK2/sidekiq-backend/sidekiq-search/app/notification"
	// "github.com/TestingSDK2/sidekiq-backend/sidekiq-search/app/profile"
	// "github.com/TestingSDK2/sidekiq-backend/sidekiq-search/app/recent"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-search/app/search"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-search/cache"
	repo "github.com/TestingSDK2/sidekiq-backend/sidekiq-search/model"
)

type api struct {
	config *common.Config
	// boardService  board.Service
	searchService search.Service
	cache         *cache.Cache
}

// New creates a new board api
func New(conf *common.Config,
	searchService search.Service, repos *repo.Repos) *api {
	return &api{
		config: conf,
		// boardService:  boardService,
		searchService: searchService,
		cache:         repos.Cache,
	}
}
