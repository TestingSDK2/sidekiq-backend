package collection

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/consts"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/util"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-models/model"

	// peoplerpc "github.com/sidekiq-people/proto/people"
	peoplerpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AddCollection - creates a new collection in the board
func (a *api) AddCollection(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	if ctx.Profile == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	boardID := ctx.Vars["boardID"]
	postID := ctx.Vars["postID"]

	if boardID == "" || postID == "" {
		w.WriteHeader(http.StatusBadRequest)
	}

	var err error
	_, err = a.boardService.FetchBoardInfo(boardID)
	if err != nil {
		return errors.Wrap(err, "unable to fetch board info")
	}

	_, err = a.postService.FindPost(boardID, postID)
	if err != nil {
		return errors.Wrap(err, "unable to fetch post info")
	}

	var payload model.Collection

	profileID := ctx.Profile
	err = json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode payload json for add collection")
	}

	payload.FileProcStatus = consts.FileCOMPLETE

	res, err := a.collectionService.AddCollection(payload, profileID, boardID, postID)
	if err != nil {
		return errors.Wrap(err, "error adding new collection")
	}
	err = a.profileService.UpdateProfileTagsNew(strconv.Itoa(ctx.Profile))
	if err != nil {
		return errors.Wrap(err, "unable to update tags")
	}
	json.NewEncoder(w).Encode(res)
	return nil
}

// GetCollection - creates a new collection in the board
func (a *api) GetCollection(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	if ctx.Profile == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}
	var err error
	profileID := ctx.Profile
	// boardID := r.URL.Query().Get("boardID")
	boardID := ctx.Vars["boardID"]
	postID := ctx.Vars["postID"]
	limit := r.URL.Query().Get("limit")
	page := r.URL.Query().Get("page")

	if boardID == "" || postID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	_, err = a.boardService.FetchBoardInfo(boardID)
	if err != nil {
		return errors.Wrap(err, "unable to fetch board info")
	}

	_, err = a.postService.FindPost(boardID, postID)
	if err != nil {
		return errors.Wrap(err, "unable to fetch post info")
	}

	res, err := a.collectionService.GetCollection(boardID, postID, profileID, "", []string{}, "", 0, page, limit)
	if err != nil {
		return errors.Wrap(err, "error adding new collection")
	}
	json.NewEncoder(w).Encode(res)
	return nil
}

// UpdateCollection - updates collection in the board
func (a *api) UpdateCollection(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	id := ctx.Profile
	if id == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}
	var err error
	boardID := ctx.Vars["boardID"]
	postID := ctx.Vars["postID"]
	collectionID := ctx.Vars["collectionID"]

	if boardID == "" || postID == "" || collectionID == "" {
		w.WriteHeader(http.StatusBadRequest)
	}

	_, err = a.boardService.FetchBoardInfo(boardID)
	if err != nil {
		return errors.Wrap(err, "unable to fetch board info")
	}

	_, err = a.postService.FindPost(boardID, postID)
	if err != nil {
		return errors.Wrap(err, "unable to fetch post info")
	}

	var payload model.UpdateCollection
	err = json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode payload json for add collection")
	}

	res, err := a.collectionService.UpdateCollection(payload, boardID, postID, collectionID, ctx.Profile)
	if err != nil {
		return errors.Wrap(err, "unable to update collection in db")
	}
	if res["status"] == 0 {
		json.NewEncoder(w).Encode(res)
		return nil
	}
	err = a.profileService.UpdateProfileTagsNew(strconv.Itoa(ctx.Profile))
	if err != nil {
		return errors.Wrap(err, "unable to update tags")
	}
	json.NewEncoder(w).Encode(res)
	return err
}

