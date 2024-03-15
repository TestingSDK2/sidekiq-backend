package post

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/consts"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/helper"
	peoplerpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"
	searchrpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-search/v1"
	"github.com/pkg/errors"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/util"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-models/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (a *api) FetchPostsOfBoard2(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	if ctx.Profile == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	boardID := ctx.Vars["boardID"]
	if boardID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	limit, page := r.URL.Query().Get("limit"), r.URL.Query().Get("page")
	sortBy, filterBy := r.URL.Query().Get("sortBy"), r.URL.Query().Get("filterBy")

	cps := make(map[int]*peoplerpc.ConciseProfileReply)

	res, err := a.postService.GetPostsOfBoard(ctx.Profile, boardID, limit, page, filterBy, sortBy)
	if err != nil {
		return errors.Wrap(err, "unable to fetch posts of board")
	}

	goRoutines := 0
	mutex := sync.Mutex{}
	var postsMap []map[string]interface{}

	posts := res["data"].(map[string]interface{})["info"].([]model.Post)
	for i := 0; i < len(posts); i++ {
		postsMap = append(postsMap, posts[i].ToMap())
	}

	errChan := make(chan error, len(postsMap))
	ch := make(chan bool, 4)

	for i := 0; i < len(postsMap); i++ {
		ch <- true
		go func(i int, errChan chan<- error) {
			// get board info
			boardInfo, err := a.boardService.FetchBoardDetailsByID(boardID)
			if err != nil {
				errors.Wrap(err, "board not found")
			}
			postsMap[i]["boardInfo"] = boardInfo["data"]

			// get post things
			allThings, err := a.postService.GetPostThings(boardID, postsMap[i]["_id"].(string), ctx.Profile)
			if err != nil {
				errChan <- err
			}
			if len(allThings) != 0 {
				mutex.Lock()
				postsMap[i]["things"] = allThings[0]
				mutex.Unlock()
			}

			// get owner info (use map to improve performance)
			owner, err := strconv.Atoi(postsMap[i]["owner"].(string))
			if err != nil {
				errors.Wrap(err, "unable to convert string to int")
			}
			// check in map
			var cp *peoplerpc.ConciseProfileReply
			// check in redis
			// v, err := a.cache.GetValue(fmt.Sprintf("user:%d", ctx.User.ID))
			// if err != nil {
			// 	errChan <- errors.Wrap(err, "unable to fetch concise profile from redis")
			// }
			// err = json.Unmarshal([]byte(v), cp)
			// if err != nil {
			// 	errChan <- errors.Wrap(err, "unable to unmarshal string json")
			// }

			if val, ok := cps[owner]; !ok {
				// cp, err = a.profileService.FetchConciseProfile(owner)

				cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(owner)}
				cp, err := a.repos.PeopleGrpcServiceClient.GetConciseProfile(context.TODO(), cpreq)
				if err != nil {
					errChan <- errors.Wrap(err, "unable to fetch concise profile")
				}

				if err != nil {
					errChan <- errors.Wrap(err, "unable to fetch concise profile")
				}

				cps[owner] = cp
			} else {
				cp = val
			}
			mutex.Lock()
			postsMap[i]["ownerInfo"] = cp

			if postsMap[i]["likes"] != nil {
				postsMap[i]["totalLikes"] = len(postsMap[i]["likes"].([]interface{}))
				var likes []string
				for _, value := range postsMap[i]["likes"].([]interface{}) {
					likes = append(likes, value.(string))
				}
				if util.Contains(likes, fmt.Sprint(ctx.Profile)) {
					postsMap[i]["isLiked"] = true
				} else {
					postsMap[i]["isLiked"] = false
				}
			}

			if postsMap[i]["comments"] != nil {
				postsMap[i]["totalComments"] = len(postsMap[i]["comments"].([]interface{}))
			}

			mutex.Unlock()
			<-ch
			errChan <- nil
		}(i, errChan)
	}

	// waiting for the goroutines to finish
	for goRoutines != 0 {
		if err := <-errChan; err != nil {
			return errors.Wrap(err, "error from go routine")
		}
		goRoutines--
	}

	res["data"].(map[string]interface{})["info"] = postsMap
	json.NewEncoder(w).Encode(res)
	return nil
}

