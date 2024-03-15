package profileapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/ProImaging/sidekiq-backend/sidekiq-models/model"
	"github.com/ProImaging/sidekiq-backend/sidekiq-people/app"
	"github.com/ProImaging/sidekiq-backend/sidekiq-people/consts"
	"github.com/ProImaging/sidekiq-backend/sidekiq-people/util"
	contentProtobuf "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-content/v1"
	notiProtobuf "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-notification/v1"
	"github.com/pkg/errors"
)

func (a *api) GetProfiles(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	userIDStr := ctx.Vars["userID"]
	userID, _ := strconv.Atoi(userIDStr)

	ret, err := a.App.ProfileService.GetProfilesByUserID(userID)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(ret)
	return nil
}

func (a *api) GetProfileCount(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	userIDStr := ctx.Vars["userID"]
	userID, _ := strconv.Atoi(userIDStr)

	ret, err := a.App.ProfileService.GetProfileCountByUserID(userID)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(ret)
	return nil
}

func (a *api) SetProfile(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {

	//GRPC METHOD FOR FetchBoards
	//GRPC METHOD FOR GetBoardPermissionByProfile

	// boardsRes, err := a.boardService.FetchBoards(ctx.Profile, true, "", "")
	// boards := boardsRes["data"].([]*model.Board)

	// var boardPermissions *model.BoardPermission
	// if err != nil {
	// 	boardPermissions, _ = a.boardService.GetBoardPermissionByProfile(boards, ctx.Profile)
	// }

	// cache board permissions as per profile
	// cacheKey := fmt.Sprintf("boards:%s", strconv.Itoa(ctx.Profile))
	// a.cache.SetValue(cacheKey, boardPermissions.ToJSON())

	res := util.SetResponse(nil, 1, "All processes complete")
	json.NewEncoder(w).Encode(res)
	return nil
}

func (a *api) AddProfile(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	r.ParseMultipartForm(12 << 20)
	err := r.ParseForm()
	if err != nil {
		return errors.Wrap(err, "error parsing form")
	}
	profileData := r.FormValue("profile")
	var payload model.Profile
	err = json.Unmarshal([]byte(profileData), &payload)
	if err != nil {
		return errors.Wrap(err, "Error Parsing Profile metadata")
	}
	if payload.ConnectCodeExpiration == "" {
		payload.ConnectCodeExpiration = "1w"
	}
	res, err := a.App.ProfileService.AddProfile(payload, ctx.User.ID)
	if err != nil {
		if errors.Is(err, consts.ProfileLimitError) {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(util.SetResponse(nil, 0, consts.ProfileLimitError.Error()))
			return nil
		}
		return err
	}
	profileIDInt := int(res["data"].(map[string]interface{})["id"].(int64))

	// default things board
	request := &contentProtobuf.AddBoardRequest{
		Board: &contentProtobuf.Board{
			Title:          "Default Board",
			Type:           "BOARD",
			Owner:          strconv.Itoa(profileIDInt),
			IsDefaultBoard: true,
			Description:    "This is the default board. This is cannot be deleted",
		},
		ProfileID: int32(profileIDInt),
	}

	response, err := a.App.Repos.ContentServiceClient.AddBoard(context.Background(), request)
	if err != nil {
		return err
	}

	var board model.Board
	err = json.Unmarshal(response.GetData().Value, &board)
	if err != nil {
		return err
	}

	// add default things board ID to profile
	err = a.App.ProfileService.UpdateDefaultThingsBoard(profileIDInt, board.Id.Hex())
	if err != nil {
		return errors.Wrap(err, "unable to update default board ID")
	}

	if err == nil {
		json.NewEncoder(w).Encode(res)
	}

	return err
}

// EditProfile
func (a *api) EditProfile(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	if ctx.Profile == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}
	r.ParseMultipartForm(12 << 20)
	err := r.ParseForm()
	if err != nil {
		return errors.Wrap(err, "error parsing form")
	}
	profileData := r.FormValue("profile")
	var payload model.Profile
	err = json.Unmarshal([]byte(profileData), &payload)
	if err != nil {
		return err
	}
	payload.ID = ctx.Profile
	query := r.URL.Query()
	if query.Get("profileID") != "" {
		payload.ID, _ = strconv.Atoi(query.Get("profileID"))
	}
	_, err = a.App.ProfileService.EditProfile(payload)
	if err != nil {
		return errors.Wrap(err, "unable to update profile in db")
	}

	profileMapInfo, err := a.App.ProfileService.GetProfileInfo(payload.ID)
	if err != nil {
		return errors.Wrap(err, "unable to fetch profile in db")
	}

	profileInfo := profileMapInfo["data"].(model.Profile)
	json.NewEncoder(w).Encode(util.SetResponse(profileInfo, 1, "Profiles updated successfully"))
	return nil
}

