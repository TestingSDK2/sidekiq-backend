package board

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/thing"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/consts"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/helper"
	model "github.com/TestingSDK2/sidekiq-backend/sidekiq-models"
	"github.com/pkg/errors"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/permissions"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/util"

	notfrpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-notification/v1"
	peoplerpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"
	searchrpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-search/v1"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FetchBoardThings - fetches all the things of a board
func (a *api) FetchBoardThings(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	var payload map[string]interface{}
	var isBoardFollower bool = false
	var showBoardThings bool = false
	var isClientAccessible bool = false
	var isPasswordValidate bool = false
	var isPassword bool = false
	var hasSentPass bool = false
	var err error
	var tagArr []string
	var totalGoRoutines int

	// query params for filtering
	// Get the value of the query parameter "tags"
	tags := r.URL.Query().Get("tags")
	if tags != "" {
		// Split the parameter into individual values
		tagArr = strings.Split(tags, ",")
	}
	fileType := r.URL.Query().Get("fileType")
	owner := r.URL.Query().Get("owner")
	d := make(map[string]interface{})
	uploadDate := r.URL.Query().Get("uploadDate")
	errChan := make(chan error)
	limit := 10
	response := make(map[string]interface{})
	response["data"] = make(map[string]interface{})
	// var subBoardRes, taskRes, noteRes, fileRes, collectionRes map[string]interface{}
	var subBoardRes, taskRes, noteRes, fileRes map[string]interface{}
	var profileKey string
	var ownerBoardPermission model.BoardPermission

	profileID := ctx.Profile
	if profileID == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	var mutex sync.Mutex
	boardID := ctx.Vars["boardID"]
	if boardID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	// check if user has send password or not
	err = json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		if err.Error() == "EOF" { // no body
			hasSentPass = false
		}
	} else {
		hasSentPass = true
	}

	// get the board
	profileKey = fmt.Sprintf("boards:%s", strconv.Itoa(ctx.Profile))
	boardMap, err := a.boardService.FetchBoardByID(boardID, strconv.Itoa(ctx.Profile))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, err.Error()))
		log.Println(err, "unable to find board by id")
		return nil
	}
	boardObj, ok := boardMap["data"].(*model.Board)
	if !ok {
		return errors.Wrap(err, "unable to assert board")
	}
	boardInfo, err := a.boardService.FetchBoardInfo(boardID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "unable to find board info"))
		log.Println(err, "unable to find board info")
		return nil
	}
	ownerBoardPermission = permissions.GetBoardPermissionsNew(profileKey, a.cache, boardObj, strconv.Itoa(ctx.Profile))
	role := ownerBoardPermission[boardID]

	fmt.Println("board's visible @ : ", boardObj.Visible)
	fmt.Println("board password: ", boardInfo["isPassword"].(bool), boardInfo["password"].(string))

	// setting default response IF NOT sending data!
	d["_id"] = boardObj.Id
	d["owner"] = boardObj.Owner
	d["title"] = boardObj.Title
	if boardInfo["password"].(string) != "" {
		isPassword = true
		isPasswordValidate = false
		d["isPassword"] = isPassword
		d["isPasswordValidate"] = isPasswordValidate
	} else {
		isPassword = false
		isPasswordValidate = true
		d["isPassword"] = isPassword
		d["isPasswordValidate"] = isPasswordValidate
	}

	if util.Contains(boardObj.Followers, strconv.Itoa(profileID)) {
		isBoardFollower = true
	}
	response["data"].(map[string]interface{})["isBoardFollower"] = isBoardFollower

	// if user has not sent password, check flags: permissions, board password, accessible or not
	fmt.Println("role ", role)
	if !hasSentPass {
		// get the permissions of the profile
		if boardObj.Visible == "PRIVATE" {
			if role == "" || role == "blocked" { // not accessible
				isClientAccessible = false
				isPassword = false
				isPasswordValidate = false
				d["isPassword"] = isPassword
				d["isPasswordValidate"] = isPasswordValidate
				d["isClientAccessible"] = isClientAccessible
				json.NewEncoder(w).Encode(util.SetResponse(d, 0, "User does not have the access"))
				return nil
			}
			isClientAccessible = true
			d["isClientAccessible"] = isClientAccessible
			if isClientAccessible && !isPassword && isPasswordValidate {
				showBoardThings = true
				goto X
			} else if isClientAccessible && isPassword && !isPasswordValidate {
				d["isPasswordValidate"] = false
				json.NewEncoder(w).Encode(util.SetResponse(d, 1, ""))
				return nil
			}
		} else if boardObj.Visible == "PUBLIC" { // PUBLIC board
			fmt.Println("140: ")
			if role == "blocked" { // not accessible
				// not accessible
				isClientAccessible = false
				isPassword = false
				isPasswordValidate = false
				d["isPassword"] = isPassword
				d["isPasswordValidate"] = isPasswordValidate
				d["isClientAccessible"] = isClientAccessible
				json.NewEncoder(w).Encode(util.SetResponse(d, 0, "User does not have the access"))
				return nil
			} else { // accessible
				isClientAccessible = true
				d["isClientAccessible"] = isClientAccessible
				if isPassword {
					isPasswordValidate = false
					d["isPasswordValidate"] = isPasswordValidate
					json.NewEncoder(w).Encode(util.SetResponse(d, 1, ""))
					return nil
				} else { // no password
					isPasswordValidate = true
					showBoardThings = true
					goto X
				}
			}
		}
	} else { // user has sent password
		// check the accessibility when the user has entered the password
		if boardObj.Visible == "PRIVATE" {
			if role != "blocked" {
				isClientAccessible = true
			} else {
				isClientAccessible = false
			}
		} else if boardObj.Visible == "PUBLIC" {
			isClientAccessible = true
		}
		// validate the password
		if payload["password"].(string) == boardInfo["password"].(string) {
			isPasswordValidate = true
			isClientAccessible = true
			showBoardThings = true
			goto X
		} else {
			d["isPasswordValidate"] = false
			d["isClientAccessible"] = isClientAccessible
			json.NewEncoder(w).Encode(util.SetResponse(d, 0, "Invalid password"))
		}
	}