func (a *api) getCoverImageForPost(boardID, postID, fileExt string, boardownerInfo *peoplerpc.ConciseProfileReply) (model.Thumbnails, error) {
	key := util.GetKeyForPostCover(int(boardownerInfo.AccountID), int(boardownerInfo.Id), boardID, postID, "")
	fileName := fmt.Sprintf("%s%s", postID, fileExt)
	fileData, err := a.storageService.GetUserFile(key, fileName)
	if err != nil {
		return model.Thumbnails{}, err
	}

	thumbKey := util.GetKeyForPostCover(int(boardownerInfo.AccountID), int(boardownerInfo.Id), boardID, postID, "thumbs")
	thumbfileName := fmt.Sprintf("%s%s", postID, ".png")
	thumbs, err := helper.GetThumbnails(a.storageService, thumbKey, thumbfileName, []string{})
	if err != nil {
		thumbs = model.Thumbnails{}
	}

	thumbs.Original = fileData.Filename
	return thumbs, nil
}

func (a *api) FetchPostsOfBoard(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	if ctx.Profile == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	boardID := ctx.Vars["boardID"]
	if boardID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	limit, page := r.URL.Query().Get("limit"), r.URL.Query().Get("page")
	sortBy, filterBy := r.URL.Query().Get("sortBy"), r.URL.Query().Get("filterBy")

	boardInfo, err := a.boardService.FetchBoardDetailsByID(boardID)
	if err != nil {
		return errors.Wrap(err, "board not found")
	}

	res, err := a.postService.GetPostsOfBoard(ctx.Profile, boardID, limit, page, filterBy, sortBy)
	if err != nil {
		return errors.Wrap(err, "unable to fetch posts of board")
	}

	goRoutines := 0
	mutex := sync.Mutex{}

	postsMap := res["data"].(map[string]interface{})["info"].([]map[string]interface{})
	postownerInfo := make(map[string]*peoplerpc.ConciseProfileReply)

	var wg sync.WaitGroup
	errChan := make(chan error, len(postsMap))

	for i := 0; i < len(postsMap); i++ {
		goRoutines++
		wg.Add(1)
		go func(nwg *sync.WaitGroup, errChan chan<- error, npostsMap *map[string]interface{}, nboardInfo *map[string]interface{}) {
			defer nwg.Done()
			defer util.RecoverGoroutinePanic(nil)

			mutex.Lock()
			(*npostsMap)["boardInfo"] = (*nboardInfo)["data"]
			mutex.Unlock()

			things, ok := (*npostsMap)["things"]
			if ok && things != nil {
				ret, err := a.postService.GetThumbnailAndImageforPostThing((*npostsMap)["_id"].(primitive.ObjectID).Hex(), (*npostsMap)["boardID"].(primitive.ObjectID).Hex(), ctx.Profile, (*npostsMap)["things"].(map[string]interface{}))
				if err != nil {
					errChan <- errors.Wrap(err, "error from get first post thing")
					return
				}

				mutex.Lock()
				(*npostsMap)["things"] = ret
				mutex.Unlock()
			}

			mutex.Lock()
			_, ok = postownerInfo[(*npostsMap)["owner"].(string)]
			if !ok {
				// get owner info
				owner, err := strconv.Atoi((*npostsMap)["owner"].(string))
				if err != nil {
					errChan <- errors.Wrap(err, "unable to convert string to int")
					return
				}
				// cp, err := a.profileService.FetchConciseProfile(owner)
				// if err != nil {
				// 	errChan <- errors.Wrap(err, "unable to fetch concise profile")
				// 	return
				// }

				cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(owner)}
				cp, err := a.repos.PeopleGrpcServiceClient.GetConciseProfile(context.TODO(), cpreq)
				if err != nil {
					errChan <- errors.Wrap(err, "unable to fetch concise profile")
				}

				postownerInfo[(*npostsMap)["owner"].(string)] = cp
				(*npostsMap)["ownerInfo"] = cp
			} else {
				(*npostsMap)["ownerInfo"] = postownerInfo[(*npostsMap)["owner"].(string)]
			}
			mutex.Unlock()

			ok, isCoverImage := (*npostsMap)["isCoverImage"].(bool)
			if ok && isCoverImage {
				mutex.Lock()
				boardowner, err := strconv.Atoi((*nboardInfo)["data"].(*model.Board).Owner)
				if err != nil {
					errChan <- errors.Wrap(err, "unable to convert string to int")
					return
				}
				// boardownerInfo, err := a.profileService.FetchConciseProfile(boardowner)
				// if err != nil {
				// 	errChan <- errors.Wrap(err, "unable to fetch concise profile")
				// 	return
				// }

				cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(boardowner)}
				boardownerInfo, err := a.repos.PeopleGrpcServiceClient.GetConciseProfile(context.TODO(), cpreq)
				if err != nil {
					errChan <- errors.Wrap(err, "unable to fetch concise profile")
				}

				thumbs, err := a.getCoverImageForPost(boardID, (*npostsMap)["_id"].(primitive.ObjectID).Hex(), (*npostsMap)["fileExt"].(string), boardownerInfo)
				if err != nil {
					fmt.Println("errorr ------")
					errChan <- errors.Wrap(err, "unable to get cover image for post")
					return
				}
				(*npostsMap)["thumbs"] = thumbs
				(*npostsMap)["coverImageUrl"] = thumbs.Original
				(*npostsMap)["isCoverImage"] = true
				mutex.Unlock()
			} else {
				(*npostsMap)["thumbs"] = model.Thumbnails{}
			}

			isbookmarked, bid, err := a.thingService.IsBookMarkedByProfile((*npostsMap)["_id"].(primitive.ObjectID).Hex(), ctx.Profile)
			if err != nil {
				(*npostsMap)["bookmarkID"] = ""
				(*npostsMap)["isBookmarked"] = false
			} else {
				(*npostsMap)["bookmarkID"] = bid
				(*npostsMap)["isBookmarked"] = isbookmarked
			}

			if (*npostsMap)["likes"] != nil {
				(*npostsMap)["totalLikes"] = len((*npostsMap)["likes"].(primitive.A))
				var likes []string
				for _, value := range (*npostsMap)["likes"].(primitive.A) {
					likes = append(likes, value.(string))
				}
				if util.Contains(likes, fmt.Sprint(ctx.Profile)) {
					(*npostsMap)["isLiked"] = true
				} else {
					(*npostsMap)["isLiked"] = false
				}
			} else {
				(*npostsMap)["totalLikes"] = 0
			}

			if (*npostsMap)["comments"] != nil {
				(*npostsMap)["totalComments"] = len((*npostsMap)["comments"].(primitive.A))
			} else {
				(*npostsMap)["totalComments"] = 0
			}

			errChan <- nil
		}(&wg, errChan, &postsMap[i], &boardInfo)
	}

	// waiting for the goroutines to finish
	for goRoutines != 0 {
		if err := <-errChan; err != nil {
			return errors.Wrap(err, "error from go routine 250 ")
		}
		goRoutines--
	}

	wg.Wait()

	fmt.Println(strings.Repeat("*", 100))
	fmt.Println("for loop ended 254")

	res["data"].(map[string]interface{})["info"] = postsMap
	json.NewEncoder(w).Encode(res)
	return err
}

