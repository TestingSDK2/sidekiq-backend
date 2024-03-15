package dashboard

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-models/model"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-search/app"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-search/util"
	"github.com/pkg/errors"
)

func (a *api) FetchAll(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	res := make(map[string]interface{})
	res["data"] = make(map[string]interface{})
	// page := r.URL.Query().Get("page")
	// limit := r.URL.Query().Get("limit")

	profileID := ctx.Profile
	if profileID == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	var wg sync.WaitGroup
	wg.Add(1)

	errChan := make(chan error)

	// fetching boards of a profile
	// fetchSubBoards := false
	go func(errChan chan<- error) {
		defer util.RecoverGoroutinePanic(nil)
		// GRPC for FetchBoards
		// boards, err := a.boardService.FetchBoards(profileID, fetchSubBoards, page, limit)
		// if err != nil {
		// 	errChan <- err
		// }
		// if boards["data"] != nil {
		// 	res["data"].(map[string]interface{})["boards"] = boards["data"].(map[string]interface{})["info"].([]*model.Board)
		// }
		wg.Done()
		errChan <- nil
	}(errChan)

	for i := 0; i < 1; i++ {
		if err := <-errChan; err != nil {
			return errors.Wrap(err, "error from go routine")
		}
	}
	res["status"] = 1
	res["message"] = "Recent things & boards fetched successfully."
	wg.Wait()
	json.NewEncoder(w).Encode(res)
	return nil
}

func (a *api) Autocomplete(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	var err error
	query := r.URL.Query().Get("search")

	if ctx.Profile == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	res, err := a.searchService.AutoComplete(ctx.Profile, query)
	if err == nil {
		json.NewEncoder(w).Encode(res)
	}
	return err
}

func (a *api) FullTextSearch(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	if ctx.Profile == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	var err error
	var res map[string]interface{}
	page := r.URL.Query().Get("page")
	limit := r.URL.Query().Get("limit")
	query := r.URL.Query().Get("search")
	sortBy := r.URL.Query().Get("sortBy")
	orderBy := r.URL.Query().Get("orderBy")

	// get filter params
	// fileType := r.URL.Query().Get("fileType")
	// people := r.URL.Query().Get("people")
	// location := r.URL.Query().Get("location")
	// uploadDate := r.URL.Query().Get("uploadDate")

	// filterMap := map[string]interface{}{
	// 	"fileType":   fileType,
	// 	"people":     people,
	// 	"location":   location,
	// 	"uploadDate": uploadDate,
	// }
	// searchFilter := util.ParseSearchFilter(filterMap)

	// check if filter is given or not
	var filter *model.GlobalSearchFilter
	err = json.NewDecoder(r.Body).Decode(&filter)
	if err != nil {
		if err.Error() == "EOF" { // no body
			filter = nil
		} else {
			return err
		}
	}

	res, err = a.searchService.FTSOnDashboard(filter, ctx.Profile, query, page, limit, sortBy, orderBy, false)
	if err == nil {
		json.NewEncoder(w).Encode(res)
	}

	return err
}

func (a *api) GetDashBoardSearchHistory(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	if ctx.Profile == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	res, err := a.searchService.FetchSearchHistory(ctx.Profile)
	if err == nil {
		json.NewEncoder(w).Encode(res)
	}

	return err
}

func (a *api) UpdateDashBoardSearchHistory(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	if ctx.Profile == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	var payload map[string]interface{}
	var err error
	err = json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode json")
	}

	res, err := a.searchService.AddToSearchHistory(ctx.Profile, payload["search"].(string))
	if err == nil {
		json.NewEncoder(w).Encode(res)
	}

	return err
}
