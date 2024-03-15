package dashboard

import (
	"github.com/ProImaging/sidekiq-backend/sidekiq-search/api/common"

	// "github.com/ProImaging/sidekiq-backend/sidekiq-search/app/notification"
	// "github.com/ProImaging/sidekiq-backend/sidekiq-search/app/profile"
	// "github.com/ProImaging/sidekiq-backend/sidekiq-search/app/recent"
	"github.com/ProImaging/sidekiq-backend/sidekiq-search/app/search"
	"github.com/ProImaging/sidekiq-backend/sidekiq-search/cache"
	repo "github.com/ProImaging/sidekiq-backend/sidekiq-search/model"
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