func (a *api) AddPost(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	if ctx.Profile == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	boardID := ctx.Vars["boardID"]
	if boardID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	var payload model.Post
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode post payload json")
	}

	// unpack the post payload
	res, err := a.postService.AddPost(ctx.Profile, boardID, payload)
	if err != nil {
		return err
	}

	// add to search results
	postMap := res["data"].(model.Post).ToMap()
	postMap["_id"] = res["data"].(model.Post).Id
	postMap["boardID"] = res["data"].(model.Post).BoardID

	// err = a.searchService.UpdateSearchResults(postMap, "insert")
	// if err != nil {
	// 	return err
	// }

	reqVal, err := util.MapToMapAny(postMap)
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

	recentThing := model.Recent{
		ThingID:      res["data"].(model.Post).Id,
		DisplayTitle: res["data"].(model.Post).Title,
		BoardID:      res["data"].(model.Post).Id,
		ProfileID:    strconv.Itoa(ctx.Profile),
		ThingType:    strings.ToUpper("POST"),
	}

	err = a.recentThingsService.AddToDashBoardRecent(recentThing)
	if err != nil {
		return errors.Wrap(err, "unable to add to recent thing")
	}

	// add to activity
	// cp, _ := a.profileService.FetchConciseProfile(ctx.Profile)

	cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(ctx.Profile)}
	cp, err := a.repos.PeopleGrpcServiceClient.GetConciseProfile(context.TODO(), cpreq)
	if err != nil {
		return errors.Wrap(err, "unable to fetch concise profile")
	}

	msg := model.ThingActivity{}
	postTitle := ""
	if res["data"].(model.Post).Title != "" {
		postTitle = res["data"].(model.Post).Title
	}
	msg.Create(
		res["data"].(model.Post).BoardID,
		res["data"].(model.Post).Id,
		ctx.Profile,
		strings.ToUpper(consts.Post),
		fmt.Sprintf("%s %s added the post <b>%s</b>", cp.FirstName, cp.LastName, postTitle),
	)
	err = a.thingActivityService.PushThingActivityToSQS(msg.ToMap())
	if err != nil {
		return errors.Wrap(err, "unable to push to SQS")
	}

	json.NewEncoder(w).Encode(res)
	return err
}

