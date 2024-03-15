package dashboard

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/util"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-models/model"
	"github.com/pkg/errors"
)

func (a *api) FetchAll(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	res := make(map[string]interface{})
	res["data"] = make(map[string]interface{})
	page := r.URL.Query().Get("page")
	limit := r.URL.Query().Get("limit")

	profileID := ctx.Profile
	if profileID == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	var wg sync.WaitGroup
	wg.Add(1)

	errChan := make(chan error)

	// fetching boards of a profile
	fetchSubBoards := false
	go func(errChan chan<- error) {
		defer util.RecoverGoroutinePanic(nil)
		boards, err := a.boardService.FetchBoards(profileID, fetchSubBoards, page, limit)
		if err != nil {
			errChan <- err
		}
		if boards["data"] != nil {
			res["data"].(map[string]interface{})["boards"] = boards["data"].(map[string]interface{})["info"].([]*model.Board)
		}
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