func (a *api) GetProfileInfo(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	id := ctx.Profile
	if id == -1 {
		// Profile not authorized to perform this action
		res := util.SetResponse(nil, 0, "Profile not authorized")
		json.NewEncoder(w).Encode(res)
		return nil
	}
	query := r.URL.Query()
	if query.Get("profileID") != "" {
		id, _ = strconv.Atoi(query.Get("profileID"))
	}

	// Update any pending profile tags update
	requestTags := &contentProtobuf.ProfileIDRequest{
		ProfileID: int32(id),
	}

	responseTags, err := a.App.Repos.ContentServiceClient.GetProfileTags(context.TODO(), requestTags)
	if err != nil {
		return err
	}

	err = a.App.ProfileService.UpdateProfileTagsNew(strconv.Itoa(id), responseTags.Tags)
	if err != nil {
		return err
	}

	res, err := a.App.ProfileService.GetProfileInfo(id)
	if err != nil {
		return errors.Wrap(err, "unable to get profile info")
	}

	request := notiProtobuf.GetNotificationDisplayCountRequest{
		ProfileID: fmt.Sprint(ctx.Profile),
	}
	response, err := a.App.Repos.NotificationServiceClient.GetNotificationDisplayCount(context.TODO(), &request)
	if err != nil {
		return errors.Wrap(err, "unable to get profile notification count")
	}

	if profile, ok := res["data"].(model.Profile); ok {
		profile.TotalNotifications = int64(response.Count)
		res["data"] = profile
	}

	json.NewEncoder(w).Encode(res)
	return nil
}

func (a *api) GetProfileTags(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	id := ctx.Profile
	if id == -1 {
		// Profile not authorized to perform this action
		res := util.SetResponse(nil, 0, "Profile not authorized")
		json.NewEncoder(w).Encode(res)
		return nil
	}

	res, err := a.App.ProfileService.FetchTags(id)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(res)
	return nil
}

// DeleteProfile
func (a *api) DeleteProfile(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	id := ctx.Profile
	if id == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}
	callerUserID := ctx.Vars["userID"]

	var payload struct {
		ProfileIDs []string `json:"profileIDs"`
	}
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode request body")
	}
	res, err := a.App.ProfileService.DeleteProfile(callerUserID, payload.ProfileIDs)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(res)
	return nil
}

func (a *api) GenerateCode(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	var err error
	CallerprofileID := ctx.Profile
	if CallerprofileID == -1 {
		res := util.SetResponse(nil, 0, "Profile not authorized")
		json.NewEncoder(w).Encode(res)
		return nil
	}
	var payload model.ConnectionRequest
	err = json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode json body")
	}
	res, err := a.App.ProfileService.GenerateCode(ctx.User.ID, CallerprofileID, payload)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(res)
	return nil
}

func (a *api) DeleteCode(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	var err error
	if ctx.Profile == -1 {
		res := util.SetResponse(nil, 0, "Profile not authorized")
		json.NewEncoder(w).Encode(res)
		return nil
	}

	var payload map[string]interface{}
	err = json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode json")
	}

	res, err := a.App.ProfileService.DeleteCode(ctx.User.ID, payload)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(res)
	return nil
}

// UpdateProfileSettings
func (a *api) UpdateProfileSettings(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	id := ctx.Profile
	if id == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}
	var payload model.UpdateProfileSettings
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode request body")
	}
	res, err := a.App.ProfileService.UpdateProfileSettings(payload, ctx.Profile)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(res)
	return nil
}