func (a *api) AddThingsOnPost(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	if ctx.Profile == -1 {
		res := util.SetResponse(nil, 0, "Profile not authorized")
		json.NewEncoder(w).Encode(res)
		return nil
	}
	errChan := make(chan error)
	goroutines := 0

	boardID, postID := ctx.Vars["boardID"], ctx.Vars["postID"]
	if boardID == "" || postID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	var payload model.PostThingsPayload
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode post things payload json")
	}

	postObjID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return errors.Wrap(err, "unable to convert string to objectID")
	}

	postRes, err := a.postService.FindPost(boardID, postID)
	if err != nil {
		return errors.Wrap(err, "unable to find post")
	}

	if payload.Notes != nil {
		goroutines += 1
		go func(errChan chan<- error) {
			defer util.RecoverGoroutinePanic(errChan)
			err = a.noteService.AddNotes(postObjID, payload.Notes)
			if err != nil {
				errChan <- errors.Wrap(err, "unable to add notes in post")
			}
			errChan <- nil
		}(errChan)
	}
	if payload.Tasks != nil {
		goroutines += 1
		go func(errChan chan<- error) {
			err = a.taskService.AddTasks(postObjID, payload.Tasks)
			if err != nil {
				errChan <- errors.Wrap(err, "unable to add tasks in post")
			}
			errChan <- nil
		}(errChan)
	}

	if len(payload.Collections) > 0 {
		for _, collectionobj := range payload.Collections {
			collectionobj.PostID = postObjID
			_, err = a.collectionService.AddCollection(collectionobj, ctx.Profile, boardID, postID)
			if err != nil {
				return errors.Wrap(err, "unable to add collection for post")
			}
		}
	}

	// waiting for goroutines to finish
	for goroutines != 0 {
		goroutines--
		if err := <-errChan; err != nil {
			return errors.Wrap(err, "error from go routine")
		}
	}

	// get post things
	allThings, err := a.postService.GetPostThings(boardID, postID, ctx.Profile)
	if err != nil {
		return err
	}

	postRes["data"].(model.Post).ToMap()["things"] = allThings
	postRes["message"] = "Things added successfully."
	json.NewEncoder(w).Encode(postRes)

	return nil
}

