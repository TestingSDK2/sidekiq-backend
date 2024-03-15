package thing

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/consts"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/permissions"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/util"
	model "github.com/TestingSDK2/sidekiq-backend/sidekiq-models"
	notfrpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-notification/v1"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (a *api) LikeThing(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	thingID := ctx.Vars["thingID"]
	thingType := ctx.Vars["thingType"]
	if thingID == "" || thingType == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	thingTypes := []string{"NOTE", "TASK", "FILE", "POST", "COLLECTION", "BOARD"}
	if !strings.Contains(strings.Join(thingTypes, ","), strings.ToUpper(thingType)) {
		json.NewEncoder(w).Encode((util.SetResponse(nil, 0, "Possible values of thingType can be NOTE, TASK, FILE, POST, COLLECTION, BOARD")))
		w.WriteHeader(http.StatusUnprocessableEntity)
		return nil
	}

	if strings.ToUpper(thingType) == "BOARD" {
		likeRes, err := a.thingService.LikeThing(thingID, thingType, ctx.Profile)
		if err != nil {
			return errors.Wrap(err, "unable to like things")
		}

		json.NewEncoder(w).Encode(likeRes)
		return nil
	} else {
		post, err := a.thingService.GetPostDetailsByThingID(thingID, thingType)
		if err != nil {
			return errors.Wrap(err, "unable to get post details from thingID")
		}

		if strings.ToUpper(thingType) == "POST" {
			if post.Reactions {
				likeRes, err := a.thingService.LikeThing(thingID, thingType, ctx.Profile)
				if err != nil {
					return errors.Wrap(err, "unable to like post")
				}

				json.NewEncoder(w).Encode(likeRes)
				return nil
			} else {
				res := util.SetResponse(nil, 0, "Reactions are currently turned off for this post")
				json.NewEncoder(w).Encode(res)
				return nil
			}
		} else {
			if post.ThingOptSettings {
				likeRes, err := a.thingService.LikeThing(thingID, thingType, ctx.Profile)
				if err != nil {
					return errors.Wrap(err, "unable to like things")
				}

				json.NewEncoder(w).Encode(likeRes)
				return nil
			} else {
				res := util.SetResponse(nil, 0, "Reactions are currently turned off for this thing.")
				json.NewEncoder(w).Encode(res)
				return nil
			}
		}
	}

}

func (a *api) DislikeThing(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	thingID := ctx.Vars["thingID"]
	thingType := ctx.Vars["thingType"]
	if thingID == "" || thingType == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	thingTypes := []string{"NOTE", "TASK", "FILE", "POST", "COLLECTION", "BOARD"}
	if !strings.Contains(strings.Join(thingTypes, ","), strings.ToUpper(thingType)) {
		json.NewEncoder(w).Encode((util.SetResponse(nil, 0, "Possible values of thingType can be NOTE, TASK, FILE, POST, COLLECTION, BOARD")))
		w.WriteHeader(http.StatusUnprocessableEntity)
		return nil
	}

	taskRes, err := a.thingService.DislikeThing(thingID, thingType, ctx.Profile)
	if err == nil {
		json.NewEncoder(w).Encode(taskRes)
	}
	return err
}
func (a *api) AddComment2(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	boardID := ctx.Vars["boardID"]
	thingID := ctx.Vars["thingID"]
	thingType := ctx.Vars["thingType"]
	if thingID == "" || thingType == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	thingTypes := []string{consts.NoteType, consts.TaskType, consts.FileType, consts.PostType, consts.CollectionType, consts.BoardType}
	if !strings.Contains(strings.Join(thingTypes, ","), strings.ToUpper(thingType)) {
		json.NewEncoder(w).Encode((util.SetResponse(nil, 0, "Possible values of thingType can be NOTE, TASK, FILE, POST, COLLECTION, BOARD")))
		w.WriteHeader(http.StatusUnprocessableEntity)
		return nil
	}

	var payload map[string]string
	var commentRes map[string]interface{}
	var err error
	var isNotf bool
	err = json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode user json")
	}

	// get the thing
	thingData, err := a.thingService.GetThingBasedOffIDAndType(thingID, thingType)
	if err != nil {
		return err
	}
	// get the board
	boardres, err := a.boardService.FetchBoardDetailsByID(boardID)
	if err != nil {
		return errors.Wrap(err, "unable to get board details")
	}

	senderId := ctx.Profile
	receiverID, err := strconv.Atoi(string(boardres["data"].(*model.Board).Owner))
	if err != nil {
		return errors.Wrap(err, "unable to convert to integer")
	}

	// check if notification is allowed
	if val, ok := thingData["isReactions"]; ok { // if value exists
		if val.(bool) {
			// add comment
			commentRes, err = a.thingService.AddThingComment2(thingID, thingType, ctx.Profile, payload["comment"])
			if err != nil {
				return errors.Wrap(err, "Unable to add comment on things")
			}
			isNotf = true
			// send the notification
			// CALL GRPC
			request := notfrpc.NotificationHandlerRequest{
				ReceiverIDs: []int32{int32(receiverID)},
				SenderID:    int32(senderId),
				ThingType:   strings.ToUpper(thingType),
				ActionType:  consts.AddComment,
				Message:     "",
				ThingID:     thingID,
			}
			_, err = a.repos.NotificationGrpcServiceClient.NotificationHandler(context.TODO(), &request)
			if err != nil {
				return errors.Wrap(err, "unable send and create notification")
			}
		}
	}
	if !isNotf {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Your reactions are not turned on. Please turn them on to add your reactions."))
		return nil
	}

	json.NewEncoder(w).Encode(commentRes)
	return nil
}

