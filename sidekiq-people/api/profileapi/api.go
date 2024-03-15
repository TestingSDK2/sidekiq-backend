package profileapi

import (
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-people/api/common"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-people/app"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-people/cache"
	repo "github.com/TestingSDK2/sidekiq-backend/sidekiq-people/model"
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