// func (a *api) SearchCollectionThings(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
// 	if ctx.Profile == -1 {
// 		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
// 		return nil
// 	}
// 	var err error
// 	profileID := ctx.Profile
// 	thingName := r.URL.Query().Get("thing")
// 	boardID := ctx.Vars["boardID"]
// 	limit := r.URL.Query().Get("limit")
// 	page := r.URL.Query().Get("page")
// 	res, err := a.collectionService.SearchCollectionThings(a.cache, a.profileService, a.storageService, boardID, thingName, profileID, 0, page, limit)
// 	if err != nil {
// 		return errors.Wrap(err, "error adding new collection")
// 	}
// 	json.NewEncoder(w).Encode(res)
// 	return nil
// }

// GetCollection - creates a new collection in the board
func (a *api) GetCollectionByID(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	if ctx.Profile == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	var err error

	profileID := ctx.Profile
	boardID := ctx.Vars["boardID"]
	postID := ctx.Vars["postID"]
	collectionID := ctx.Vars["collectionID"]

	if boardID == "" || collectionID == "" || postID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	_, err = a.boardService.FetchBoardInfo(boardID)
	if err != nil {
		return errors.Wrap(err, "unable to fetch board info")
	}

	_, err = a.postService.FindPost(boardID, postID)
	if err != nil {
		return errors.Wrap(err, "unable to fetch post info")
	}

	res, err := a.collectionService.GetCollectionByID(boardID, postID, collectionID, profileID)
	if err != nil {
		return errors.Wrap(err, "error getting collection by id")
	}
	json.NewEncoder(w).Encode(res)
	return nil
}

func (a *api) UpdateCollectionStatus(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	if ctx.Profile == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	var err error

	profileID := ctx.Profile
	boardID := ctx.Vars["boardID"]
	postID := ctx.Vars["postID"]
	collectionID := ctx.Vars["collectionID"]

	if boardID == "" || collectionID == "" || postID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}
	_, err = a.boardService.FetchBoardInfo(boardID)
	if err != nil {
		return errors.Wrap(err, "unable to fetch board info")
	}

	_, err = a.postService.FindPost(boardID, postID)
	if err != nil {
		return errors.Wrap(err, "unable to fetch post info")
	}

	type requestData struct {
		Status string `json:"status"`
	}

	var req requestData
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return errors.Wrap(err, "unable to decode payload json for add collection")
	}

	if req.Status == "" {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "status value not be empty"))
		return nil
	}

	if strings.ToUpper(req.Status) != consts.FileCOMPLETE && strings.ToUpper(req.Status) != consts.FilePROCESSING {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "status value can be COMPLETE or PROCESSING"))
		return nil
	}

	req.Status = strings.ToUpper(req.Status)

	res, err := a.collectionService.UpdateCollecitonStatusByID(boardID, postID, collectionID, req.Status, profileID)
	if err != nil {
		return errors.Wrap(err, "error updating status for collection by id")
	}

	json.NewEncoder(w).Encode(res)
	return nil
}

func (a *api) GetMediaUploadStatusCollection(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	boardID := ctx.Vars["boardID"]
	postID := ctx.Vars["postID"]
	collectionID := ctx.Vars["collectionID"]

	if boardID == "" || postID == "" || collectionID == "" {
		w.WriteHeader(http.StatusBadRequest)
	}

	boardInfo, err := a.boardService.FetchBoardInfo(boardID)
	if err != nil {
		return errors.Wrap(err, "unable to fetch board info")
	}
	_, err = a.postService.FindPost(boardID, postID)
	if err != nil {
		return errors.Wrap(err, "unable to fetch post info")
	}

	boardOwnerInt, err := strconv.Atoi(boardInfo["owner"].(string))
	if err != nil {
		return err
	}

	// ownerInfo, err := a.profileService.FetchConciseProfile(boardOwnerInt)
	// if err != nil {
	// 	return err
	// }

	cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(boardOwnerInt)}
	ownerInfo, err := a.repos.PeopleGrpcServiceClient.GetConciseProfile(context.TODO(), cpreq)
	if err != nil {
		return err
	}

	fmt.Println("FileUploadStatus for board: ", boardID)
	key := util.GetKeyForPostCollectionMedia(int(ownerInfo.AccountID), int(ownerInfo.Id), boardID, postID, collectionID, "")
	parts, err := a.storageService.GetFileParts(ctx.Profile, key, ctx.Vars["id"])
	if err != nil {
		return err
	}
	fmt.Println(len(parts))
	json.NewEncoder(w).Encode(parts)
	return nil
}