func (a *api) UpdateShareableSettings(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	if ctx.Profile == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	var shareableSettings model.ShareableSettings
	err := json.NewDecoder(r.Body).Decode(&shareableSettings)
	if err != nil {
		return errors.Wrap(err, "unable to decode request body")
	}
	response, err := a.App.ProfileService.UpdateShareableSettings(ctx.Profile, shareableSettings)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(response)
	return nil
}

func (a *api) SendCoManagerRequest(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	id := ctx.Profile
	if id == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	var connReq map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&connReq)
	if err != nil {
		return errors.Wrap(err, "unable to decode request body")
	}

	res, err := a.App.ProfileService.SendCoManagerRequest(id, connReq)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(res)
	return nil
}

func (a *api) AcceptCoManagerRequest(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	id := ctx.Profile
	if id == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	var payload map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode request body")
	}

	res, err := a.App.ProfileService.AcceptCoManagerRequest(ctx.User.ID, id, payload["code"].(string))
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(res)
	return nil
}

func (a *api) ListProfilesWithCoManager(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	search := r.URL.Query().Get("search")
	page := r.URL.Query().Get("page")
	limit := r.URL.Query().Get("limit")

	if ctx.Profile == -1 {
		res := util.SetResponse(nil, 0, "Profile not authorized")
		json.NewEncoder(w).Encode(res)
		return nil
	}

	allProfilesRes, err := a.App.ProfileService.GetProfilesWithInfoByUserID(ctx.User.ID)
	if err != nil {
		return err
	}

	// fetch profiles along with their co-managers
	res, err := a.App.ProfileService.FetchProfilesWithCoManager(ctx.Profile, allProfilesRes["data"].([]model.Profile), search, page, limit)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(res)
	return nil
}

func (a *api) ListExternalProfiles(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	id := ctx.Profile
	if id == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}
	search := r.URL.Query().Get("search")
	page := r.URL.Query().Get("page")
	limit := r.URL.Query().Get("limit")

	res, err := a.App.ProfileService.FetchExternalProfiles(ctx.User.ID, search, page, limit)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(res)
	return nil
}

func (a *api) GetPeopleInfo(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	var payload map[string]interface{}
	var res map[string]interface{}
	var err error

	query := r.URL.Query()
	limit := query.Get("limit")
	page := query.Get("page")

	if ctx.Profile == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	err = json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode request body")
	}

	if len(payload) < 1 || len(payload) > 3 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Invalid body found in request"))
		return nil
	}

	if len(payload) > 1 {
		// only one of the paramter exists so setting other to empty
		if len(payload) == 2 {
			for key := range payload {
				if key == "search" {
					payload["boardID"] = ""
					break
				} else if key == "boardID" {
					payload["search"] = ""
					break
				}
			}
		}
	} else {
		// no parameters found in body set default value to empty
		payload["search"] = ""
		payload["boardID"] = ""
	}

	payload["search"] = strings.Replace(payload["search"].(string), "%20", " ", -1)
	res, err = a.App.ProfileService.GetPeopleInfo(ctx.Profile, int(payload["type"].(float64)), limit, page, payload["search"].(string), payload["boardID"].(string))
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(res)
	return nil
}

func (a *api) FetchBoards(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	id := ctx.Profile
	if id == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	res, err := a.App.ProfileService.FetchBoards(id, ctx.Vars["type"])
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(res)
	return nil
}

func (a *api) ListCoManagerCandidates(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	id := ctx.Profile
	if id == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	ret, err := a.App.ProfileService.ListAllOpenProfiles()
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(ret)
	return nil
}

func (a *api) MoveConnection(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	id := ctx.Profile
	if id == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	var payload map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode request body")
	}
	switch reflect.TypeOf(payload).Kind() {
	case reflect.Map:
		res, err := a.App.ProfileService.MoveConnection(payload, id)
		if err != nil {
			return err
		}

		json.NewEncoder(w).Encode(res)
		return nil
	default:
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Invalid request body"))
		return nil
	}
}

