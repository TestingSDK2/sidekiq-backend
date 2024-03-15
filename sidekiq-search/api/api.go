package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-search/api/common"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-search/api/dashboard"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-search/app"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-search/cache"
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

// Init initializes the api
func (a *API) Init(r *mux.Router) {

	// SERVER-STATUS
	r.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"OKK","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
	})

	/* ****************** DASHBOARD ****************** */
	dashBoardAPI := dashboard.New(a.Config,
		a.App.SearchService, a.App.Repos)
	r.Handle("/dashboard", a.handler(dashBoardAPI.FetchAll, true)).Methods(http.MethodGet)
	r.Handle("/dashboard/search", a.handler(dashBoardAPI.FullTextSearch, true)).Methods(http.MethodPost)
	r.Handle("/dashboard/search/ac", a.handler(dashBoardAPI.Autocomplete, true)).Methods(http.MethodGet)
	r.Handle("/dashboard/search/history", a.handler(dashBoardAPI.GetDashBoardSearchHistory, true)).Methods(http.MethodGet)
	r.Handle("/dashboard/search/history", a.handler(dashBoardAPI.UpdateDashBoardSearchHistory, true)).Methods(http.MethodPut)
}