// FetchFilesByCollection - fetches file for the collection if the current user is viewer, member, admin or owner
func (a *api) FetchFilesByCollection(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	boardID := ctx.Vars["boardID"]
	postID := ctx.Vars["postID"]
	collectionID := ctx.Vars["collectionID"]

	if boardID == "" || postID == "" || collectionID == "" {
		w.WriteHeader(http.StatusBadRequest)
	}

	boardInfo, err := a.boardService.FetchBoardInfo(boardID)
	if err != nil {
		return errors.Wrap(err, "unable to fetch board info")
	}

	_, err = a.postService.FindPost(boardID, postID)
	if err != nil {
		return errors.Wrap(err, "unable to fetch post info")
	}

	fileName := r.URL.Query().Get("filename")
	page := r.URL.Query().Get("page")
	limit := r.URL.Query().Get("limit")

	boardOwnerInt, err := strconv.Atoi(boardInfo["owner"].(string))
	if err != nil {
		return errors.Wrap(err, "unable to convert string to int")
	}

	// ownerInfo, err := a.profileService.FetchConciseProfile(boardOwnerInt)
	// if err != nil {
	// 	return err
	// }

	cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(boardOwnerInt)}
	ownerInfo, err := a.repos.PeopleGrpcServiceClient.GetConciseProfile(context.TODO(), cpreq)
	if err != nil {
		return err
	}

	collectionFileRes, err := a.collectionService.FetchFilesByCollection(boardID, postID, collectionID, fileName, ctx.Profile, 0, page, limit, ownerInfo)
	if err == nil {
		json.NewEncoder(w).Encode(collectionFileRes)
		return nil
	}
	return err
}