X:
	if showBoardThings {
		totalGoRoutines += 1
		// fetch the board object
		go func(errChan chan<- error) {
			defer util.RecoverGoroutinePanic(errChan)
			log.Printf("Started --------> board")
			boardRes, err := a.boardService.FetchBoardByID(boardID, role)
			if err != nil {
				errChan <- errors.Wrap(err, "error from fetch board details")
			}
			if boardRes["data"] != nil {
				mutex.Lock()
				response["data"].(map[string]interface{})["board"] = boardRes["data"].(*model.Board)
				boardThingsTags, err := a.boardService.GetBoardThingsTags(boardID)
				if err != nil {
					errChan <- errors.Wrap(err, "error from fetch board things tags")
				}
				response["data"].(map[string]interface{})["board"].(*model.Board).AllThingsTags = boardThingsTags
				mutex.Unlock()
			} else {
				mutex.Lock()
				response["data"].(map[string]interface{})["board"] = nil
				mutex.Unlock()
			}
			log.Printf("done ------> board")
			errChan <- nil
		}(errChan)

		if fileType == "" {
			totalGoRoutines += 4
			// sub - boards
			go func(errChan chan<- error) {
				defer util.RecoverGoroutinePanic(errChan)
				log.Printf("started --------> sub-boards")
				subBoardRes, err = a.boardService.FetchSubBoards(boardID, profileID, limit)
				if err != nil {
					log.Println("err from sub-boards:", err)
					errChan <- errors.Wrap(err, "error in fetching sub-boards of the board ")
				}
				if subBoardRes["data"] != nil {
					mutex.Lock()
					response["data"].(map[string]interface{})["subBoards"] = subBoardRes["data"].([]*model.Board)
					mutex.Unlock()
				} else {
					mutex.Lock()
					response["data"].(map[string]interface{})["subBoards"] = nil
					mutex.Unlock()
				}
				log.Printf("done --------> sub-boards")
				errChan <- nil
			}(errChan)

			// tasks
			go func(errChan chan<- error) {
				defer util.RecoverGoroutinePanic(errChan)
				log.Printf("Started --------> tasks")
				taskRes, err = a.taskService.FetchTasksOfBoard(boardID, profileID, owner, tagArr, uploadDate, limit, "", "")
				if err != nil {
					log.Println("err from tasks:", err)
					errChan <- errors.Wrap(err, "error in fetching board tasks ")
				}
				if taskRes["data"] != nil {
					mutex.Lock()
					if reflect.TypeOf(taskRes["data"]) == reflect.TypeOf([]*model.Task{}) {
						if len(taskRes["data"].([]*model.Task)) > 0 {
							response["data"].(map[string]interface{})["tasks"] = taskRes["data"].([]*model.Task)
						} else {
							response["data"].(map[string]interface{})["tasks"] = nil
						}
					} else {
						if len(taskRes["data"].(map[string]interface{})["info"].([]*model.Task)) > 0 {
							response["data"].(map[string]interface{})["tasks"] = taskRes["data"].(map[string]interface{})["info"].([]*model.Task)
						} else {
							response["data"].(map[string]interface{})["tasks"] = nil
						}
					}
					mutex.Unlock()
				} else {
					mutex.Lock()
					response["data"].(map[string]interface{})["tasks"] = nil
					mutex.Unlock()
				}
				log.Printf("done --------> tasks")
				errChan <- nil
			}(errChan)
			// notes
			go func(errChan chan<- error) {
				defer util.RecoverGoroutinePanic(errChan)
				log.Printf("Started --------> notes")
				noteRes, err = a.noteService.FetchNotesByBoard(boardID, profileID, owner, tagArr, uploadDate, limit, "", "")
				if err != nil {
					log.Println("err from notes:", err)
					errChan <- errors.Wrap(err, "error in fetching board notes ")
				}
				if noteRes["data"] != nil {
					mutex.Lock()
					if reflect.TypeOf(noteRes["data"]) == reflect.TypeOf([]*model.Note{}) {
						if len(noteRes["data"].([]*model.Note)) > 0 {
							response["data"].(map[string]interface{})["notes"] = noteRes["data"].([]*model.Note)
						} else {
							response["data"].(map[string]interface{})["notes"] = nil
						}
					} else {
						if len(noteRes["data"].(map[string]interface{})["info"].([]*model.Note)) > 0 {
							response["data"].(map[string]interface{})["notes"] = noteRes["data"].(map[string]interface{})["info"].([]*model.Note)
						} else {
							response["data"].(map[string]interface{})["notes"] = nil
						}
					}
					mutex.Unlock()
				} else {
					mutex.Lock()
					response["data"].(map[string]interface{})["notes"] = nil
					mutex.Unlock()
				}
				log.Printf("done --------> notes")
				errChan <- nil
			}(errChan)

			// collection
			// go func(errChan chan<- error) {
			// 	defer util.RecoverGoroutinePanic(errChan)
			// 	log.Printf("Started --------> collection")
			// 	collectionRes, err = a.collectionService.GetCollection(a.cache, a.profileService, a.storageService, boardID, profileID, owner, tagArr, uploadDate, limit, "", "")
			// 	if err != nil {
			// 		log.Println("err from collection:", err)
			// 		errChan <- errors.Wrap(err, "error in fetching board collection ")
			// 	}
			// 	if collectionRes["data"] != nil {
			// 		mutex.Lock()
			// 		if reflect.TypeOf(collectionRes["data"]) == reflect.TypeOf([]*model.Collection{}) {
			// 			if len(collectionRes["data"].([]*model.Collection)) > 0 {
			// 				response["data"].(map[string]interface{})["collections"] = collectionRes["data"].([]*model.Collection)
			// 			} else {
			// 				response["data"].(map[string]interface{})["collections"] = nil
			// 			}
			// 		} else {
			// 			if len(collectionRes["data"].(map[string]interface{})["info"].([]*model.Collection)) > 0 {
			// 				response["data"].(map[string]interface{})["collections"] = collectionRes["data"].(map[string]interface{})["info"].([]*model.Collection)
			// 			} else {
			// 				response["data"].(map[string]interface{})["collections"] = nil
			// 			}
			// 		}
			// 		mutex.Unlock()
			// 	} else {
			// 		mutex.Lock()
			// 		response["data"].(map[string]interface{})["collections"] = nil
			// 		mutex.Unlock()
			// 	}
			// 	log.Printf("done --------> collections")
			// 	errChan <- nil
			// }(errChan)
		}

		totalGoRoutines += 1
		// files
		go func(errChan chan<- error) {
			defer util.RecoverGoroutinePanic(errChan)
			log.Printf("Started --------> files")
			fileRes, err = a.fileService.FetchFilesByBoard(boardID, ctx.Profile, fileType, owner, tagArr, uploadDate, limit, "", "")
			if err != nil {
				log.Println("err from files:", err)
				errChan <- errors.Wrap(err, "error in fetching board files ")
			}
			if fileRes["data"] != nil {
				mutex.Lock()
				if reflect.TypeOf(fileRes["data"]) == reflect.TypeOf([]*model.UploadedFile{}) {
					response["data"].(map[string]interface{})["files"] = fileRes["data"].([]*model.UploadedFile)
				} else {
					response["data"].(map[string]interface{})["files"] = fileRes["data"].(map[string]interface{})["info"].([]*model.UploadedFile)
				}
				mutex.Unlock()
			} else {
				mutex.Lock()
				response["data"].(map[string]interface{})["files"] = nil
				mutex.Unlock()
			}
			log.Printf("done --------> files")
			errChan <- nil
		}(errChan)

		for totalGoRoutines != 0 {
			totalGoRoutines--
			if err := <-errChan; err != nil {
				return errors.Wrap(err, "error from go routine")
			}
		}
		if fileType != "" {
			response["data"].(map[string]interface{})["tasks"] = nil
			response["data"].(map[string]interface{})["collections"] = nil
			response["data"].(map[string]interface{})["notes"] = nil
		}
		if len(tagArr) != 0 || fileType != "" || owner != "" || uploadDate != "" {
			response["data"].(map[string]interface{})["recentlyAdded"] = nil
		}
		response["data"].(map[string]interface{})["role"] = ownerBoardPermission[boardID]
		response["data"].(map[string]interface{})["isPassword"] = isPassword
		response["data"].(map[string]interface{})["isPasswordValidate"] = isPasswordValidate
		response["data"].(map[string]interface{})["isClientAccessible"] = isClientAccessible
		response["status"] = 1
		response["message"] = "Board things fetched successfully."
		json.NewEncoder(w).Encode(response)
	}
	return err
}

// FetchBoards - fetches all boards in which the current user is viewer, member, admin or owner
func (a *api) FetchBoards(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	limit := r.URL.Query().Get("limit")
	page := r.URL.Query().Get("page")
	boardsRes, err := a.boardService.FetchBoards(ctx.Profile, true, page, limit)
	if err == nil {
		json.NewEncoder(w).Encode(boardsRes)
		return nil
	}

	return err
}

