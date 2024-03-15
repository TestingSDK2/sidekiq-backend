package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/util"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-models/model"
	peoplerpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"
	"github.com/pkg/errors"

	// peoplerpc "github.com/sidekiq-people/proto/people"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (a *api) GetAllBoardPostMedia(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	boardID := ctx.Vars["boardID"]
	postID := ctx.Vars["postID"]
	page := r.URL.Query().Get("page")
	limit := r.URL.Query().Get("limit")

	if boardID == "" || postID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	boardInfo, err := a.boardService.FetchBoardInfo(boardID)
	if err != nil {
		return errors.Wrap(err, "unable to fetch board info")
	}

	// get the post
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
	cp, err := a.repos.PeopleGrpcServiceClient.GetConciseProfile(context.TODO(), cpreq)
	if err != nil {
		return err
	}

	fileRes, err := a.fileService.FetchFilesByPost(boardID, postID, cp, ctx.Profile, "", "", []string{}, "", 0, page, limit)
	if err == nil {
		json.NewEncoder(w).Encode(fileRes)
		return nil
	}
	return err
}

func (a *api) GetBoardPostMedia(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	boardID := ctx.Vars["boardID"]
	postID := ctx.Vars["postID"]
	mediaID := ctx.Vars["mediaID"]

	if boardID == "" || postID == "" || mediaID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	boardInfo, err := a.boardService.FetchBoardInfo(boardID)
	if err != nil {
		return errors.Wrap(err, "unable to fetch board info")
	}

	// get the post
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

	fileRes, err := a.fileService.FetchFileByMediaIDforPost(boardID, postID, mediaID, ownerInfo, ctx.Profile)
	if err == nil {
		json.NewEncoder(w).Encode(fileRes)
		return nil
	}
	return err
}