// DeleteCollectionMedia - Delete files in collection
func (a *api) DeleteCollectionMedia(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	boardID := ctx.Vars["boardID"]
	// thingType := ctx.Vars["thingType"]
	thingID := ctx.Vars["fileID"]
	thingType := "file"
	postID := ctx.Vars["postID"]
	collectionID := ctx.Vars["collectionID"]

	if thingID == "" || thingType == "" || boardID == "" || postID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	_, err := a.boardService.FetchBoardInfo(boardID)
	if err != nil {
		return errors.Wrap(err, "unable to fetch board info")
	}

	_, err = a.postService.FindPost(boardID, postID)
	if err != nil {
		return errors.Wrap(err, "unable to fetch post info")
	}

	thingTypes := []string{"note", "task", "file"}
	if !strings.Contains(strings.Join(thingTypes, ","), strings.ToLower(thingType)) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return nil
	}

	// Update collection mongo
	res, err := a.collectionService.DeleteCollectionMedia(collectionID, thingID)
	if err != nil {
		return errors.Wrap(err, "unable to delete from mongo collection")
	}

	// Delete actual thing
	if res["status"].(int) != 0 {
		var delRes map[string]interface{}
		switch thingType {
		case "note":
			note, err := a.noteService.DeleteNote(boardID, thingID, ctx.Profile)
			if err == nil {
				// delete from recentlyAdded, if present
				// _, err = a.recentThingsService.DeleteFromBoardThingsRecent(ctx.Profile, thingID, boardID)

				// add to activity
				// cp, _ := a.profileService.FetchConciseProfile(ctx.Profile)

				cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(ctx.Profile)}
				cp, err := a.repos.PeopleGrpcServiceClient.GetConciseProfile(context.TODO(), cpreq)
				if err != nil {
					return err
				}

				noteObjID, err := primitive.ObjectIDFromHex(thingID)
				if err != nil {
					return errors.Wrap(err, "unable to convert string to objectID")
				}
				boardObjID, err := primitive.ObjectIDFromHex(boardID)
				if err != nil {
					return errors.Wrap(err, "unable to convert string to objectID")
				}
				postObjID, err := primitive.ObjectIDFromHex(postID)
				if err != nil {
					return errors.Wrap(err, "unable to convert string to objectID")
				}

				_, err = a.thingService.FlagBookmarkForDelete(ctx.Profile, thingID, time.Now())
				if err != nil {
					fmt.Println(err.Error())
					return errors.Wrap(err, "unable to update flag in bookmark")
				}

				msg := model.ThingActivity{}
				msg.Id = primitive.NewObjectID()
				msg.BoardID = boardObjID
				msg.PostID = postObjID
				msg.ThingID = noteObjID
				msg.ThingType = "NOTE"
				msg.ProfileID = ctx.Profile
				// msg.Name = fmt.Sprintf("%s %s", cp.FirstName, cp.LastName)
				msg.Message = fmt.Sprintf("%s %s deleted a Note", cp.FirstName, cp.LastName)
				msg.LastModifiedDate = time.Now()
				a.thingActivityService.PushThingActivityToSQS(msg.ToMap())
			} else {
				return errors.Wrap(err, "unable to delete note")
			}
			json.NewEncoder(w).Encode(note)
		case "task":
			taskRes, err := a.taskService.DeleteTask(boardID, thingID, ctx.Profile)
			if err != nil {
				return errors.Wrap(err, "unable to delete Task")
			} else {
				fmt.Println("task deleted successfully")
			}

			if taskRes["status"].(int) != 0 {
				// delete from recentlyAdded, if present
				// _, err = a.recentThingsService.DeleteFromBoardThingsRecent(ctx.Profile, thingID, boardID)

				// add to activity
				// cp, _ := a.profileService.FetchConciseProfile(ctx.Profile)
				cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(ctx.Profile)}
				cp, err := a.repos.PeopleGrpcServiceClient.GetConciseProfile(context.TODO(), cpreq)
				if err != nil {
					return err
				}

				taskObjID, err := primitive.ObjectIDFromHex(thingID)
				if err != nil {
					return errors.Wrap(err, "unable to convert string to objectID")
				}
				boardObjID, err := primitive.ObjectIDFromHex(boardID)
				if err != nil {
					return errors.Wrap(err, "unable to convert string to objectID")
				}
				postObjID, err := primitive.ObjectIDFromHex(postID)
				if err != nil {
					return errors.Wrap(err, "unable to convert string to objectID")
				}

				_, err = a.thingService.FlagBookmarkForDelete(ctx.Profile, thingID, time.Now())
				if err != nil {
					fmt.Println(err.Error())
					return errors.Wrap(err, "unable to update flag in bookmark")
				}

				msg := model.ThingActivity{}
				msg.Id = primitive.NewObjectID()
				msg.BoardID = boardObjID
				msg.PostID = postObjID
				msg.ThingID = taskObjID
				msg.ThingType = "TASK"
				msg.ProfileID = ctx.Profile
				// msg.Name = fmt.Sprintf("%s %s", cp.FirstName, cp.LastName)
				msg.Message = fmt.Sprintf("%s %s deleted a Task", cp.FirstName, cp.LastName)
				msg.LastModifiedDate = time.Now()
				a.thingActivityService.PushThingActivityToSQS(msg.ToMap())
			}
		case "file":
			delRes, err = a.fileService.DeleteFile(boardID, postID, thingID, ctx.Profile)
			if err != nil {
				return errors.Wrap(err, "unable to delete file from mongo")
			} else {
				fmt.Println("file deleted successfully")

				_, err = a.thingService.FlagBookmarkForDelete(ctx.Profile, thingID, time.Now())
				if err != nil {
					fmt.Println(err.Error())
					return errors.Wrap(err, "unable to update flag in bookmark")
				}
			}

		}
		if delRes["status"].(int) == 1 {
			json.NewEncoder(w).Encode(util.SetResponse(nil, 1, "Collection media deleted successfully"))
		} else {
			json.NewEncoder(w).Encode(delRes)
		}
		return nil
	}
	json.NewEncoder(w).Encode(res)
	return err
}