func (a *api) FetchPostThings(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	if ctx.Profile == -1 {
		res := util.SetResponse(nil, 0, "Profile not authorized")
		json.NewEncoder(w).Encode(res)
		return nil
	}

	boardID, postID := ctx.Vars["boardID"], ctx.Vars["postID"]
	if boardID == "" || postID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	// check board or post exists
	postRes, err := a.postService.FindPost(boardID, postID)
	if err != nil {
		return errors.Wrap(err, "unable to find post")
	}
	postData := postRes["data"].(model.Post)
	postMap := postRes["data"].(model.Post).ToMap()

	// get board title
	bRes, err := a.boardService.FetchBoardInfo(boardID)
	if err != nil {
		return err
	}

	role, err := a.boardService.GetBoardProfileRole(boardID, fmt.Sprint(ctx.Profile))
	if err != nil {
		return err
	}

	// get post things
	allThings, err := a.postService.GetPostThings(boardID, postID, ctx.Profile)
	if err != nil {
		return err
	}

	// get owner info
	owner, err := strconv.Atoi(postMap["owner"].(string))
	if err != nil {
		return errors.Wrap(err, "unable to convert string to int")
	}
	// cp, err := a.profileService.FetchConciseProfile(owner)
	// if err != nil {
	// 	return errors.Wrap(err, "unable to fetch concise profile")
	// }

	cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(owner)}
	cp, err := a.repos.PeopleGrpcServiceClient.GetConciseProfile(context.TODO(), cpreq)
	if err != nil {
		return err
	}

	fmt.Println("all things len: ", len(allThings))

	if postMap["likes"] != nil {
		postMap["totalLikes"] = len(postMap["likes"].([]interface{}))
		var likes []string
		for _, value := range postMap["likes"].([]interface{}) {
			likes = append(likes, value.(string))
		}
		if util.Contains(likes, fmt.Sprint(ctx.Profile)) {
			postMap["isLiked"] = true
		} else {
			postMap["isLiked"] = false
		}
	}

	if postMap["comments"] != nil {
		postMap["totalComments"] = len(postMap["comments"].([]interface{}))
	}
	postMap["boardTitle"] = bRes["title"].(string)
	postMap["things"] = allThings
	postMap["ownerInfo"] = cp
	postMap["boardRole"] = role

	isbookmarked, bid, err := a.thingService.IsBookMarkedByProfile(postMap["_id"].(string), ctx.Profile)
	if err != nil {
		postMap["bookmarkID"] = ""
		postMap["isBookmarked"] = false
	} else {
		postMap["bookmarkID"] = bid
		postMap["isBookmarked"] = isbookmarked
	}

	ok, isCoverImage := postMap["isCoverImage"].(bool)
	if ok && isCoverImage {
		boardowner, err := strconv.Atoi(bRes["owner"].(string))
		if err != nil {
			return errors.Wrap(err, "unable to convert string to int")
		}
		// boardownerInfo, err := a.profileService.FetchConciseProfile(boardowner)
		// if err != nil {
		// 	return errors.Wrap(err, "unable to fetch concise profile")
		// }

		cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(boardowner)}
		boardownerInfo, err := a.repos.PeopleGrpcServiceClient.GetConciseProfile(context.TODO(), cpreq)
		if err != nil {
			return err
		}

		thumbs, err := a.getCoverImageForPost(boardID, postMap["_id"].(string), postMap["fileExt"].(string), boardownerInfo)
		if err != nil {
			return errors.Wrap(err, "unable to get cover image for post")
		}
		postMap["thumbs"] = thumbs
		postMap["coverImageUrl"] = thumbs.Original
		postMap["isCoverImage"] = true
	}

	postRes["data"] = postMap

	recentThing := model.Recent{
		ThingID:      postData.Id,
		DisplayTitle: postData.Title,
		BoardID:      postData.BoardID,
		ProfileID:    strconv.Itoa(ctx.Profile),
		ThingType:    "POST",
		// Thing:        postMap,
	}

	err = a.recentThingsService.AddToDashBoardRecent(recentThing)
	if err != nil {
		return errors.Wrap(err, "unable to add to recent thing")
	}

	json.NewEncoder(w).Encode(postRes)

	return err
}