func (a *api) AddComment(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	thingID := ctx.Vars["thingID"]
	thingType := ctx.Vars["thingType"]
	if thingID == "" || thingType == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	thingTypes := []string{consts.NoteType, consts.TaskType, consts.FileType, consts.PostType, consts.CollectionType, consts.BoardType}
	if !strings.Contains(strings.Join(thingTypes, ","), strings.ToUpper(thingType)) {
		json.NewEncoder(w).Encode((util.SetResponse(nil, 0, "Possible values of thingType can be NOTE, TASK, FILE, POST, COLLECTION, BOARD")))
		w.WriteHeader(http.StatusUnprocessableEntity)
		return nil
	}

	var payload map[string]string
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode user json")
	}

	var commentRes map[string]interface{}
	isNotificationRequired := false

	if strings.ToUpper(thingType) == "BOARD" {
		boardres, err := a.boardService.FetchBoardDetailsByID(thingID)
		if err != nil {
			return errors.Wrap(err, "unable to get board details")
		}
		if boardres["data"].(*model.Board).Reactions {
			commentRes, _, err = a.thingService.AddThingComment(thingID, thingType, ctx.Profile, payload["comment"])
			if err != nil {
				return errors.Wrap(err, "Unable to add comment on things")
			}
			if boardres != nil {
				if boardres["data"] != nil && boardres["status"] == 1 {
					if boardData, ok := boardres["data"].(*model.Board); ok {
						if boardData.Owner != "" {
							senderId := ctx.Profile
							receiverID, err := strconv.Atoi(string(boardData.Owner))
							if err != nil {
								return errors.Wrap(err, "unable to parse AssignedToID")
							}
							// err = a.notificationService.NotificationHandler(receiverID, senderId, a.clientMgr, strings.ToUpper(thingType), thingID, consts.AddComment, "")
							// if err != nil {
							// 	return errors.Wrap(err, "unable send and create notification")
							// }

							// CALL GRPC
							request := notfrpc.NotificationHandlerRequest{
								ReceiverIDs: []int32{int32(receiverID)},
								SenderID:    int32(senderId),
								ThingType:   strings.ToUpper(thingType),
								ActionType:  consts.AddComment,
								Message:     "",
								ThingID:     thingID,
							}
							_, err = a.repos.NotificationGrpcServiceClient.NotificationHandler(context.TODO(), &request)
							if err != nil {
								return errors.Wrap(err, "unable send and create notification")
							}
						}
					}
				}
			}

		}

		if isNotificationRequired {
		}
		json.NewEncoder(w).Encode(commentRes)
		return nil
	} else {
		post, err := a.thingService.GetPostDetailsByThingID(thingID, thingType)
		if err != nil {
			return errors.Wrap(err, "unable to get post details from thingID")
		}

		if strings.ToUpper(thingType) == "POST" {
			if post.Reactions {
				commentRes, err = a.thingService.AddPostComment(post, ctx.Profile, payload["comment"])
				if err != nil {
					return errors.Wrap(err, "Unable to add comment on post")
				}
				isNotificationRequired = true
			} else {
				isNotificationRequired = false
				commentRes = util.SetResponse(nil, 0, "Reactions are currently turned off for this post")
			}
		} else {
			if post.ThingOptSettings {
				commentRes, _, err = a.thingService.AddThingComment(thingID, thingType, ctx.Profile, payload["comment"])
				if err != nil {
					return errors.Wrap(err, "Unable to add comment on things")
				}
				isNotificationRequired = true
			} else {
				isNotificationRequired = false
				commentRes = util.SetResponse(nil, 0, "Reactions are currently turned off for this thing.")
			}
		}

		if isNotificationRequired && post.Owner != "" {
			senderId := ctx.Profile
			receiverID, err := strconv.Atoi(string(post.Owner))
			if err != nil {
				return errors.Wrap(err, "unable to parse AssignedToID")
			}

			fmt.Println(senderId, receiverID)
			// CALL GRPC
			// err = a.notificationService.NotificationHandler(receiverID, senderId, a.clientMgr, strings.ToUpper(thingType), thingID, consts.AddComment, "")
			// if err != nil {
			// 	return errors.Wrap(err, "unable send and create notification")
			// }
		}

		json.NewEncoder(w).Encode(commentRes)
	}

	return nil
}