func (a *api) SetIsBookmarkInBoard(resData map[string]interface{}, profileID int, thingService thing.Service) map[string]interface{} {
	if resData["data"] != nil && resData["data"].(map[string]interface{})["info"] != nil {
		infoData := resData["data"].(map[string]interface{})["info"]
		if boards, ok := infoData.([]*model.Board); ok {
			// run this in goroutine
			for idx := range boards {
				isbookmarked, bid, err := thingService.IsBookMarkedByProfile(boards[idx].Id.Hex(), profileID)
				if err != nil {
					boards[idx].BookmarkID = ""
					boards[idx].IsBookmarked = false
					continue
				}

				boards[idx].IsBookmarked = isbookmarked
				boards[idx].BookmarkID = bid
			}

			resData["data"].(map[string]interface{})["info"] = boards
			return resData
		} else if boards, ok := infoData.([]model.Board); ok {
			for idx := range boards {

				isbookmarked, bid, err := thingService.IsBookMarkedByProfile(boards[idx].Id.Hex(), profileID)
				if err != nil {
					boards[idx].BookmarkID = ""
					boards[idx].IsBookmarked = false
					continue
				}

				boards[idx].IsBookmarked = isbookmarked
				boards[idx].BookmarkID = bid
			}

			resData["data"].(map[string]interface{})["info"] = boards
			return resData
		} else if objects, ok := infoData.([]interface{}); ok {
			for idx := range objects {
				if board, ok := objects[idx].(*model.Board); ok {
					isbookmarked, bid, err := thingService.IsBookMarkedByProfile(board.Id.Hex(), profileID)
					if err != nil {
						board.BookmarkID = ""
						board.IsBookmarked = false
						objects[idx] = board
						continue
					}

					board.IsBookmarked = isbookmarked
					board.BookmarkID = bid

					objects[idx] = board
				} else if board, ok := objects[idx].(model.Board); ok {
					isbookmarked, bid, err := thingService.IsBookMarkedByProfile(board.Id.Hex(), profileID)
					if err != nil {
						board.BookmarkID = ""
						board.IsBookmarked = false
						objects[idx] = board
						continue
					}

					board.IsBookmarked = isbookmarked
					board.BookmarkID = bid

					objects[idx] = board
				} else if post, ok := objects[idx].(model.Post); ok {
					isbookmarked, bid, err := thingService.IsBookMarkedByProfile(post.Id.Hex(), profileID)
					if err != nil {
						post.BookmarkID = ""
						post.IsBookmarked = false
						objects[idx] = post
						continue
					}

					post.IsBookmarked = isbookmarked
					post.BookmarkID = bid
					objects[idx] = post
				} else if post, ok := objects[idx].(*model.Post); ok {
					isbookmarked, bid, err := thingService.IsBookMarkedByProfile(post.Id.Hex(), profileID)
					if err != nil {
						post.BookmarkID = ""
						post.IsBookmarked = false
						objects[idx] = post
						continue
					}

					post.IsBookmarked = isbookmarked
					post.BookmarkID = bid
					objects[idx] = post
				} else if thingObj, ok := objects[idx].(map[string]interface{}); ok {

					if thingIDstr, ok := thingObj["_id"].(string); ok {
						isbookmarked, bid, err := thingService.IsBookMarkedByProfile(thingIDstr, profileID)
						if err != nil {
							thingObj["isBookmarked"] = false
							thingObj["bookmarkID"] = ""
							objects[idx] = thingObj
							continue
						}

						thingObj["isBookmarked"] = isbookmarked
						thingObj["bookmarkID"] = bid
						objects[idx] = thingObj
					} else if thingobjID, ok := thingObj["_id"].(primitive.ObjectID); ok {
						isbookmarked, bid, err := thingService.IsBookMarkedByProfile(thingobjID.Hex(), profileID)
						if err != nil {
							thingObj["isBookmarked"] = false
							thingObj["bookmarkID"] = ""
							objects[idx] = thingObj
							continue
						}

						thingObj["isBookmarked"] = isbookmarked
						thingObj["bookmarkID"] = bid
						objects[idx] = thingObj
					}

				}
			}

			resData["data"].(map[string]interface{})["info"] = objects
			return resData
		}
	}

	return resData
}

func (a *api) FetchBoardsListing(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	reqType := r.URL.Query().Get("type")
	sortBy := r.URL.Query().Get("sortBy")
	orderBy := r.URL.Query().Get("orderBy")
	search := r.URL.Query().Get("search")
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
		pageInt = 1
	}

	profileID := ctx.Profile
	if profileID == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	switch reqType {
	case "sptl":
		res, err := a.thingService.FetchBookmarks(ctx.User.ID, ctx.Profile, limitInt, pageInt, sortBy, orderBy, "")
		if err != nil {
			json.NewEncoder(w).Encode(util.SetResponse(nil, 0, err.Error()))
			return nil
		}

		json.NewEncoder(w).Encode(res)
		return nil

	case "rcnt":
		// handle for Recent
		recentThings, err := a.recentThingsService.FetchDashBoardRecentThings(search, profileID, sortBy, orderBy, limitInt, pageInt, true)
		if err != nil {
			json.NewEncoder(w).Encode(util.SetResponse(nil, 0, err.Error()))
			return nil
		}
		json.NewEncoder(w).Encode(recentThings)
		return nil

	case "shrd":
		//handle for Shared
		resSharedBoard, err := a.boardService.GetSharedBoards(ctx.Profile, search, fmt.Sprint(pageInt), fmt.Sprint(limitInt), sortBy, orderBy)
		if err != nil {
			json.NewEncoder(w).Encode(util.SetResponse(nil, 0, err.Error()))
			return nil
		}

		resSharedBoard = a.SetIsBookmarkInBoard(resSharedBoard, ctx.Profile, a.thingService)
		json.NewEncoder(w).Encode(resSharedBoard)
		return nil

	case "flwd":
		//handle for Followed
		boards, err := a.boardService.FetchFollowedBoards(search, profileID, limitInt, pageInt, sortBy, orderBy)
		if err != nil {
			json.NewEncoder(w).Encode(util.SetResponse(nil, 0, err.Error()))
			return nil
		}

		boards = a.SetIsBookmarkInBoard(boards, ctx.Profile, a.thingService)
		json.NewEncoder(w).Encode(boards)
		return nil

	case "drft":
		//handle for Drafted
		boards, err := a.boardService.FetchBoardsAndPostByState(profileID, consts.Draft, limitInt, pageInt, sortBy, orderBy, true, search)
		if err != nil {
			json.NewEncoder(w).Encode(util.SetResponse(nil, 0, err.Error()))
			return nil
		}
		boards = a.SetIsBookmarkInBoard(boards, ctx.Profile, a.thingService)
		json.NewEncoder(w).Encode(boards)
		return nil

	case "hddn":
		//handle for Hidden
		boards, err := a.boardService.FetchBoardsAndPostByState(profileID, consts.Hidden, limitInt, pageInt, sortBy, orderBy, true, search)
		if err != nil {
			json.NewEncoder(w).Encode(util.SetResponse(nil, 0, err.Error()))
			return nil
		}
		boards = a.SetIsBookmarkInBoard(boards, ctx.Profile, a.thingService)
		json.NewEncoder(w).Encode(boards)
		return nil

	case "arch":
		//handle for Archived
		boards, err := a.boardService.FetchBoardsAndPostByState(profileID, consts.Archive, limitInt, pageInt, sortBy, orderBy, true, search)
		if err != nil {
			json.NewEncoder(w).Encode(util.SetResponse(nil, 0, err.Error()))
			return nil
		}
		boards = a.SetIsBookmarkInBoard(boards, ctx.Profile, a.thingService)
		json.NewEncoder(w).Encode(boards)
		return nil
	}

	json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Request type does not match possible value can be sptl,rcnt,shrd,flwd,drft,hddn,arch"))
	return nil
}

