package accountapi

import (
	"github.com/ProImaging/sidekiq-backend/sidekiq-people/api/common"
	"github.com/ProImaging/sidekiq-backend/sidekiq-people/app"
	"github.com/ProImaging/sidekiq-backend/sidekiq-people/cache"
	repo "github.com/ProImaging/sidekiq-backend/sidekiq-people/model"
)

// API sidekiq api
type api struct {
	config *common.Config
	cache  *cache.Cache
	App    *app.App
}

// New creates a new api
func New(conf *common.Config, repos *repo.Repos, app *app.App) *api {
	return &api{
		config: conf,
		cache:  repos.Cache,
		App:    app,
	}
}