// EditCollectionMedia - Edit images in collection
func (a *api) EditCollectionMedia(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	boardID := ctx.Vars["boardID"]
	// thingType := ctx.Vars["thingType"]
	thingID := ctx.Vars["fileID"]
	thingType := "file"
	postID := ctx.Vars["postID"]
	// collectionID := ctx.Vars["collectionID"]

	if thingID == "" || postID == "" || thingType == "" || boardID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	_, err := a.boardService.FetchBoardInfo(boardID)
	if err != nil {
		return errors.Wrap(err, "unable to fetch board info")
	}

	_, err = a.postService.FindPost(boardID, postID)
	if err != nil {
		return errors.Wrap(err, "unable to fetch post info")
	}

	thingTypes := []string{"note", "task", "file"}
	if !strings.Contains(strings.Join(thingTypes, ","), strings.ToLower(thingType)) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return nil
	}

	var payload model.UpdateCollection
	err = json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode user json")
	}

	// Update collection mongo
	res, err := a.collectionService.EditCollectionMedia(payload, strconv.Itoa(ctx.Profile), thingID, thingType, boardID, postID)
	if err != nil {
		return errors.Wrap(err, "unable to edit media")
	}
	err = a.profileService.UpdateProfileTagsNew(strconv.Itoa(ctx.Profile))
	if err != nil {
		return errors.Wrap(err, "unable to update tags")
	}
	// Delete actual thing
	json.NewEncoder(w).Encode(res)
	return nil
}

// DeleteCollection - Delete
func (a *api) DeleteCollection(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	id := ctx.Profile
	if id == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}
	var err error
	boardID := ctx.Vars["boardID"]
	postID := ctx.Vars["postID"]
	collectionID := ctx.Vars["collectionID"]
	if boardID == "" {
		w.WriteHeader(http.StatusBadRequest)
	}

	if boardID == "" || postID == "" || collectionID == "" {
		w.WriteHeader(http.StatusBadRequest)
	}

	_, err = a.boardService.FetchBoardInfo(boardID)
	if err != nil {
		return errors.Wrap(err, "unable to fetch board info")
	}

	_, err = a.postService.FindPost(boardID, postID)
	if err != nil {
		return errors.Wrap(err, "unable to fetch post info")
	}

	// fetch collection things from mongo
	res, err := a.collectionService.GetCollectionByID(boardID, postID, collectionID, ctx.Profile)
	if err != nil {
		return errors.Wrap(err, "error getting collection by id")
	}

	// delete collection object from mongo - status HIDDEN and set delete date
	delResponse, err := a.collectionService.DeleteCollection(boardID, postID, collectionID, strconv.Itoa(ctx.Profile))
	if err != nil {
		return errors.Wrap(err, "unable to soft delete collection")
	}
	if delResponse["status"] == 0 {
		json.NewEncoder(w).Encode(res)
		return nil
	}
	// delete collection things from mongo - status HIDDEN and set delete date
	var collectionThings []model.Things
	if _, ok := res["data"].(*model.Collection); ok {
		collectionThings = res["data"].(*model.Collection).Things
	}
	for i := range collectionThings {
		thingID := collectionThings[i].ThingID.Hex()
		_, err := a.fileService.DeleteFile(boardID, postID, thingID, ctx.Profile)
		if err != nil {
			// json.NewEncoder(w).Encode(delFileRes)
			fmt.Println("fileObjectID", thingID)
			return errors.Wrap(err, "unable to delete file from mongo")
		} else {

			_, err = a.thingService.FlagBookmarkForDelete(ctx.Profile, thingID, time.Now())
			if err != nil {
				fmt.Println(err.Error())
				return errors.Wrap(err, "unable to update flag in bookmark")
			}

			fmt.Println("file deleted successfully")
		}
	}
	json.NewEncoder(w).Encode(delResponse)
	return err
}