func (a *api) RecentRemove(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	var payload model.RecentDeletePayload
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode payload json")
	}

	profileID := ctx.Profile
	if profileID == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	err = a.recentThingsService.DeleteRecentItems(ctx.Profile, payload)
	if err != nil {
		return errors.Wrap(err, "unable to delete")
	}

	json.NewEncoder(w).Encode(util.SetResponse(nil, 1, "Recent item deleted."))
	return nil

}

// FetchBoards - fetches all boards in which the current user is viewer, member, admin or owner
func (a *api) SearchBoards(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	boardName := r.URL.Query().Get("search")
	fmt.Println(boardName)
	boardsRes, err := a.boardService.SearchBoards(ctx.Profile, boardName, true, "", "")
	if err == nil {
		json.NewEncoder(w).Encode(boardsRes)
		return nil
	}

	return err
}

// FetchThingsOwner - fetches owner of the things present in board
func (a *api) FetchBoardThingOwners(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	pID := ctx.Profile
	profileID := strconv.Itoa(pID)
	boardID := ctx.Vars["boardID"]
	if boardID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}
	ownersRes, err := a.boardService.GetBoardThingOwners(boardID, profileID, ctx.User.ID)
	if err == nil {
		json.NewEncoder(w).Encode(ownersRes)
		return nil
	}
	return err
}

// FetchBoardThingExt - fetches extension of the things present in board
func (a *api) FetchBoardThingExt(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	pID := ctx.Profile
	profileID := strconv.Itoa(pID)
	boardID := ctx.Vars["boardID"]
	if boardID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}
	ownersRes, err := a.boardService.FetchBoardThingExt(boardID, profileID, ctx.User.ID)
	if err == nil {
		json.NewEncoder(w).Encode(ownersRes)
		return nil
	}
	return err
}

func (a *api) AddBoard(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	var payload model.Board
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode payload json")
	}

	profileID := ctx.Profile
	if profileID == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	res, err := a.boardService.AddBoard(payload, profileID)
	if err == nil {
		// update the search results
		boardMap := res["data"].(model.Board).ToMap()
		boardMap["_id"] = res["data"].(model.Board).Id

		recentThing := model.Recent{
			ThingID:      res["data"].(model.Board).Id,
			DisplayTitle: res["data"].(model.Board).Title,
			BoardID:      res["data"].(model.Board).Id,
			ProfileID:    strconv.Itoa(ctx.Profile),
			ThingType:    strings.ToUpper("BOARD"),
		}

		err = a.recentThingsService.AddToDashBoardRecent(recentThing)
		if err != nil {
			return errors.Wrap(err, "unable to add to recent thing")
		}

		// fmt.Println("map to input : ")
		// fmt.Printf("%#v\n", boardMap)

		// err = a.searchService.UpdateSearchResults(boardMap, "insert")
		// if err != nil {
		// 	return errors.Wrap(err, "error from updating search result")
		// }

		// call grpc method here
		reqVal, err := util.MapToMapAny(boardMap)
		if err != nil {
			return err
		}
		in := &searchrpc.UpdateSearchResultRequest{
			Data:       reqVal,
			UpdateType: "insert",
			Args:       "",
		}
		grpcRes, err := a.repos.SearchGrpcServiceClient.UpdateSearchResult(context.TODO(), in)
		if err != nil {
			return err
		}
		fmt.Println(res)
		fmt.Println(grpcRes)
		// if err != nil {
		// 	fmt.Println(788)
		// 	return err
		// }
		fmt.Println("after calling grpc method")

		boardIdObj := res["data"].(model.Board).Id // last inserted board id
		if payload.ParentID != "" {
			// get parent boards
			parentBoardIds, pbErr := a.boardService.GetParentBoards(boardIdObj)
			if pbErr != nil {
				return pbErr
			}

			// true: cache parent boards
			perr := permissions.CacheBoardsPermissions(a.cache, true, parentBoardIds, boardIdObj, profileID, res["data"].(model.Board), nil)
			fmt.Println("514: ", perr)
			if perr != nil {
				return perr
			}
		} else {
			perr := permissions.CacheBoardsPermissions(a.cache, false, nil, boardIdObj, ctx.Profile, res["data"].(model.Board), nil)
			if perr != nil {
				return perr
			}
		}

		// update Profile tags
		// p := model.Profile{ID: ctx.Profile}
		// err = a.profileService.UpdateProfileTags(p, res["data"].(model.Board).Tags)
		// if err != nil {
		// 	return errors.Wrap(err, "unable to add update tags")
		// }

		err = a.profileService.UpdateProfileTagsNew(strconv.Itoa(ctx.Profile))
		if err != nil {
			return errors.Wrap(err, "unable to update tags")
		}

		// add to activity
		// cp, _ := a.profileService.FetchConciseProfile(ctx.Profile)
		cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(ctx.Profile)}
		cp, err := a.repos.PeopleGrpcServiceClient.GetConciseProfile(context.TODO(), cpreq)
		if err != nil {
			return err
		}

		msg := model.ThingActivity{}
		actMsg := fmt.Sprintf("%s %s added the Board", cp.FirstName, cp.LastName)
		if payload.ParentID != "" { // sub board
			actMsg += " " + "<b>" + res["data"].(model.Board).Title + "</b>"
		}
		msg.Create(
			res["data"].(model.Board).Id,
			primitive.NilObjectID,
			ctx.Profile,
			strings.ToUpper(consts.Board),
			actMsg,
		)
		err = a.thingActivityService.PushThingActivityToSQS(msg.ToMap())
		if err != nil {
			return errors.Wrap(err, "unable to push to SQS")
		}

		json.NewEncoder(w).Encode(res)
		return nil
	} else if err != nil && res["status"].(int) == 2 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "user do not have access to add board"))
		return nil
	}
	return err
}