func (a *api) DeleteConnection(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	id := ctx.Profile
	if id == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}
	var payload map[string][]string
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode request body")
	}
	if len(payload["connectionProfileIDs"]) == 0 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "no IDs found"))
		return nil
	}

	res, err := a.App.ProfileService.DeleteConnection(payload, id)
	if err != nil {
		return err
	}

	for _, connectionProfileID := range payload["connectionProfileIDs"] {
		receiverID, err := strconv.Atoi(string(connectionProfileID))
		if err != nil {
			return errors.Wrap(err, "unable to parse board.Owner")
		}

		senderId, err := strconv.Atoi(fmt.Sprint(ctx.Profile))
		if err != nil {
			return errors.Wrap(err, "unable to parse profileID")
		}

		request := notiProtobuf.NotificationHandlerRequest{
			ReceiverIDs: []int32{int32(receiverID)},
			SenderID:    int32(senderId),
			ThingType:   consts.Connection,
			ActionType:  consts.DeleteConnection,
			Message:     "",
		}
		_, err = a.App.Repos.NotificationServiceClient.NotificationHandler(context.TODO(), &request)
		if err != nil {
			return errors.Wrap(err, "unable send and create notification")
		}
	}

	json.NewEncoder(w).Encode(res)
	return nil
}

func (a *api) SendConnectionRequest(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	id := ctx.Profile
	if id == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}
	var req map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return errors.Wrap(err, "unable to decode request body")
	}
	res, err := a.App.ProfileService.SendConnectionRequest(id, req)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(res)
	return nil
}

func (a *api) MarkNotificationAsRead(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	id := ctx.Profile
	if id == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	var req map[string]string
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return errors.Wrap(err, "unable to decode request body")
	}

	notificationID, ok := req["id"]
	if !ok {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "id not found in request."))
		return nil
	}

	if notificationID == "" {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "id can not be empty."))
		return nil
	}

	request := notiProtobuf.MarkNotificationAsReadRequest{
		ProfileID:      fmt.Sprint(id),
		NotificationID: notificationID,
	}
	_, err = a.App.Repos.NotificationServiceClient.MarkNotificationAsRead(context.TODO(), &request)
	if err != nil {
		return errors.Wrap(err, "unable to mark notification as read.")
	}

	json.NewEncoder(w).Encode(util.SetResponse(nil, 1, "Notification mark as read"))
	return nil
}

func (a *api) MarkAllNotificationAsRead(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	id := ctx.Profile
	if id == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	request := notiProtobuf.MarkAllNotificationAsReadRequest{
		ProfileID: fmt.Sprint(id),
	}
	_, err := a.App.Repos.NotificationServiceClient.MarkAllNotificationAsRead(context.TODO(), &request)
	if err != nil {
		return errors.Wrap(err, "unable to mark notification as read.")
	}

	json.NewEncoder(w).Encode(util.SetResponse(nil, 1, "All Notification mark as read"))
	return nil
}

func (a *api) NotificationListing(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	if ctx.Profile == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	request := notiProtobuf.GetNotificationListRequest{
		ProfileID: fmt.Sprint(ctx.Profile),
	}
	response, err := a.App.Repos.NotificationServiceClient.GetNotificationList(context.TODO(), &request)
	if err != nil {
		return errors.Wrap(err, "error while getting notifications")
	}

	var notifications []model.Notification
	err = json.Unmarshal(response.GetData().Value, &notifications)
	if err != nil {
		return errors.Wrap(err, "error while mapping notifications")
	}

	var wg sync.WaitGroup
	cps := make(map[string]*model.ConciseProfile)
	for index, notification := range notifications {
		wg.Add(1)
		go func(index int, notification model.Notification) {
			defer wg.Done()
			if val, ok := cps[notification.SenderProfileID]; !ok {
				SenderProfileID, err := strconv.Atoi(notification.SenderProfileID)
				if err == nil {
					profileInfo, err := a.App.ProfileService.GetConciseProfile(SenderProfileID)
					if err == nil {
						notifications[index].SenderDetails = profileInfo
						cps[notification.SenderProfileID] = profileInfo
					}
				}
			} else {
				notifications[index].SenderDetails = val
			}
		}(index, notification)
	}

	wg.Wait()

	// get the last inviter's photo, fn, ln, invitation count
	reqProfile := contentProtobuf.ProfileIDRequest{
		ProfileID: int32(ctx.Profile),
	}

	resInvites, err := a.App.Repos.ContentServiceClient.ListBoardInvites(context.TODO(), &reqProfile)
	if err != nil {
		return err
	}

	var invites []model.ListInvitations
	err = json.Unmarshal(resInvites.GetData().Value, &invites)
	if err != nil {
		return err
	}

	res := make(map[string]interface{})
	res["invitation"] = map[string]interface{}{}
	res["notifications"] = []model.Notification{}

	if len(notifications) > 0 {
		res["notifications"] = notifications
	}

	if len(invites) > 0 {
		invite := invites[0]
		res["invitation"] = map[string]interface{}{
			"total": len(invites),
			"profile": map[string]interface{}{
				"firstName": invite.FirstName,
				"lastName":  invite.LastName,
				"photo":     invite.Photo,
			},
		}
	}

	return json.NewEncoder(w).Encode(util.SetResponse(res, 1, "Notification list"))
}

