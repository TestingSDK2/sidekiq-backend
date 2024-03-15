package api

import (
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-notification/api/common"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-notification/app"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-notification/cache"

	"github.com/gorilla/mux"
)

// API sidekiq api
type API struct {
	App    *app.App
	Config *common.Config
	Cache  *cache.Cache
}

// New creates a new api
func New(a *app.App) (api *API, err error) {
	api = &API{App: a}
	api.Config, err = common.InitConfig()
	if err != nil {
		return nil, err
	}
	return api, nil
}

func (a *API) Init(r *mux.Router) {
}