func (a *api) FetchBoardByID(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	if ctx.Profile == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	var payload map[string]interface{}
	var isClientAccessible bool = false
	var isPasswordValidate bool = false
	var isPassword bool = false
	var hasSentPass bool = false
	var showBoard bool = false
	var err error
	d := make(map[string]interface{})
	response := make(map[string]interface{})
	response["data"] = make(map[string]interface{})

	boardID := ctx.Vars["boardID"]
	if boardID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	// handle the password
	// check if user has send password or not
	err = json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		if err.Error() == "EOF" { // no body
			hasSentPass = false
		}
	} else {
		hasSentPass = true
	}

	boardInfo, err := a.boardService.FetchBoardInfo(boardID, "admins", "viewers", "authors", "subscribers", "blocked", "guests", "followers")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println(err, "unable to find board info")
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "unable to find board info"))
	}

	var boardObj *model.Board
	jsonBody, err := json.Marshal(boardInfo)
	if err != nil {
		return err
	}
	err = json.Unmarshal(jsonBody, &boardObj)
	if err != nil {
		return err
	}

	if util.Contains(boardObj.Likes, fmt.Sprint(ctx.Profile)) {
		boardObj.IsLiked = true
	}
	boardObj.TotalLikes = len(boardObj.Likes)
	boardObj.TotalComments = len(boardObj.Comments)

	recentThing := model.Recent{
		ThingID:      boardObj.Id,
		DisplayTitle: boardObj.Title,
		BoardID:      boardObj.Id,
		ProfileID:    strconv.Itoa(ctx.Profile),
		ThingType:    "BOARD",
	}

	err = a.recentThingsService.AddToDashBoardRecent(recentThing)
	if err != nil {
		return errors.Wrap(err, "unable to add to recent thing")
	}

	profileKey := fmt.Sprintf("boards:%s", strconv.Itoa(ctx.Profile))
	ownerBoardPermission := permissions.GetBoardPermissionsNew(profileKey, a.cache, boardObj, strconv.Itoa(ctx.Profile))
	role := ownerBoardPermission[boardID]

	// setting default response IF NOT sending data!
	d["_id"] = boardInfo["_id"]
	d["owner"] = boardInfo["owner"]
	d["title"] = boardInfo["title"]
	d["likes"] = boardInfo["likes"]
	d["totalLikes"] = boardInfo["totalLikes"]
	d["isLiked"] = boardInfo["isLiked"]
	d["totalComments"] = boardInfo["totalComments"]

	if boardInfo["password"].(string) != "" {
		isPassword = true
		isPasswordValidate = false
		d["isPassword"] = isPassword
		d["isPasswordValidate"] = isPasswordValidate
	} else {
		isPassword = false
		isPasswordValidate = true
		d["isPassword"] = isPassword
		d["isPasswordValidate"] = isPasswordValidate
	}

	if !hasSentPass {
		// get the permissions of the profile
		if boardInfo["visible"].(string) == strings.ToUpper(consts.Private) {
			if role == "" || role == "blocked" { // not accessible
				isClientAccessible = false
				isPassword = false
				isPasswordValidate = false
				d["isPassword"] = isPassword
				d["isPasswordValidate"] = isPasswordValidate
				d["isClientAccessible"] = isClientAccessible
				json.NewEncoder(w).Encode(util.SetResponse(d, 0, "User does not have the access"))
				return nil
			}
			isClientAccessible = true
			d["isClientAccessible"] = isClientAccessible
			if isClientAccessible && !isPassword && isPasswordValidate {
				showBoard = true
				goto X
			} else if isClientAccessible && isPassword && !isPasswordValidate {
				d["isPasswordValidate"] = false
				json.NewEncoder(w).Encode(util.SetResponse(d, 1, "Please enter password"))
				return nil
			}
		} else if boardInfo["visible"].(string) == strings.ToUpper(consts.Public) { // PUBLIC board
			fmt.Println("140: ")
			if role == "blocked" { // not accessible
				// not accessible
				isClientAccessible = false
				isPassword = false
				isPasswordValidate = false
				d["isPassword"] = isPassword
				d["isPasswordValidate"] = isPasswordValidate
				d["isClientAccessible"] = isClientAccessible
				json.NewEncoder(w).Encode(util.SetResponse(d, 0, "User does not have the access"))
				return nil
			} else { // accessible
				isClientAccessible = true
				d["isClientAccessible"] = isClientAccessible
				if isPassword {
					isPasswordValidate = false
					d["isPasswordValidate"] = isPasswordValidate
					json.NewEncoder(w).Encode(util.SetResponse(d, 1, "Please enter the password"))
					return nil
				} else { // no password
					isPasswordValidate = true
					showBoard = true
					goto X
				}
			}
		}
	} else { // user has sent password
		// check the accessibility when the user has entered the password
		if boardInfo["visible"].(string) == strings.ToUpper(consts.Private) {
			if role != "blocked" {
				isClientAccessible = true
			} else {
				isClientAccessible = false
			}
		} else if boardInfo["visible"].(string) == strings.ToUpper(consts.Public) {
			isClientAccessible = true
		}
		// validate the password
		if payload["password"].(string) == boardInfo["password"].(string) {
			isPasswordValidate = true
			isClientAccessible = true
			showBoard = true
			goto X
		} else {
			d["isPasswordValidate"] = false
			d["isClientAccessible"] = isClientAccessible
			json.NewEncoder(w).Encode(util.SetResponse(d, 0, "Invalid password"))
		}
	}

X:
	if showBoard {
		boardRes, err := a.boardService.FetchBoardByID(boardID, strconv.Itoa(ctx.Profile))
		if err != nil {
			return err
		}

		boardtemp := boardRes["data"].(*model.Board)

		if util.Contains(boardtemp.Likes, fmt.Sprint(ctx.Profile)) {
			boardtemp.IsLiked = true
		}
		boardtemp.TotalLikes = len(boardtemp.Likes)
		boardtemp.TotalComments = len(boardtemp.Comments)

		board := boardtemp.ToMap()

		if role == consts.Owner || role == consts.Admin {
			board["password"] = boardInfo["password"]
		}

		// get role and isBookmarked
		isbookmarked, bid, err := a.thingService.IsBookMarkedByProfile(boardID, ctx.Profile)
		if err != nil {
			return err
		}
		board["isBookmarked"] = isbookmarked
		board["role"] = ownerBoardPermission[boardID]
		if isbookmarked {
			board["bookmarkID"] = bid
		}
		board["isBoardFollower"] = util.Contains(boardObj.Followers, strconv.Itoa(ctx.Profile))

		// get cover and its thumbs
		fileName := fmt.Sprintf("%s.png", boardID)
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

		// original file
		key := util.GetKeyForBoardCover(int(ownerInfo.AccountID), int(ownerInfo.Id), boardID, "")
		file, err := a.storageService.GetUserFile(key, fileName)
		if err != nil {
			return errors.Wrap(err, "unable to fetch board cover")
		}

		// thumbs
		key = util.GetKeyForBoardCover(int(ownerInfo.AccountID), int(ownerInfo.Id), boardID, "thumbs")
		thumbs, err := helper.GetThumbnails(a.storageService, key, fileName, []string{"ic", "sm", "md", "lg"})
		if err != nil {
			return errors.Wrap(err, "unable to fetch board thumbnails")
		}
		thumbs.Original = file.Filename
		board["thumbs"] = thumbs

		response["data"].(map[string]interface{})["role"] = ownerBoardPermission[boardID]
		response["data"].(map[string]interface{})["isPassword"] = isPassword
		response["data"].(map[string]interface{})["isPasswordValidate"] = isPasswordValidate
		response["data"].(map[string]interface{})["isClientAccessible"] = isClientAccessible
		response["data"].(map[string]interface{})["board"] = board
		json.NewEncoder(w).Encode(response)
	}

	return err
}

func (a *api) UpdateBoard(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	boardID := ctx.Vars["boardID"]
	var err error
	if boardID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	var payload map[string]interface{}
	err = json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode payload json")
	}
	board, err := a.boardService.UpdateBoard(payload, boardID, ctx.Profile)
	if err != nil {
		return errors.Wrap(err, "unable to update board")
	}

	// update the search results
	boardMap := board["data"].(model.Board).ToMap()
	boardMap["_id"] = board["data"].(model.Board).Id
	// err = a.searchService.UpdateSearchResults(boardMap, "update")
	// if err != nil {
	// 	return errors.Wrap(err, "error from updating search result")
	// }
	reqVal, err := util.MapToMapAny(boardMap)
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

	// notify the board members if the board's title, is changed

	json.NewEncoder(w).Encode(board)

	return err
}

// DeleteBoard - delete a board if the current user is creator of it
func (a *api) DeleteBoard(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	boardId := ctx.Vars["boardID"]
	var err error
	if boardId == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}
	boardIdObj, _ := primitive.ObjectIDFromHex(boardId)
	boardRes, err := a.boardService.FetchBoardByID(boardId)
	board := boardRes["data"].(*model.Board)
	if err != nil {
		return err
	}
	var parentBoardIds []map[string]interface{}
	var pbErr error
	if board.ParentID != "" {
		parentBoardIds, pbErr = a.boardService.GetParentBoards(boardIdObj)
		if pbErr != nil {
			return pbErr
		}
	}
	res, err := a.boardService.DeleteBoard(boardId, ctx.Profile)
	if err == nil {
		var derr error
		if board.ParentID != "" {
			derr = permissions.DeleteBoardPermissions(a.cache, true, parentBoardIds, boardIdObj, ctx.Profile, *board)
		} else {
			derr = permissions.DeleteBoardPermissions(a.cache, false, nil, boardIdObj, ctx.Profile, *board)
		}
		if derr == nil {

			_, err = a.thingService.FlagBookmarkForDelete(ctx.Profile, boardId, time.Now())
			if err != nil {
				fmt.Println(err.Error())
				return errors.Wrap(err, "unable to update flag in bookmark")
			}

			json.NewEncoder(w).Encode(res)
			return nil
		}
		return errors.Wrap(derr, "unable to delete values from redis")
	}
	if err != nil {
		return errors.Wrap(err, "unable to delete values from redis")
	}

	// update the search results
	boardMap := res["data"].(model.Board).ToMap()
	boardMap["_id"] = res["data"].(model.Board).Id
	// err = a.searchService.UpdateSearchResults(boardMap, "delete")
	// if err != nil {
	// 	return errors.Wrap(err, "error from updating search result")
	// }
	reqVal, err := util.MapToMapAny(boardMap)
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

	json.NewEncoder(w).Encode(res)
	return err
}