func (a *api) DeleteComment(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	thingID := ctx.Vars["thingID"]
	thingType := ctx.Vars["thingType"]
	commentID := ctx.Vars["commentID"]
	if thingID == "" || thingType == "" || commentID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	thingTypes := []string{"NOTE", "TASK", "FILE", "POST", "COLLECTION", "BOARD"}
	if !strings.Contains(strings.Join(thingTypes, ","), strings.ToUpper(thingType)) {
		json.NewEncoder(w).Encode((util.SetResponse(nil, 0, "Possible values of thingType can be NOTE, TASK, FILE, POST, COLLECTION, BOARD")))
		w.WriteHeader(http.StatusUnprocessableEntity)
		return nil
	}

	taskRes, err := a.thingService.DeleteComment(thingID, thingType, commentID, ctx.Profile)
	if err == nil {
		json.NewEncoder(w).Encode(taskRes)
	}
	return err
}

func (a *api) EditComment(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	thingID := ctx.Vars["thingID"]
	thingType := ctx.Vars["thingType"]
	commentID := ctx.Vars["commentID"]
	if thingID == "" || thingType == "" || commentID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	var payload map[string]string
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode user json")
	}

	thingTypes := []string{"NOTE", "TASK", "FILE", "POST", "COLLECTION", "BOARD"}
	if !strings.Contains(strings.Join(thingTypes, ","), strings.ToUpper(thingType)) {
		json.NewEncoder(w).Encode((util.SetResponse(nil, 0, "Possible values of thingType can be NOTE, TASK, FILE, POST, COLLECTION, BOARD")))
		w.WriteHeader(http.StatusUnprocessableEntity)
		return nil
	}

	taskRes, err := a.thingService.EditComment(thingID, thingType, commentID, payload, ctx.Profile)
	if err == nil {
		json.NewEncoder(w).Encode(taskRes)
	}
	return err
}

func (a *api) OpenThing(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	boardID := ctx.Vars["boardID"]
	thingType := ctx.Vars["thingType"]
	thingID := ctx.Vars["thingID"]

	if thingID == "" || thingType == "" || boardID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	thingTypes := []string{"note", "task", "file", "post", "collection"}
	if !strings.Contains(strings.Join(thingTypes, ","), strings.ToLower(thingType)) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return nil
	}

	var thingRes map[string]interface{}
	var thingDisplayTitle string
	var err error

	switch strings.ToUpper(thingType) {
	case "NOTE":
		thingRes, err = a.noteService.GetNoteByID(thingID, ctx.Profile)
		if err != nil {
			return errors.Wrap(err, "unable to fetch note")
		}

		thingDisplayTitle = thingRes["data"].(map[string]interface{})["title"].(string)
	case "TASK":
		thingRes, err = a.taskService.GetTaskByID(thingID, ctx.Profile)
		if err != nil {
			return errors.Wrap(err, "unable to fetch task")
		}

		thingDisplayTitle = thingRes["data"].(map[string]interface{})["title"].(string)
	case "FILE":
		thingRes, err = a.fileService.GetFileByID(thingID, ctx.Profile)
		if err != nil {
			return errors.Wrap(err, "unable to fetch file")
		}

		thingDisplayTitle = thingRes["data"].(model.UploadedFile).Title

	case "POST":
		thingRes, err = a.postService.FindPostByPostID(thingID)
		if err != nil {
			return errors.Wrap(err, "unable to fetch post")
		}
		thingDisplayTitle = thingRes["data"].(model.Post).Title
	case "COLLECTION":
		thingRes, err = a.collectionService.GetCollectionByID(boardID, "", thingID, ctx.Profile)
		if err != nil {
			return errors.Wrap(err, "unable to fetch collection")
		}
		thingDisplayTitle = thingRes["data"].(model.Collection).Title
	}

	// store open thing to recent-things
	thingObjID, err := primitive.ObjectIDFromHex(thingID)
	if err != nil {
		return errors.Wrap(err, "unable to convert to prmititve ObjectID")
	}
	boardObjID, err := primitive.ObjectIDFromHex(boardID)
	if err != nil {
		return errors.Wrap(err, "unable to convert to primititve ObjectID")
	}

	recentThing := model.Recent{
		ThingID:      thingObjID,
		DisplayTitle: thingDisplayTitle,
		BoardID:      boardObjID,
		ProfileID:    strconv.Itoa(ctx.Profile),
		ThingType:    strings.ToUpper(thingType),
	}

	err = a.recentThingsService.AddToDashBoardRecent(recentThing)
	if err != nil {
		return errors.Wrap(err, "unable to add to recent thing")
	}

	json.NewEncoder(w).Encode(thingRes)
	return nil
}

