package thingactivity

import (
	"encoding/json"
	"net/http"

	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/util"
	"github.com/pkg/errors"
)

func (a *api) ListAllThingActivities(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	if ctx.Profile == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	thingID := ctx.Vars["thingID"]
	if thingID == "" {
		w.WriteHeader(http.StatusBadRequest)
	}

	limit := r.URL.Query().Get("limit")
	page := r.URL.Query().Get("page")

	res, err := a.thingActivityService.ListAllThingActivities(thingID, limit, page)
	if err != nil {
		return errors.Wrap(err, "unable to list thing activities")
	}
	json.NewEncoder(w).Encode(res)

	return nil
}