func (a *api) GetUploadProgress(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	boardID := ctx.Vars["boardID"]
	postID := ctx.Vars["postID"]

	if boardID == "" || postID == "" {
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

	key := util.GetKeyForBoardPostMedia(int(ownerInfo.AccountID), int(ownerInfo.Id), boardID, postID, "")

	totalSize, err := a.storageService.GetTotalFileSize(ctx.Profile, key, ctx.Vars["id"])
	if err != nil {
		return err
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
	var currentSize, old, new float64
	for {
		if old == 100.00 {
			json.NewEncoder(w).Encode(util.SetResponse(nil, 1, "File uploaded successfully"))
			return nil
		}
		parts, err := a.storageService.GetFileParts(ctx.Profile, key, ctx.Vars["id"])
		if err != nil {
			return err
		}
		for i := range parts {
			currentSize = float64(parts[i].Size)
		}
		new = (currentSize / *totalSize) * 100
		if old != new {
			old = new
			fmt.Fprintf(w, "data: %v\n\n", fmt.Sprintf("%v", old))
			flusher.Flush()
		}
	}
}

func (a *api) GetMediaUploadStatus(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	boardID := ctx.Vars["boardID"]
	postID := ctx.Vars["postID"]

	if boardID == "" || postID == "" {
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
	key := util.GetKeyForBoardPostMedia(int(ownerInfo.AccountID), int(ownerInfo.Id), boardID, postID, "")
	parts, err := a.storageService.GetFileParts(ctx.Profile, key, ctx.Vars["id"])
	if err != nil {
		return err
	}
	fmt.Println(len(parts))
	json.NewEncoder(w).Encode(parts)
	return nil
}

// UploadFile - File Upload handler
func (a *api) UploadMedia(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	if ctx.Profile == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}
	// Parse our multipart form, 12 << 20 specifies a maximum upload of 12 MB files.
	// received here in parts of 6MB..
	r.ParseMultipartForm(12 << 20)
	mFile, handler, err := r.FormFile("data")
	if err != nil {
		return errors.Wrap(err, "Error Retrieving the File")
	}
	defer mFile.Close()

	thingID := ctx.Vars["boardID"]
	postID := ctx.Vars["postID"]
	// var res map[string]interface{}

	if thingID == "" || postID == "" {
		w.WriteHeader(http.StatusBadRequest)
	}

	boardInfo, err := a.boardService.FetchBoardInfo(thingID)
	if err != nil {
		return errors.Wrap(err, "unable to fetch board info")
	}

	// get the post
	postRes, err := a.postService.FindPost(thingID, postID)
	if err != nil {
		return errors.Wrap(err, "unable to fetch post info")
	}

	fmd := r.FormValue("fmd")
	var fileMetaData map[string]interface{}
	err = json.Unmarshal([]byte(fmd), &fileMetaData)
	if err != nil {
		return errors.Wrap(err, "Error Parsing File metadata")
	}

	if postRes["data"] == nil {
		return errors.Wrap(err, "Post record does not exits")
	}

	postdata := postRes["data"]
	postOwner := postdata.(model.Post).Owner

	meta := r.FormValue("meta")
	var fileUpload model.FileUpload
	err = json.Unmarshal([]byte(meta), &fileUpload)
	if err != nil {
		return errors.Wrap(err, "Error Parsing File metadata")
	}
	fileUpload.UserID = ctx.User.ID
	fileUpload.Profile = ctx.Profile

	f := &model.File{
		Name:   fileUpload.Name,
		Type:   fileUpload.Type,
		Size:   handler.Size,
		ETag:   r.Header.Get("ETag"),
		Reader: mFile,
	}
	// notify clients if we upload complete file
	fileMetaDataChan := make(chan map[string]interface{})
	complete := make(chan *model.File)
	var wg sync.WaitGroup
	wg.Add(1)
	flag := true
	go func(profile int) {
		defer util.RecoverGoroutinePanic(nil)
		defer wg.Done()
		fmt.Println("*************************START")
		fmt.Println("---------------------------------------------------started waiting 2")
		f := <-fileMetaDataChan
		fmt.Println("---------------------------------------------------finished waiting 2")
		if f != nil {
			fmt.Println("file meta data. larger than 6 MB: ")
			fmt.Println(f)
			// update search results
			filemap := f["data"].(model.UploadedFile).ToMap()
			filemap["_id"] = f["data"].(model.UploadedFile).Id
			filemap["boardID"] = postRes["data"].(model.Post).BoardID
			// CALL GRPC
			// err = a.searchService.UpdateSearchResults(filemap, "insert")
			// if err != nil {
			// 	logrus.Error(err, "error from updating search result")
			// }
			flag = false
			json.NewEncoder(w).Encode(f)
			fmt.Println("***********************")
		}
		fmt.Println("--------------------------------started waiting")
		resultFile := <-complete
		fmt.Println("--------------------------------finished waiting")
		if resultFile != nil {
			// session := a.clientMgr.GetSession(profile)
			// if session != nil {
			// 	session.Send(&notification.Message{
			// 		Type:    "file",
			// 		Content: resultFile.ToJSON(),
			// 	})
			// }
			// a.storageService.PushToClients(profile, resultFile)
		}
		fmt.Println("****************************DONE")
	}(ctx.Profile)

	var key string
	fileMetaData["_id"] = primitive.NewObjectID()
	fileMetaData["fileExt"] = filepath.Ext(handler.Filename)
	fileName := fmt.Sprintf("%s%s", fileMetaData["_id"].(primitive.ObjectID).Hex(), fileMetaData["fileExt"])
	postObjID, _ := primitive.ObjectIDFromHex(postID)
	boardObjID, _ := primitive.ObjectIDFromHex(thingID)
	fileMetaData["boardID"] = boardObjID
	fileMetaData["postID"] = postObjID
	fileMetaData["collectionID"] = primitive.NilObjectID

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

	key = util.GetKeyForBoardPostMedia(int(ownerInfo.AccountID), int(ownerInfo.Id), thingID, postObjID.Hex(), "")
	fmt.Println("key: ", key)
	var response map[string]interface{}
	fmt.Println("------------------------------------------------------storing file started")
	storedFile, err := a.storageService.UploadUserFile(postOwner, key, fileName, f, &fileUpload, fileMetaData, complete, fileMetaDataChan, false)
	if err != nil {
		fmt.Println("<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<err here")
		complete <- nil
		return err
	} else if fileUpload.TotalSize == storedFile.Size {
		fmt.Println("full upload completed...")
		complete <- storedFile

		// ********* STORE FILE METADATA IN MONGO *********
		fileMetaData["fileMime"] = mime.TypeByExtension(filepath.Ext(fileMetaData["fileExt"].(string)))
		fileMetaData["fileSize"] = util.ConvertBytesToHumanReadable(storedFile.Size)
		fileMetaData["fileName"] = storedFile.Name
		fileMetaData["storageLocation"] = storedFile.Filename

		url, _ := a.fileStore.GetPresignedDownloadURL(key, fileName)
		fileMetaData["url"] = url
		response, err = a.fileService.AddFile(thingID, postID, fileMetaData, ctx.Profile)
		if err != nil {
			return errors.Wrap(err, "unable to add file metadata at mongo")
		}

		// update profile tags
		// p := model.Profile{ID: ctx.Profile}
		// err = a.profileService.UpdateProfileTags(p, response["data"].(model.UploadedFile).Tags)
		tags := []string{}

		if response["data"].(map[string]interface{})["tags"] != nil {
			tags = response["data"].(map[string]interface{})["tags"].([]string)
		}

		err = a.boardService.UpdateBoardThingsTags(ctx.Profile, thingID, fileMetaData["_id"].(primitive.ObjectID).Hex(), tags)
		if err != nil {
			return errors.Wrap(err, "unable to update tags of a file")
		}
		err = a.profileService.UpdateProfileTagsNew(strconv.Itoa(ctx.Profile))
		if err != nil {
			return errors.Wrap(err, "unable to update tags")
		}

		// add to activity
		// cp, err := a.profileService.FetchConciseProfile(ctx.Profile, a.storageService)
		// if err != nil {
		// 	return errors.Wrap(err, "unable to fetch concise Profile")
		// }

		// msg := model.ThingActivity{}
		// msg.Id = primitive.NewObjectID()
		// msg.PostID = postObjID
		// msg.BoardID = response["data"].(model.Task).BoardID
		// msg.ThingID = response["data"].(model.Task).Id
		// msg.ThingType = "TASK"
		// msg.ProfileID = ctx.Profile
		// msg.Name = fmt.Sprintf("%s %s", cp.FirstName, cp.LastName)
		// msg.Message = " updated the Task"
		// msg.DateModified = msg.LastModifiedDate.Format("01-02-2006 15:04:05")
		// err = a.thingActivity.PushThingActivityToSQS(msg.ToMap())
		// if err != nil {
		// 	return errors.Wrap(err, "unable to push activity to SQS")
		// }

		util.PrettyPrint(381, "file meta data ", response)
		json.NewEncoder(w).Encode(response)
		return nil
	}
	wg.Wait()
	if flag {
		json.NewEncoder(w).Encode(storedFile)
	}
	return nil
}

func (a *api) DeleteMedia(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {

	profileID := ctx.Profile
	if profileID == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	mediaID := ctx.Vars["mediaID"]
	boardID := ctx.Vars["boardID"]
	postID := ctx.Vars["postID"]
	if mediaID == "" || boardID == "" || postID == "" {
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

	res, err := a.storageService.DeletePostMedia(ownerInfo, boardID, mediaID, postID)
	if err == nil {

		_, err = a.thingService.FlagBookmarkForDelete(ctx.Profile, mediaID, time.Now())
		if err != nil {
			fmt.Println(err.Error())
			return errors.Wrap(err, "unable to update flag in bookmark")
		}

		json.NewEncoder(w).Encode(res)
	}

	// update search results
	// CALL GRPC
	// err = a.searchService.UpdateSearchResults(nil, "delete", mediaID)
	// if err != nil {
	// 	return errors.Wrap(err, "error from updating search result")
	// }

	return err
}

// func (a *api) UploadBoardCover(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
// 	boardID := ctx.Vars["boardID"]
// 	if boardID == "" {
// 		w.WriteHeader(http.StatusBadRequest)
// 	}

// 	profileID := ctx.Profile
// 	if profileID == -1 {
// 		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
// 		return nil
// 	}

// 	r.ParseMultipartForm(12 << 20)
// 	mFile, handler, err := r.FormFile("data")
// 	if err != nil {
// 		return errors.Wrap(err, "Error Retrieving the File")
// 	}
// 	defer mFile.Close()

// 	boardInfo, err := a.boardService.FetchBoardInfo(boardID)
// 	if err != nil {
// 		return errors.Wrap(err, "unable to fetch board info")
// 	}
// 	boardOwnerInt, err := strconv.Atoi(boardInfo["owner"].(string))
// 	if err != nil {
// 		return err
// 	}
// 	ownerInfo, err := a.profileService.FetchConciseProfile(boardOwnerInt)
// 	if err != nil {
// 		return err
// 	}
// 	key := util.GetKeyForBoardCover(ownerInfo.UserID, ownerInfo.Id, boardID, "")
// 	fileName := fmt.Sprintf("%s.png", boardID)

// 	f := &model.File{
// 		Name:   fileName,
// 		Type:   "image/png",
// 		Size:   handler.Size,
// 		ETag:   r.Header.Get("ETag"),
// 		Reader: mFile,
// 	}

// 	_, err = a.storageService.UploadUserFile("", key, fileName, f, nil, nil, nil, nil, true)
// 	if err != nil {
// 		return errors.Wrap(err, "unable to upload board cover")
// 	}

// 	dat, err := a.storageService.GetUserFile(key, fileName)
// 	if err == nil {
// 		json.NewEncoder(w).Encode(util.SetResponse(dat.Filename, 1, "Board cover fetched sucessfully"))
// 	}

// 	return errors.Wrap(err, "unable to presign board cover")
// }

func (a *api) GetBoardCover(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	boardID := ctx.Vars["boardID"]
	if boardID == "" {
		w.WriteHeader(http.StatusBadRequest)
	}

	if ctx.Profile == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	fileName := fmt.Sprintf("%s.png", boardID)
	boardInfo, err := a.boardService.FetchBoardInfo(boardID)
	if err != nil {
		return errors.Wrap(err, "unable to fetch board info")
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

	key := util.GetKeyForBoardCover(int(ownerInfo.AccountID), int(ownerInfo.Id), boardID, "")
	file, err := a.storageService.GetUserFile(key, fileName)
	if err == nil {
		json.NewEncoder(w).Encode(file)
	}

	return err
}

func (a *api) DeleteBoardCover(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	boardID := ctx.Vars["boardID"]
	if boardID == "" {
		w.WriteHeader(http.StatusBadRequest)
	}

	if ctx.Profile == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
		return nil
	}

	res, err := a.storageService.DeleteBoardCover(boardID)
	if err == nil {
		json.NewEncoder(w).Encode(res)
	}

	return err
}

func (a *api) ComputeCloudStorage(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	var prefix map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&prefix)
	if err != nil {
		return errors.Wrap(err, "unable to decode payload json")
	}
	key := prefix["prefix"].(string)
	parts, err := a.storageService.ComputeCloudStorage(key)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(parts)
	return nil
}

func (a *api) RemoveMediaChunks(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	if ctx.Profile == -1 {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Profile not authorized"))
	}
	res, err := a.storageService.RemoveMediaChunks(ctx.Profile)
	if err != nil {
		return errors.Wrap(err, "unable to delete media chunks")
	}
	json.NewEncoder(w).Encode(res)
	return nil
}