// BoardUnfollow - unfollow the board if the current user is not creator of it
func (a *api) BoardUnfollow(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	profileID := ctx.Profile
	if profileID == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile unauthorized to perform this action"))
		return nil
	}

	boardId := ctx.Vars["boardID"]
	if boardId == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}
	res, err := a.boardService.BoardUnfollow(boardId, profileID)
	if err != nil {
		return err
	}
	json.NewEncoder(w).Encode(res)
	return nil
}

func (a *api) AutoComplete(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	boardID := ctx.Vars["boardID"]
	if boardID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	profileID := ctx.Profile
	if profileID == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile unauthorized to perform this action"))
		return nil
	}

	return nil
}

func (a *api) BoardSettings(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	profileID := ctx.Profile
	if profileID == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile unauthorized to perform this action"))
		return nil
	}

	boardId := ctx.Vars["boardID"]
	if boardId == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	boardObjID, err := primitive.ObjectIDFromHex(boardId)
	if err != nil {
		return err
	}

	var payload map[string]interface{}
	err = json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode json")
	}

	// board.Password = payload["password"].(string)
	res, err := a.boardService.BoardSettings(boardId, profileID, payload)
	if err != nil {
		return err
	}
	err = a.profileService.UpdateProfileTagsNew(strconv.Itoa(ctx.Profile))
	if err != nil {
		return errors.Wrap(err, "unable to update tags")
	}
	// cp, err := a.profileService.FetchConciseProfile(ctx.Profile)
	// if err != nil {
	// 	return errors.Wrap(err, "unable to fetch basic info")
	// }

	cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(ctx.Profile)}
	cp, err := a.repos.PeopleGrpcServiceClient.GetConciseProfile(context.TODO(), cpreq)
	if err != nil {
		return err
	}

	msg := model.ThingActivity{}
	msg.Create(
		boardObjID,
		primitive.NilObjectID,
		ctx.Profile,
		strings.ToUpper(consts.Board),
		fmt.Sprintf("%s %s updated the settings", cp.FirstName, cp.LastName),
	)
	err = a.thingActivityService.PushThingActivityToSQS(msg.ToMap())
	if err != nil {
		return errors.Wrap(err, "unable to push to SQS")
	}

	json.NewEncoder(w).Encode(res)
	return nil
}

func (a *api) FetchMembersFromConnections(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	boardID := ctx.Vars["boardID"]
	if boardID == "" {
		w.WriteHeader(http.StatusBadRequest)
	}
	var err error

	profileID := ctx.Profile
	if profileID == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	res, err := a.boardService.FetchConnectionsMembers(profileID, boardID)
	if err == nil {
		json.NewEncoder(w).Encode(res)
	}

	return err
}

func (a *api) FetchBoardMembers(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	limit := r.URL.Query().Get("limit")
	page := r.URL.Query().Get("page")
	search := r.URL.Query().Get("search")
	role := r.URL.Query().Get("role")

	boardId := ctx.Vars["boardID"]
	if boardId == "" {
		w.WriteHeader(http.StatusBadRequest)
	}
	profileID := ctx.Profile
	if profileID == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	members, err := a.boardService.GetBoardMembers(boardId, limit, page, search, role)
	if err == nil {
		json.NewEncoder(w).Encode(members)
	}

	return err
}

func (a *api) InviteMembers(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	boardID := ctx.Vars["boardID"]
	if boardID == "" {
		w.WriteHeader(http.StatusBadRequest)
	}
	var err error
	var invites model.BoardMemberRequest
	err = json.NewDecoder(r.Body).Decode(&invites)
	if err != nil {
		return errors.Wrap(err, "unable to decode json")
	}

	profileID := ctx.Profile
	if profileID == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	res, err := a.boardService.InviteMembers(boardID, profileID, invites.Data)
	if err == nil {
		return json.NewEncoder(w).Encode(res)
	}
	return err
}

func (a *api) ListBoardInvitations(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	var err error
	profileID := ctx.Profile
	if profileID == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	res, err := a.boardService.ListBoardInvites(profileID)
	if err == nil {
		json.NewEncoder(w).Encode(res)
	}

	return err
}

func (a *api) HandleBoardInvitation(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	var err error
	var hbi model.HandleBoardInvitation
	err = json.NewDecoder(r.Body).Decode(&hbi)
	if err != nil {
		return errors.Wrap(err, "unable to decode json")
	}

	profileID := ctx.Profile
	if profileID == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	handleres, err := a.boardService.HandleBoardInvitation(profileID, hbi)
	if err != nil {
		return errors.Wrap(err, "unable to handle board invitation")
	}

	key := fmt.Sprintf("boards:%s", strconv.Itoa(ctx.Profile))
	boardPerms := make(model.BoardPermission)
	cacheBoards, err := a.cache.GetValue(key)
	if err != nil {
		return errors.Wrap(err, "unable to get cache boards")
	}
	err = json.Unmarshal([]byte(cacheBoards), &boardPerms)
	if err != nil {
		return errors.Wrap(err, "unable to unmarshal into board permission")
	}

	cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(ctx.Profile)}
	cp, err := a.repos.PeopleGrpcServiceClient.GetConciseProfile(context.TODO(), cpreq)
	if err != nil {
		return err
	}

	// get board members
	res, err := a.boardService.GetBoardMembers2(hbi.BoardID, "", "", "", "")
	if err != nil {
		return errors.Wrap(err, "unable to find board members")
	}
	members := res["data"].([]string)
	membersInt := []int32{}
	for _, m := range members {
		v, err := strconv.Atoi(m)
		if err != nil {
			return errors.Wrap(err, "unable to convert string to int")
		}
		membersInt = append(membersInt, int32(v))
	}

	if handleres["message"].(string) == "Invitation accepted!" &&
		(boardPerms[hbi.BoardID] == consts.Admin || boardPerms[hbi.BoardID] == consts.Guest ||
			boardPerms[hbi.BoardID] == consts.Subscriber || boardPerms[hbi.BoardID] == consts.Author) {

		// send notification
		request := notfrpc.NotificationHandlerRequest{
			ReceiverIDs: membersInt,
			SenderID:    int32(profileID),
			ThingType:   consts.BoardType,
			ActionType:  consts.BoardMemberAdded,
			Message:     fmt.Sprintf("%s %s has been added to the board.", cp.FirstName, cp.LastName),
			ThingID:     hbi.BoardID,
		}
		_, err = a.repos.NotificationGrpcServiceClient.NotificationHandler(context.TODO(), &request)
		if err != nil {
			return errors.Wrap(err, "unable send and create notification")
		}
	}

	// notify the board members that a new member is added

	// receiver's ID
	// cp, err := a.profileService.FetchConciseProfile(ctx.Profile)
	// if err != nil {
	// 	return errors.Wrap(err, "unable to fetch concise Profile")
	// }

	// sender's ID
	// cp2, err := a.profileService.FetchConciseProfile(res["data"].(int))
	// if err != nil {
	// 	return errors.Wrap(err, "unable to fetch concise Profile")
	// }

	cpreq2 := &peoplerpc.ConciseProfileRequest{ProfileId: int32(handleres["data"].(int))}
	cp2, err := a.repos.PeopleGrpcServiceClient.GetConciseProfile(context.TODO(), cpreq2)
	if err != nil {
		return err
	}

	boardObjID, err := primitive.ObjectIDFromHex(hbi.BoardID)
	if err != nil {
		return errors.Wrap(err, "unable to convert string to ObjectID")
	}

	msg := model.ThingActivity{}
	msg.Create(
		boardObjID,
		primitive.NilObjectID,
		ctx.Profile,
		strings.ToUpper(consts.Board),
		fmt.Sprintf("%s %s added %s %s to the board", cp2.FirstName, cp2.LastName, cp.FirstName, cp.LastName),
	)
	err = a.thingActivityService.PushThingActivityToSQS(msg.ToMap())
	if err != nil {
		return errors.Wrap(err, "unable to push to SQS")
	}

	res["data"] = nil
	json.NewEncoder(w).Encode(handleres)

	return err
}