func (a *api) UpdateThings(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	if ctx.Profile == -1 {
		res := util.SetResponse(nil, 0, "Profile not authorized")
		json.NewEncoder(w).Encode(res)
		return nil
	}

	boardID, postID := ctx.Vars["boardID"], ctx.Vars["postID"]
	if boardID == "" || postID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	_, err := a.postService.FindPost(boardID, postID)
	if err != nil {
		return errors.Wrap(err, "unable to find post")
	}

	postObjId, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return errors.Wrap(err, "unable to convert object id")
	}

	var payload model.PostThingUpdate
	err = json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode update post things payload json")
	}

	err = a.postService.UpdatePostThing(payload.UpdateThing, postObjId, fmt.Sprint(ctx.Profile))
	if err != nil {
		return errors.Wrap(err, "unable to update things.")
	}

	json.NewEncoder(w).Encode(util.SetResponse(nil, 1, "Post things updated."))
	return nil

}

func (a *api) DeleteSelectedThings(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	if ctx.Profile == -1 {
		res := util.SetResponse(nil, 0, "Profile not authorized")
		json.NewEncoder(w).Encode(res)
		return nil
	}

	boardID, postID := ctx.Vars["boardID"], ctx.Vars["postID"]
	if boardID == "" || postID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	_, err := a.postService.FindPost(boardID, postID)
	if err != nil {
		return errors.Wrap(err, "unable to find post")
	}

	postObjId, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return errors.Wrap(err, "unable to convert object id")
	}

	var payload model.PostThingDelete
	err = json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode delete post things payload json")
	}

	err = a.postService.DeleteSelectedPostThing(payload.DeleteThing, postObjId)
	if err != nil {
		return errors.Wrap(err, "unable to delete things.")
	}

	json.NewEncoder(w).Encode(util.SetResponse(nil, 1, "Post things delete."))
	return nil
}

func (a *api) handleThingEvent(payload model.PostThingEvent, boardID, postID string, profileID int) error {
	eventPayload := model.PostThingEventMember{
		BoardID:   boardID,
		PostID:    postID,
		ThingID:   payload.ThingID,
		ThingType: payload.ThingType,
	}

	var (
		editByProfileInfo *peoplerpc.ConciseProfileReply
		eventType         string
	)

	if payload.EditBy != "" {
		editbyInt, err := strconv.Atoi(payload.EditBy)
		if err != nil {
			return errors.Wrap(err, "unable to convert int for editBy")
		}

		cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(editbyInt)}
		editByProfileInfo, err = a.repos.PeopleGrpcServiceClient.GetConciseProfile(context.TODO(), cpreq)
		if err != nil {
			return err
		}

		currentTime := time.Now()
		eventType = consts.PostThingBlocked
		payload.EditDate = &currentTime
	} else {
		eventType = consts.PostThingUnBlocked
		payload.EditDate = nil
	}

	eventPayload.EditByInfo = editByProfileInfo
	eventPayload.EventType = eventType
	eventPayload.EditDate = payload.EditDate

	eventByteData, err := json.Marshal(eventPayload)
	fmt.Println(eventByteData)
	if err != nil {
		return fmt.Errorf("unable to marshal event payload: %v", err)
	}

	boardData, err := a.boardService.FetchBoardDetailsByID(boardID)
	if err != nil {
		return fmt.Errorf("unable to get board details: %v", err)
	}

	if boardData["data"] != nil && boardData["status"].(int) == 1 {
		board := boardData["data"].(*model.Board)
		members := make(map[string]string)
		addMembers := func(memberList []string) {
			for _, member := range memberList {
				members[member] = member
			}
		}

		addMembers([]string{board.Owner})
		addMembers(board.Admins)
		addMembers(board.Authors)
		addMembers(board.Viewers)
		addMembers(board.Subscribers)
		delete(members, payload.EditBy)

		for memberKey := range members {
			memberKeyInt, err := strconv.Atoi(memberKey)
			if err != nil {
				return errors.Wrap(err, "unable to convert int for memberKey")
			}
			fmt.Println(memberKeyInt)

			// CALL GRPC
			// session := a.clientMgr.GetSession(memberKeyInt)
			// if session != nil {
			// 	session.Send(&notification.Message{
			// 		Type:    eventPayload.EventType,
			// 		Content: string(eventByteData),
			// 	})
			// }
		}

	}

	objID, err := primitive.ObjectIDFromHex(payload.ThingID)
	if err != nil {
		return err
	}

	reqpayload := make(map[string]interface{})
	reqpayload["editBy"] = payload.EditBy
	reqpayload["editDate"] = payload.EditDate
	switch strings.ToUpper(payload.ThingType) {
	case "COLLECTION":
		err = a.collectionService.UpdateCollectionById(objID, reqpayload)
		if err != nil {
			return err
		}
	case "TASK":
		_, err = a.taskService.UpdateTask(reqpayload, boardID, postID, objID.Hex(), profileID)
		if err != nil {
			return err
		}
	case "NOTE":
		_, err = a.noteService.UpdateNote(reqpayload, boardID, postID, objID.Hex(), profileID)
		if err != nil {
			return err
		}
	case "FILE":
		err = a.storageService.UpdateFileById(objID, reqpayload)
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *api) SendThingEvent(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	if ctx.Profile == -1 {
		res := util.SetResponse(nil, 0, "Profile not authorized")
		json.NewEncoder(w).Encode(res)
		return nil
	}

	boardID, postID := ctx.Vars["boardID"], ctx.Vars["postID"]
	if boardID == "" || postID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	_, err := a.postService.FindPost(boardID, postID)
	if err != nil {
		return errors.Wrap(err, "unable to find post")
	}

	var payload model.PostThingEvent
	err = json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode post event payload json")
	}

	if payload.ThingID == "" {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "thingID can not be empty in request payload."))
		return nil
	}

	if payload.ThingType == "" {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "thingType can not be empty in request payload."))
		return nil
	}

	if strings.ToUpper(payload.ThingType) != "TASK" && strings.ToUpper(payload.ThingType) != "NOTE" && strings.ToUpper(payload.ThingType) != "FILE" && strings.ToUpper(payload.ThingType) != "COLLECTION" {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "thingType possible value can be TASK, NOTE, FILE or COLLECTION"))
		return nil
	}

	err = a.handleThingEvent(payload, boardID, postID, ctx.Profile)
	if err != nil {
		return errors.Wrap(err, "Error from handleThingEvent")
	}

	message := ""
	if payload.EditBy != "" {
		message = "Post thing blocked event send."
	} else {
		message = "Post thing unblocked event send."
	}
	json.NewEncoder(w).Encode(util.SetResponse(nil, 1, message))
	return nil
}

