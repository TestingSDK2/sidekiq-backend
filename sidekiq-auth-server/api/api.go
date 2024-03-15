package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-auth-server/api/common"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-auth-server/app"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-auth-server/cache"

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

	r.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"OKK","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
	})

	r.Handle("/auth", a.handler(a.AuthUser, false)).Methods(http.MethodPost)

}