func (a *api) GetAllBookmarks(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	if ctx.Profile == -1 {
		res := util.SetResponse(nil, 0, "Profile not authorized")
		json.NewEncoder(w).Encode(res)
		return nil
	}
	// pagination calculation
	query := r.URL.Query()
	limit := query.Get("limit")
	page := query.Get("page")
	filterByThing := query.Get("filterBy")

	if filterByThing != "" && strings.ToLower(filterByThing) != "all" {
		thingTypes := []string{"note", "task", "file", "post", "board", "collection"}
		if !strings.Contains(strings.Join(thingTypes, ","), strings.ToLower(filterByThing)) {
			json.NewEncoder(w).Encode("Possible value for filter is note, task, file, post, board and collection")
			w.WriteHeader(http.StatusUnprocessableEntity)
			return nil
		}
	}

	var limitInt, pageInt int
	var err error
	if limit != "" && page != "" {
		pageInt, err = strconv.Atoi(page)
		if err != nil {
			return errors.Wrap(err, "unable to convert string to int")
		}
		limitInt, err = strconv.Atoi(limit)
		if err != nil {
			return errors.Wrap(err, "unable to convert string to int")
		}
	} else {
		limitInt = 10
		pageInt = 0
	}
	res, err := a.thingService.FetchBookmarks(ctx.User.ID, ctx.Profile, limitInt, pageInt, "", "", filterByThing)
	if err == nil {
		json.NewEncoder(w).Encode(res)
		return nil
	}
	return err
}

func (a *api) AddBookmark(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	if ctx.Profile == -1 {
		// Profile not authorized to perform this action
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}
	var payload model.Bookmark
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode user json")
	}
	payload.ProfileID = ctx.Profile
	res, err := a.thingService.AddBookmark(payload)
	if err == nil {
		json.NewEncoder(w).Encode(res)
		return nil
	}
	return err
}

func (a *api) DeleteBookmark(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	if ctx.Profile == -1 {
		// Profile not authorized to perform this action
		res := util.SetResponse(nil, 0, "Profile not authorized")
		json.NewEncoder(w).Encode(res)
		return nil
	}
	if ctx.Vars["bookmarkID"] == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}
	res, err := a.thingService.DeleteBookmark(ctx.Profile, ctx.Vars["bookmarkID"])
	if err == nil {
		json.NewEncoder(w).Encode(res)
		return nil
	}
	return err
}

func (a *api) DeleteAllBookmarks(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	if ctx.Profile == -1 {
		// Profile not authorized to perform this action
		res := util.SetResponse(nil, 0, "Profile not authorized")
		json.NewEncoder(w).Encode(res)
		return nil
	}
	res, err := a.thingService.DeleteAllBookmarks(ctx.Profile)
	if err == nil {
		json.NewEncoder(w).Encode(res)
		return nil
	}
	return err
}