func (a *api) UpdatePostThingsUnblocked(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	if ctx.Profile == -1 {
		res := util.SetResponse(nil, 0, "Profile not authorized")
		json.NewEncoder(w).Encode(res)
		return nil
	}

	boardID, postID := ctx.Vars["boardID"], ctx.Vars["postID"]
	if boardID == "" || postID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	_, err := a.postService.FindPost(boardID, postID)
	if err != nil {
		return errors.Wrap(err, "unable to find post")
	}

	postObjId, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return errors.Wrap(err, "unable to convert objectId")
	}

	err = a.postService.UpdatePostThingUnblocked(postObjId, fmt.Sprint(ctx.Profile))
	if err != nil {
		return errors.Wrap(err, "unable to find post")
	}

	json.NewEncoder(w).Encode(util.SetResponse(nil, 1, "Post thing unblocked."))
	return nil
}

func (a *api) DeletePost(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	if ctx.Profile == -1 {
		res := util.SetResponse(nil, 0, "Profile not authorized")
		json.NewEncoder(w).Encode(res)
		return nil
	}

	boardID, postID := ctx.Vars["boardID"], ctx.Vars["postID"]
	if boardID == "" || postID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	_, err := a.postService.FindPost(boardID, postID)
	if err != nil {
		return errors.Wrap(err, "unable to find post")
	}
	errChan := make(chan error)
	goroutines := 0

	// delete the post
	err = a.postService.DeletePost(postID)
	if err != nil {
		return errors.Wrap(err, "unable to delete post")
	} else {

		allthings, err := a.postService.GetPostThings(boardID, postID, ctx.Profile)
		if err == nil {
			if len(allthings) > 0 {
				for _, thing := range allthings {
					_, ok := thing["_id"]
					if ok {
						_, err = a.thingService.FlagBookmarkForDelete(ctx.Profile, thing["_id"].(string), time.Now())
						if err != nil {
							fmt.Println(err.Error())
							return errors.Wrap(err, "unable to update flag in bookmark")
						}
					}
				}
			}
		}

		// delete notes based of post
		goroutines += 1
		go func(errChan chan<- error) {
			err = a.noteService.DeleteNotesOnPost(postID)
			if err != nil {
				errChan <- errors.Wrap(err, "unable to delete notes on post")
			}
			errChan <- nil
		}(errChan)

		// delete notes based of post
		goroutines += 1
		go func(errChan chan<- error) {
			err = a.taskService.DeleteTasksOnPost(postID)
			if err != nil {
				errChan <- errors.Wrap(err, "unable to delete tasks on post")
			}
			errChan <- nil
		}(errChan)

		// waiting for goroutines to finish
		for goroutines != 0 {
			goroutines--
			if err := <-errChan; err != nil {
				return errors.Wrap(err, "error from go routine")
			}
		}
	}

	_, err = a.thingService.FlagBookmarkForDelete(ctx.Profile, postID, time.Now())
	if err != nil {
		fmt.Println(err.Error())
		return errors.Wrap(err, "unable to update flag in bookmark")
	}

	json.NewEncoder(w).Encode(util.SetResponse(nil, 1, "Post deleted successfully"))
	return err
}