func (a *api) ChangeProfileRole(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	boardID := ctx.Vars["boardID"]
	if boardID == "" {
		w.WriteHeader(http.StatusBadRequest)
	}
	var err error
	var cbp model.ChangeProfileRole
	err = json.NewDecoder(r.Body).Decode(&cbp)
	if err != nil {
		return errors.Wrap(err, "unable to decode json")
	}

	profileID := ctx.Profile
	if profileID == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	res, err := a.boardService.ChangeProfileRole(profileID, boardID, cbp)
	if err == nil {
		json.NewEncoder(w).Encode(res)
	}

	return err
}

func (a *api) BlockMembers(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	boardID := ctx.Vars["boardID"]
	if boardID == "" {
		w.WriteHeader(http.StatusBadRequest)
	}
	var err error
	var bmp model.BoardMemberRequest
	err = json.NewDecoder(r.Body).Decode(&bmp)
	if err != nil {
		return err
	}

	profileID := ctx.Profile
	if profileID == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	res, err := a.boardService.BlockMembers(profileID, boardID, bmp.Data)
	if err == nil {
		json.NewEncoder(w).Encode(res)
	}

	return err
}

func (a *api) UnblockMembers(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	boardID := ctx.Vars["boardID"]
	if boardID == "" {
		w.WriteHeader(http.StatusBadRequest)
	}
	var err error
	var bmp model.BoardMemberRequest
	err = json.NewDecoder(r.Body).Decode(&bmp)
	if err != nil {
		return err
	}

	profileID := ctx.Profile
	if profileID == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}
	ret, err := a.boardService.UnblockMembers(profileID, boardID, bmp.Data)
	if err == nil {
		json.NewEncoder(w).Encode(ret)
	}

	return nil
}

func (a *api) ListBlockedMembers(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	boardID := ctx.Vars["boardID"]
	if boardID == "" {
		w.WriteHeader(http.StatusBadRequest)
	}

	query := r.URL.Query()
	limit := query.Get("limit")
	page := query.Get("page")
	search := query.Get("search")

	profileID := ctx.Profile
	if profileID == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	ret, err := a.boardService.ListBlockedMembers(profileID, page, limit, boardID, search)
	if err == nil {
		json.NewEncoder(w).Encode(ret)
	}
	return err
}

// FollowBoard - allow user to follow
func (a *api) BoardFollow(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	if ctx.Profile == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile unauthorized to perform this action"))
		return nil
	}
	boardId := ctx.Vars["boardID"]
	if boardId == "" {
		w.WriteHeader(http.StatusBadRequest)
	}
	var payload model.BoardFollowInfo
	payload.BoardID = boardId
	payload.ProfileID = ctx.Profile
	res, err := a.boardService.BoardFollow(payload)
	if err != nil {
		return errors.Wrap(err, "unable to complete BoardFollow")
	}

	boardData, err := a.boardService.FetchBoardDetailsByID(boardId)
	if err != nil {
		return errors.Wrap(err, "Unable to get board details")
	}

	if boardData["data"] != nil && boardData["status"].(int) == 1 {
		board := boardData["data"].(*model.Board)
		receiverID, err := strconv.Atoi(string(board.Owner))
		if err != nil {
			return errors.Wrap(err, "unable to parse board.Owner")
		}

		senderId, err := strconv.Atoi(fmt.Sprint(ctx.Profile))
		if err != nil {
			return errors.Wrap(err, "unable to parse profileID")
		}

		fmt.Println(receiverID, senderId)

		// err = a.notificationService.NotificationHandler(receiverID, senderId, a.clientMgr, consts.BoardType, boardId, consts.BoardFollowed, "")
		// if err != nil {
		// 	return errors.Wrap(err, "unable send and create notification")
		// }

		// CALL GRPC
		request := notfrpc.NotificationHandlerRequest{
			ReceiverIDs: []int32{int32(receiverID)},
			SenderID:    int32(senderId),
			ThingType:   consts.BoardType,
			ActionType:  consts.BoardFollowed,
			Message:     "",
			ThingID:     boardId,
		}
		_, err = a.repos.NotificationGrpcServiceClient.NotificationHandler(context.TODO(), &request)
		if err != nil {
			return errors.Wrap(err, "unable send and create notification")
		}
	}

	json.NewEncoder(w).Encode(res)
	return nil
}

func (a *api) RemoveMembers(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	boardID := ctx.Vars["boardID"]
	if boardID == "" {
		w.WriteHeader(http.StatusBadRequest)
	}
	var err error
	var bmp model.BoardMemberRequest
	err = json.NewDecoder(r.Body).Decode(&bmp)
	if err != nil {
		return err
	}

	if ctx.Profile == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	res, err := a.boardService.RemoveMembers(ctx.Profile, boardID, bmp.Data)
	if err != nil {
		return errors.Wrap(err, "unable to remove members")
	}
	// if res["status"].(int) != 0 {
	// add to activity
	// boardObjID, err := primitive.ObjectIDFromHex(boardID)
	// if err != nil {
	// 	return errors.Wrap(err, "unable to convert string to ObjectID")
	// }
	// cp, err := a.profileService.FetchConciseProfile(ctx.Profile, a.storageService)
	// if err != nil {
	// 	return errors.Wrap(err, "unable to fetch concise Profile")
	// }

	// msg := model.ThingActivity{}
	// msg.Id = primitive.NewObjectID()
	// msg.BoardID = boardObjID
	// msg.ThingType = "BOARD"
	// msg.ProfileID = ctx.Profile
	// msg.Name = fmt.Sprintf("%s %s", cp.FirstName, cp.LastName)
	// msg.Message = fmt.Sprintf(" removed %s from the board", strings.Join(res["data"].([]string), ", "))
	// msg.LastModifiedDate = time.Now()
	// msg.DateModified = msg.LastModifiedDate.Format("01-02-2006 15:04:05")
	// err = a.thingActivityService.PushThingActivityToSQS(msg.ToMap())
	// if err != nil {
	// 	return errors.Wrap(err, "unable to push activity to SQS")
	// }
	// }

	res["data"] = nil
	json.NewEncoder(w).Encode(res)

	return err
}

func (a *api) AddViewerInBoard(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	boardID := ctx.Vars["boardID"]
	if boardID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "need boardID in path param"))
		return nil
	}

	if ctx.Profile == -1 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}
	profileIDStr := strconv.Itoa(ctx.Profile)
	board, err := a.boardService.FetchBoardDetailsByID(boardID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(err)
		return nil
	}

	boardData := board["data"].(*model.Board)
	if boardData.Owner == profileIDStr {
		fmt.Println("Hello")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(util.SetResponse(nil, 1, "Profile is an owner"))
		return nil
	}
	res, err := a.boardService.AddViewerInBoardByID(boardID, profileIDStr)
	if err == nil {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(res)
		return nil
	}
	return err
}