func (a *api) UpdateThing(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	profileID := ctx.Profile
	if profileID == -1 {
		return json.NewEncoder(w).Encode(util.SetResponse(nil, 1, "Profile not authorized"))
	}
	var payload, res map[string]interface{}
	var err error

	err = json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return err
	}

	thingID := ctx.Vars["thingID"]
	thingType := ctx.Vars["thingType"]
	boardID := ctx.Vars["boardID"]
	postID := ctx.Vars["postID"]

	if boardID == "" || thingType == "" {
		w.WriteHeader(http.StatusBadRequest)
	}
	var collName string

	switch strings.ToUpper(thingType) {
	case "BOARD":
		collName = consts.Board
	case "NOTE":
		collName = consts.Note
	case "FILE":
		collName = consts.File
	case "TASK":
		collName = consts.Task
	default:
		return json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Unable to match thing type."))
	}

	dbconn, err := a.db.New(consts.Board)
	if err != nil {
		return err
	}
	boardColl, boardClient := dbconn.Collection, dbconn.Client
	defer boardClient.Disconnect(context.TODO())

	profileIDStr := strconv.Itoa(profileID)
	isValid, err := permissions.CheckValidPermissions(profileIDStr, a.cache, boardColl, boardID, []string{"owner", "admin"}, false)
	if err != nil {
		return err
	}
	if !isValid {
		return json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "You don't have the permission to update"))
	}

	b, _ := json.Marshal(payload)

	switch collName {
	case consts.Board:
		var board map[string]interface{}
		json.Unmarshal(b, &board)
		res, err = a.boardService.UpdateBoard(board, boardID, profileID)
		fmt.Println("updated board: ", res["data"].(*model.Board))
	case consts.Note:
		var note map[string]interface{}
		json.Unmarshal(b, &note)
		res, err = a.noteService.UpdateNote(note, boardID, postID, thingID, profileID)
	case consts.Task:
		var task map[string]interface{}
		json.Unmarshal(b, &task)
		res, err = a.taskService.UpdateTask(task, boardID, postID, thingID, profileID)
	case consts.File:
		var file map[string]interface{}
		json.Unmarshal(b, &file)
		res, err = a.fileService.UpdateFile(file, boardID, postID, thingID, profileID)
	}

	if err == nil {
		return json.NewEncoder(w).Encode(res)
	}
	return err
}

func (a *api) FetchReactions(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	thingID := ctx.Vars["thingID"]
	thingType := ctx.Vars["thingType"]
	rectionType := ctx.Vars["reactionType"]
	if thingID == "" || thingType == "" || rectionType == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	thingTypes := []string{"NOTE", "TASK", "FILE", "POST", "COLLECTION", "BOARD"}
	if !strings.Contains(strings.Join(thingTypes, ","), strings.ToUpper(thingType)) {
		json.NewEncoder(w).Encode((util.SetResponse(nil, 0, "Possible values of thingType can be NOTE, TASK, FILE, POST, COLLECTION, BOARD")))
		w.WriteHeader(http.StatusUnprocessableEntity)
		return nil
	}

	if strings.ToUpper(rectionType) != "COMMENTS" && strings.ToUpper(rectionType) != "LIKES" {
		json.NewEncoder(w).Encode((util.SetResponse(nil, 0, "Possible values of rectionType can be COMMENTS,LIKES")))
		w.WriteHeader(http.StatusUnprocessableEntity)
		return nil
	}

	// pagination calculation
	query := r.URL.Query()
	limit := query.Get("limit")
	page := query.Get("page")
	var limitInt, pageInt int
	var err error
	if limit != "" && page != "" {
		pageInt, err = strconv.Atoi(page)
		if err != nil {
			return errors.Wrap(err, "unable to convert string to int")
		}
		limitInt, err = strconv.Atoi(limit)
		if err != nil {
			return errors.Wrap(err, "unable to convert string to int")
		}
	} else {
		limitInt = 10
		pageInt = 0
	}
	taskRes, err := a.thingService.FetchReactions(thingID, thingType, rectionType, ctx.Profile, limitInt, pageInt)
	if err == nil {
		json.NewEncoder(w).Encode(taskRes)
	}

	return err
}

// func (a *api) TrashThing(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
// 	profileID := ctx.Profile
// 	if profileID == -1 {
// 		return json.NewEncoder(w).Encode(util.SetResponse(nil, 1, "Profile not authorized"))
// 	}
// 	var payload, res map[string]interface{}
// 	var err error

// 	err = json.NewDecoder(r.Body).Decode(&payload)
// 	if err != nil {
// 		return err
// 	}

// 	thingID := ctx.Vars["thingID"]
// 	thingType := ctx.Vars["thingType"]
// 	boardID := ctx.Vars["boardID"]

// 	if boardID == "" || thingType == "" {
// 		w.WriteHeader(http.StatusBadRequest)
// 	}
// 	return nil
// }