func (a *api) AcceptConnectionRequest(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	id := ctx.Profile
	if id == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}
	var payload map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode request body")
	}

	res, err := a.App.ProfileService.AcceptConnectionRequest(id, payload["code"].(string))
	if err == nil {
		if res["data"] != nil && res["status"] == 1 {
			receiverID, err := strconv.Atoi(string(res["data"].(model.Connection).ConnectionProfileID))
			if err != nil {
				return errors.Wrap(err, "unable to parse ConnectionProfileID")
			}

			request := notiProtobuf.NotificationHandlerRequest{
				ReceiverIDs: []int32{int32(receiverID)},
				SenderID:    int32(ctx.Profile),
				ThingType:   consts.Connection,
				ThingID:     res["data"].(model.Connection).ID.Hex(),
				ActionType:  consts.AcceptConnectionRequest,
				Message:     "",
			}
			_, err = a.App.Repos.NotificationServiceClient.NotificationHandler(context.TODO(), &request)
			if err != nil {
				return errors.Wrap(err, "unable send and create notification")
			}
		}

		json.NewEncoder(w).Encode(res)
		return nil
	}

	return err
}

// AddConnectionDetails
func (a *api) AddConnectionDetails(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	var err error
	if ctx.Profile == -1 {
		res := util.SetResponse(nil, 0, "Profile not authorized")
		json.NewEncoder(w).Encode(res)
		return nil
	}
	var payload model.Connection
	err = json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode json body")
	}

	payload.ProfileID = strconv.Itoa(ctx.Profile)
	res, err := a.App.ProfileService.AddConnectionDetails(payload)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(res)
	return nil
}

// GETConnectionDetails
func (a *api) GetConnectionDetails(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	var err error
	if ctx.Profile == -1 {
		res := util.SetResponse(nil, 0, "Profile not authorized")
		json.NewEncoder(w).Encode(res)
		return nil
	}
	// connection's profile id
	query := r.URL.Query()
	connectionProfileID := query.Get("profileID")

	// caller's profile ID
	callerProfileID := strconv.Itoa(ctx.Profile)

	res, err := a.App.ProfileService.GetConnectionDetails(callerProfileID, connectionProfileID)
	if err != nil {
		return err
	}
	json.NewEncoder(w).Encode(res)
	return nil
}

func (a *api) ProfileView(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	resp := map[string]interface{}{
		"boards":  map[string]interface{}{},
		"profile": map[string]interface{}{},
		// "isPrivate": false,
	}
	callerID := strconv.Itoa(ctx.Profile)
	profileID := ctx.Vars["profileID"]

	requestTags := &contentProtobuf.ProfileIDRequest{
		ProfileID: int32(ctx.Profile),
	}

	responseTags, err := a.App.Repos.ContentServiceClient.GetProfileTags(context.TODO(), requestTags)
	if err != nil {
		return err
	}

	err = a.App.ProfileService.UpdateProfileTagsNew(strconv.Itoa(ctx.Profile), responseTags.Tags)
	if err != nil {
		return errors.Wrap(err, "unable to update tags")
	}

	profile, err := a.App.ProfileService.FetchProfileView(profileID, callerID)
	if err != nil {
		return errors.Wrap(err, "unable to fetch profile-about")
	}
	profileData, ok := profile["data"].(model.ProfileView)
	if !ok {
		return errors.Wrap(err, "error in type asserting profile data")
	}

	resp["profile"] = profile["data"]
	if profileID != callerID && profileData.ShowBoards == 0 {
		resp["boards"] = nil
		json.NewEncoder(w).Encode(util.SetResponse(resp, 1, "Profile view fetched successfully"))
		return nil
	}
	boards, err := a.App.ProfileService.FetchProfileBoardsView(profileID, callerID)
	if err != nil {
		return errors.Wrap(err, "unable to fetch profile-boards")
	}
	resp["boards"] = boards["data"]
	if err == nil {
		json.NewEncoder(w).Encode(util.SetResponse(resp, 1, "Profile view fetched successfully"))
	}
	return err
}