func (a *api) MovePost(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	if ctx.Profile == -1 {
		res := util.SetResponse(nil, 0, "Profile not authorized")
		json.NewEncoder(w).Encode(res)
		return nil
	}

	boardID, postID := ctx.Vars["boardID"], ctx.Vars["postID"]
	if boardID == "" || postID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	var payload map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode post payload json")
	}

	if payload["boardID"].(string) == boardID {
		return json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Post can not be moved in same board."))
	}

	// find post
	postRes, err := a.postService.FindPost(boardID, postID)
	if err != nil {
		return errors.Wrap(err, "board or post not found")
	}
	post := postRes["data"].(model.Post)

	err = a.postService.MovePost(post, postID, payload["boardID"].(string))
	if err != nil {
		return errors.Wrap(err, "unable to move post")
	}

	// handle other changes such as in searching?
	json.NewEncoder(w).Encode(util.SetResponse(nil, 1, "Post moved successfully"))

	return err
}

func (a *api) UpdatePostSettings(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	if ctx.Profile == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	boardID, postID := ctx.Vars["boardID"], ctx.Vars["postID"]
	if boardID == "" || postID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	var payload map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode post payload json")
	}

	// find post
	postRes, err := a.postService.FindPost(boardID, postID)
	if err != nil {
		return errors.Wrap(err, "board or post not found")
	}
	post := postRes["data"].(model.Post)

	res, err := a.postService.UpdatePostSettings(ctx.Profile, postID, post, payload)
	if err != nil {
		return err
	}

	// add to search results
	postMap := res["data"].(model.Post).ToMap()
	postMap["_id"] = res["data"].(model.Post).Id
	postMap["boardID"] = res["data"].(model.Post).BoardID

	// err = a.searchService.UpdateSearchResults(postMap, "update")
	// if err != nil {
	// 	return err
	// }

	reqVal, err := util.MapToMapAny(postMap)
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
	// cp, _ := a.profileService.FetchConciseProfile(ctx.Profile)

	cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(ctx.Profile)}
	cp, err := a.repos.PeopleGrpcServiceClient.GetConciseProfile(context.TODO(), cpreq)
	if err != nil {
		return err
	}

	msg := model.ThingActivity{}
	msg.Create(
		postRes["data"].(model.Post).BoardID,
		postRes["data"].(model.Post).Id,
		ctx.Profile,
		strings.ToUpper(consts.Post),
		fmt.Sprintf("%s %s updated the post settings", cp.FirstName, cp.LastName),
	)
	err = a.thingActivityService.PushThingActivityToSQS(msg.ToMap())
	if err != nil {
		return errors.Wrap(err, "unable to push to SQS")
	}

	res["data"] = nil
	json.NewEncoder(w).Encode(res)

	return err
}
