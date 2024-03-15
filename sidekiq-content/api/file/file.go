package file

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/util"
	model "github.com/TestingSDK2/sidekiq-backend/sidekiq-models"
	searchrpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-search/v1"
	"github.com/pkg/errors"
	// v1 "github.com/sidekiq-search/proto/search"
)

// FetchFilesByBoard - fetches file for the boards if the current user is viewer, member, admin or owner
func (a *api) FetchFilesByBoard(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	boardID := ctx.Vars["boardID"]
	if boardID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	page := r.URL.Query().Get("page")
	limit := r.URL.Query().Get("limit")

	fileRes, err := a.fileService.FetchFilesByBoard(boardID, ctx.Profile, "", "", []string{}, "", 0, page, limit)
	if err == nil {
		json.NewEncoder(w).Encode(fileRes)
		return nil
	}
	return err
}

func (a *api) FetchFileByName(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	var payload map[string]interface{}

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode payload json")
	}
	fileRes, err := a.fileService.FetchFileByName(payload["boardID"].(string), payload["fileName"].(string), ctx.Profile)
	if err == nil {
		json.NewEncoder(w).Encode(fileRes)
		return nil
	}
	return err
}

// AddFile - insert the file for the boards if the current user is viewer, member, admin or owner
func (a *api) AddFile(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	var payload map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode payload json")
	}

	boardID, postID := ctx.Vars["boardID"], ctx.Vars["postID"]
	if boardID == "" || postID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	postdata, err := a.postService.FindPost(boardID, postID)
	if err != nil {
		return errors.Wrap(err, "unable to find post")
	}
	post := postdata["data"].(model.Post)

	fileRes, err := a.fileService.AddFile(boardID, "", payload, ctx.Profile)
	if err == nil {
		json.NewEncoder(w).Encode(fileRes)
		return nil
	}
	// add to search results
	fileRes["data"].(map[string]interface{})["boardID"] = post.BoardID

	// err = a.searchService.UpdateSearchResults(fileRes["data"].(map[string]interface{}), "insert")
	// if err != nil {
	// 	return err
	// }

	reqVal, err := util.MapToMapAny(fileRes["data"].(map[string]interface{}))
	if err != nil {
		return err
	}
	in := &searchrpc.UpdateSearchResultRequest{
		Data:       reqVal,
		UpdateType: "insert",
		Args:       "",
	}
	_, err = a.repos.SearchGrpcServiceClient.UpdateSearchResult(context.TODO(), in)
	if err != nil {
		return err
	}

	// add to activity
	return err
}

// UpdateFile - update the file for the boards if the current user is viewer, member, admin or owner
func (a *api) UpdateFile(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	var payload map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode payload json")
	}

	boardID, postID, fileID := ctx.Vars["boardID"], ctx.Vars["postID"], ctx.Vars["fileID"]
	if boardID == "" || postID == "" || fileID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	postdata, err := a.postService.FindPost(boardID, postID)
	if err != nil {
		return errors.Wrap(err, "unable to find post")
	}
	post := postdata["data"].(model.Post)

	fileRes, err := a.fileService.UpdateFile(payload, boardID, postID, fileID, ctx.Profile)
	if err != nil {
		return errors.Wrap(err, "unable to update file metadata")
	}

	if fileRes["status"].(int) != 0 {
		err := a.profileService.UpdateProfileTagsNew(strconv.Itoa(ctx.Profile))
		if err != nil {
			return errors.Wrap(err, "unable to update Profile Tags")
		}
		// add to search results
		fileRes["data"].(map[string]interface{})["boardID"] = post.BoardID
		// err = a.searchService.UpdateSearchResults(fileRes["data"].(map[string]interface{}), "update")
		// if err != nil {
		// 	return err
		// }

		reqVal, err := util.MapToMapAny(fileRes["data"].(map[string]interface{}))
		if err != nil {
			return err
		}
		in := &searchrpc.UpdateSearchResultRequest{
			Data:       reqVal,
			UpdateType: "update",
			Args:       "",
		}
		_, err = a.repos.SearchGrpcServiceClient.UpdateSearchResult(context.TODO(), in)
		if err != nil {
			return err
		}

		// add to activity
		// cp, err := a.profileService.FetchConciseProfile(ctx.Profile, a.storageService)
		// if err != nil {
		// 	return errors.Wrap(err, "unable to fetch concise Profile")
		// }

		// msg := model.ThingActivity{}
		// msg.Id = primitive.NewObjectID()
		// msg.BoardID = updateFileRes["data"].(model.UploadedFile).BoardID
		// msg.ThingID = updateFileRes["data"].(model.UploadedFile).Id
		// msg.ThingType = "FILE"
		// msg.ProfileID = ctx.Profile
		// msg.Name = fmt.Sprintf("%s %s", cp.FirstName, cp.LastName)
		// msg.Message = " updated the file"
		// msg.DateModified = msg.LastModifiedDate.Format("01-02-2006 15:04:05")
		// err = a.thingActivityService.PushThingActivityToSQS(msg.ToMap())
		// if err != nil {
		// 	return errors.Wrap(err, "unable to push activity to SQS")
		// }
	}
	json.NewEncoder(w).Encode(fileRes)
	return err
}

// DeleteFile - delete the file for the boards if the current user is creator of it
func (a *api) DeleteFile(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	boardID, postID, fileID := ctx.Vars["boardID"], ctx.Vars["postID"], ctx.Vars["fileID"]
	if boardID == "" || postID == "" || fileID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	delFileRes, err := a.fileService.DeleteFile(boardID, postID, fileID, ctx.Profile)

	if err == nil {

		_, err = a.thingService.FlagBookmarkForDelete(ctx.Profile, fileID, time.Now())
		if err != nil {
			fmt.Println(err.Error())
			return errors.Wrap(err, "unable to update flag in bookmark")
		}

		json.NewEncoder(w).Encode(delFileRes)
		return nil
	}

	json.NewEncoder(w).Encode(delFileRes)

	return err
}

func (a *api) GetFileByID(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	fileID, boardID := ctx.Vars["fileID"], ctx.Vars["boardID"]
	if fileID == "" || boardID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	res, err := a.fileService.GetFileByID(fileID, ctx.Profile)
	if err == nil {
		json.NewEncoder(w).Encode(res)
		return nil
	}
	return err
}