func (a *api) FetchBoardThingsByProfile(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	boardID := ctx.Vars["boardID"]
	if boardID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}
	var err error
	response := make(map[string]interface{})
	response["data"] = make(map[string]interface{})
	var boardObj *model.Board
	errChan := make(chan error)
	var subBoardRes map[string]interface{}
	var taskRes map[string]interface{}
	var noteRes map[string]interface{}
	var fileRes map[string]interface{}

	profileID := ctx.Profile
	if profileID == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}
	publicOnly, _ := strconv.ParseBool(r.URL.Query().Get("publicOnly"))
	limit := 10

	profileKey := fmt.Sprintf("boards:%s", strconv.Itoa(ctx.Profile))
	bres, err := a.boardService.FetchBoardByID(boardID, strconv.Itoa(profileID))
	boardObj = bres["data"].(*model.Board)

	ownerBoardPermission := permissions.GetBoardPermissionsNew(profileKey, a.cache, boardObj, strconv.Itoa(profileID))
	response["data"].(map[string]interface{})["isBoardFollower"] = util.Contains(boardObj.Followers, strconv.Itoa(profileID))
	if ownerBoardPermission[boardID] == "" {
		ownerBoardPermission[boardID] = "viewer"
	}
	response["data"].(map[string]interface{})["role"] = ownerBoardPermission[boardID]
	// fetch board-info
	go func(errChan chan<- error) {
		defer util.RecoverGoroutinePanic(errChan)
		log.Printf("started --------> board-info")
		if bres["data"] != nil {
			response["data"].(map[string]interface{})["board"] = bres["data"]
		} else {
			response["data"].(map[string]interface{})["board"] = nil
		}
		log.Printf("done --------> board-info")
		errChan <- nil
	}(errChan)
	// sub-boards
	go func(errChan chan<- error) {
		defer util.RecoverGoroutinePanic(errChan)
		log.Printf("started --------> sub-boards")
		subBoardRes, err = a.boardService.FetchSubBoardsByProfile(boardID, profileID, limit, publicOnly)
		if err != nil {
			errChan <- errors.Wrap(err, "error in fetching sub-boards of the board ")
		}
		if subBoardRes["data"] != nil {
			response["data"].(map[string]interface{})["subBoards"] = subBoardRes["data"]
			// }
		} else {
			response["data"].(map[string]interface{})["subBoards"] = nil
		}
		log.Printf("done --------> sub-boards")
		errChan <- nil
	}(errChan)
	// tasks
	go func(errChan chan<- error) {
		defer util.RecoverGoroutinePanic(errChan)
		log.Printf("started --------> tasks")
		taskRes, err = a.taskService.FetchTasksByProfile(boardID, profileID, limit, publicOnly)
		if err != nil {
			errChan <- errors.Wrap(err, "error in fetching board tasks ")
		}
		if taskRes["data"] != nil {
			response["data"].(map[string]interface{})["tasks"] = taskRes["data"]
		} else {
			response["data"].(map[string]interface{})["tasks"] = nil
		}
		log.Printf("done --------> tasks")
		errChan <- nil
	}(errChan)
	// notes
	go func(errChan chan<- error) {
		defer util.RecoverGoroutinePanic(errChan)
		log.Printf("started --------> notes")
		noteRes, err = a.noteService.FetchNotesByProfile(boardID, profileID, limit, publicOnly)
		if err != nil {
			errChan <- errors.Wrap(err, "error in fetching board notes")
		}
		if noteRes["data"] != nil {
			response["data"].(map[string]interface{})["notes"] = noteRes["data"]
			// }
		} else {
			response["data"].(map[string]interface{})["notes"] = nil
		}
		log.Printf("done --------> notes")
		errChan <- nil
	}(errChan)

	// files
	go func(errChan chan<- error) {
		defer util.RecoverGoroutinePanic(errChan)
		log.Printf("started --------> files")
		fileRes, err = a.fileService.FetchFilesByProfile(boardID, ctx.Profile, limit, publicOnly)
		if err != nil {
			errChan <- errors.Wrap(err, "error in fetching board files ")
		}
		if fileRes["data"] != nil {
			response["data"].(map[string]interface{})["files"] = fileRes["data"]
			// }
		} else {
			response["data"].(map[string]interface{})["files"] = nil
		}
		log.Printf("done --------> files")
		errChan <- nil
	}(errChan)

	totalGoroutines := 6
	// totalGoroutines := 2
	if publicOnly {
		totalGoroutines = 5
	}
	for i := 0; i < totalGoroutines; i++ {
		if err := <-errChan; err != nil {
			return errors.Wrap(err, "error from go routine")
		}
	}
	response["data"].(map[string]interface{})["role"] = ownerBoardPermission[boardID]
	response["status"] = 1
	response["message"] = "Board things fetched successfully."
	fmt.Println("Waiting started . . .")
	json.NewEncoder(w).Encode(response)
	fmt.Println("Data: ", response)
	return nil
}

func (a *api) BoardAuth(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	boardId := ctx.Vars["boardID"]
	if boardId == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}
	boardIdObj, err := primitive.ObjectIDFromHex(boardId)
	if err != nil {
		return err
	}
	var payload map[string]string
	err = json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return err
	}
	res, err := a.boardService.BoardAuth(boardIdObj, payload["password"])
	if err == nil {
		json.NewEncoder(w).Encode(res)
		return nil
	}
	return err
}

func (a *api) FetchSubBoardsOfProfile(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	profileID := ctx.Profile
	if profileID == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}
	page := r.URL.Query().Get("page")
	limit := r.URL.Query().Get("limit")
	res, err := a.boardService.FetchSubBoardsOfProfile(profileID, page, limit)
	if err == nil {
		json.NewEncoder(w).Encode(res)
	}
	return nil
}

func (a *api) FetchSubBoardsOfBoard(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	var err error
	if ctx.Profile == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}
	page := r.URL.Query().Get("page")
	limit := r.URL.Query().Get("limit")
	boardID := ctx.Vars["boardID"]
	if boardID == "" {
		w.WriteHeader(http.StatusBadRequest)
	}
	res, err := a.boardService.FetchSubBoardsOfBoard(ctx.Profile, boardID, page, limit)
	if err == nil {
		json.NewEncoder(w).Encode(res)
	}
	return err
}

func (a *api) GetBoardFollowers(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	limit := r.URL.Query().Get("limit")
	page := r.URL.Query().Get("page")
	search := r.URL.Query().Get("search")

	boardID := ctx.Vars["boardID"]
	if boardID == "" {
		w.WriteHeader(http.StatusBadRequest)
	}

	if ctx.Profile == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	res, err := a.boardService.GetBoardFollowers(boardID, search, page, limit)
	if err == nil {
		json.NewEncoder(w).Encode(res)
		return nil
	}

	return err
}

func (a *api) GetSharedBoards(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	if ctx.Profile == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	limit := r.URL.Query().Get("limit")
	page := r.URL.Query().Get("page")
	search := r.URL.Query().Get("search")

	res, err := a.boardService.GetSharedBoards(ctx.Profile, search, page, limit, "", "")
	if err == nil {
		json.NewEncoder(w).Encode(res)
	}

	return err
}

func (a *api) FetchBoardActivities(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	boardID := ctx.Vars["boardID"]
	if boardID == "" {
		w.WriteHeader(http.StatusBadRequest)
	}

	// res, err := a.boardService.GetBoardActivities(boardID)
	return nil
}