func (a *api) GetOrgStaff(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	id := ctx.Profile
	if id == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	query := r.URL.Query()
	limit := query.Get("limit")
	page := query.Get("page")
	searchParameter := query.Get("search")
	ret, err := a.App.ProfileService.GetOrgStaff(id, ctx.Vars["userID"], limit, page, searchParameter)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(ret)
	return nil
}

func (a *api) GetStaffProfile(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	comanagerID, err := strconv.Atoi(ctx.Vars["comanagerID"])
	if err != nil {
		return err
	}
	res, err := a.App.ProfileService.FetchStaffProfile(ctx.User.ID, comanagerID)
	if err != nil {
		json.NewEncoder(w).Encode(res)
		return nil
	}

	json.NewEncoder(w).Encode(res)
	return nil
}

func (a *api) UpdateStaffProfile(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	comanagerID, err := strconv.Atoi(ctx.Vars["comanagerID"])
	if err != nil {
		return errors.Wrap(err, "unable to convert to int")
	}
	// profileData := r.FormValue("profile")

	var payload model.OrgStaff
	err = json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode request body")
	}

	res, err := a.App.ProfileService.UpdateStaffProfile(ctx.User.ID, comanagerID, payload)
	if err != nil {
		return errors.Wrap(err, "unable to update staff profile")
	}

	json.NewEncoder(w).Encode(res)
	return nil
}

// SetOrganizationInfo - Set organization information in DB
func (a *api) SetOrganizationInfo(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	r.ParseMultipartForm(12 << 20)
	err := r.ParseForm()
	if err != nil {
		return errors.Wrap(err, "error parsing form")
	}
	var payload *model.Organization
	err = json.Unmarshal([]byte(r.FormValue("orgInfo")), &payload)
	if err != nil {
		return errors.Wrap(err, "error Parsing Profile metadata")
	}
	payload.AccountID = ctx.User.ID
	response, err := a.App.ProfileService.SetOrganizationInfo(payload)
	if err != nil {
		return errors.Wrap(err, "unable to insert organization info")
	}
	if response["status"] == 0 {
		json.NewEncoder(w).Encode(response)
		return nil
	}

	json.NewEncoder(w).Encode(response)
	return nil
}

func (a *api) GetOrgInfo(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	res, err := a.App.ProfileService.GetOrgInfo(ctx.User.ID)
	if err == nil {
		json.NewEncoder(w).Encode(res)
		return nil
	}
	return err
}

func (a *api) UpdateOrgInfo(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	r.ParseMultipartForm(12 << 20)
	err := r.ParseForm()
	if err != nil {
		return errors.Wrap(err, "error parsing form")
	}
	var payload model.Organization
	err = json.Unmarshal([]byte(r.FormValue("orgInfo")), &payload)
	if err != nil {
		return errors.Wrap(err, "error Parsing Profile metadata")
	}
	payload.AccountID = ctx.User.ID

	response, err := a.App.ProfileService.UpdateOrgInfo(payload)
	if err != nil {
		return errors.Wrap(err, "unable to update organization info")
	}
	if response["status"] == 0 {
		json.NewEncoder(w).Encode(response)
		return nil
	}
	res, err := a.App.ProfileService.GetOrgInfo(payload.AccountID)
	if err != nil {
		return errors.Wrap(err, "unable to fetch org info")
	}

	json.NewEncoder(w).Encode(res)
	return nil
}
