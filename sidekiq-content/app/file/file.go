package file

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/board"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/storage"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/cache"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/consts"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/database"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/helper"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/mongodatabase"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/permissions"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/util"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-models/model"
	"github.com/pkg/errors"

	// peoplerpc "github.com/sidekiq-people/proto/people"
	peoplerpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func getFilesByBoard(db *mongodatabase.DBConfig, mysql *database.Database, cache *cache.Cache,
	boardService board.Service, profileService peoplerpc.AccountServiceClient, storageService storage.Service, boardID string, profileID int, fileType, owner string, tagsArr []string, uploadDate string, limit int, page string, l string,
) (map[string]interface{}, error) {
	var files []*model.UploadedFile

	dbconn, err := db.New(consts.Board)
	if err != nil {
		return nil, err
	}
	boardCollection, boardClient := dbconn.Collection, dbconn.Client
	defer boardClient.Disconnect(context.TODO())

	profileStr := strconv.Itoa(profileID)
	isValid, err := permissions.CheckValidPermissions(profileStr, cache, boardCollection, boardID, []string{consts.Blocked}, true)
	if err != nil {
		return nil, err
	}
	if !isValid {
		return nil, errors.New("User does not have the access to the board!")
	}

	boardObjID, err := primitive.ObjectIDFromHex(boardID)
	if err != nil {
		return nil, err
	}

	var board *model.Board
	err = boardCollection.FindOne(context.TODO(), bson.M{"_id": boardObjID}).Decode(&board)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find board")
	}

	dbconn2, err := db.New(consts.File)
	if err != nil {
		return nil, errors.Wrap(err, "unable to establish connection with File collection.")
	}

	fileCollection, fileClient := dbconn2.Collection, dbconn2.Client
	defer fileClient.Disconnect(context.TODO())

	findFilesFilter := bson.M{}

	findFilesFilter = bson.M{"$and": bson.A{
		bson.M{"boardID": boardObjID},
		bson.M{"state": bson.M{"$ne": consts.Hidden}},
		bson.M{"collectionID": primitive.NilObjectID},
	}}
	if owner != "" {
		findFilesFilter["owner"] = owner
	}
	if len(tagsArr) != 0 {
		findFilesFilter["tags"] = bson.M{"$all": tagsArr}
	}
	if fileType != "" {
		findFilesFilter["fileExt"] = "." + fileType
	}
	if uploadDate != "" {
		copyDate := uploadDate
		custom := "2006-01-02T15:04:05Z"

		start := copyDate + "T00:00:00Z"
		dayStart, _ := time.Parse(custom, start)

		uploadDate = uploadDate + "T11:59:59Z"
		dayEnd, _ := time.Parse(custom, uploadDate)

		findFilesFilter["createDate"] = bson.M{"$gte": dayStart, "$lte": dayEnd}
	}

	// total count
	total, err := fileCollection.CountDocuments(context.TODO(), findFilesFilter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find total count")
	}

	var opts *options.FindOptions
	var curr *mongo.Cursor

	findOptions := options.Find()
	findOptions.SetSort(bson.M{"createDate": -1})

	if limit != 0 {
		opts = options.Find().SetLimit(int64(limit))
		curr, err = fileCollection.Find(context.TODO(), findFilesFilter, opts, findOptions)
	} else {
		// pagination
		if l != "" && page != "" {
			pgInt, _ := strconv.Atoi(page)
			limitInt, _ := strconv.Atoi(l)
			offset := (pgInt - 1) * limitInt
			findOptions.SetSkip(int64(offset))
			findOptions.SetLimit(int64(limitInt))
		}
		curr, err = fileCollection.Find(context.TODO(), findFilesFilter, findOptions)
	}

	if err != nil {
		return nil, errors.Wrap(err, "unable to find file")
	}
	err = curr.All(context.TODO(), &files)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch files")
	}
	if len(files) == 0 {
		if l != "" {
			return util.SetPaginationResponse(nil, 0, 1, "Board has no files. Please add one."), nil
		} else {
			return util.SetResponse(nil, 1, "Board has no files. Please add one."), nil
		}
	}

	for idx := range files {
		fileOwner, _ := strconv.Atoi(files[idx].Owner)
		// ownerInfo, err := profileService.FetchConciseProfile(fileOwner)
		// if err != nil {
		// 	return nil, errors.Wrap(err, "unable to fetch basic info")
		// }
		cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(fileOwner)}
		ownerInfo, err := profileService.GetConciseProfile(context.TODO(), cpreq)
		if err != nil {
			return nil, errors.Wrap(err, "unable to fetch basic info")
		}

		ownerInfo.Id = int32(fileOwner)
		files[idx].OwnerInfo = ownerInfo

		// presign beforehand
		if strings.Contains(files[idx].FileMime, "image/") || strings.Contains(files[idx].FileMime, "video/") {
			boardInfo, err := boardService.FetchBoardInfo(files[idx].BoardID.Hex())
			if err != nil {
				logrus.Error(err, "unable to fetch boardInfo")
				continue
			}
			boardOwnerInt, err := strconv.Atoi(boardInfo["owner"].(string))
			if err != nil {
				logrus.Error(err, "unable to convert to int")
				continue
			}
			// ownerInfo, err = profileService.FetchConciseProfile(boardOwnerInt)
			// if err != nil {
			// 	logrus.Error(err, "unable to fetch concise profile")
			// 	continue
			// }

			cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(boardOwnerInt)}
			ownerInfo, err := profileService.GetConciseProfile(context.TODO(), cpreq)
			if err != nil {
				logrus.Error(err, "unable to fetch concise profile")
				continue
			}

			key := util.GetKeyForBoardMedia(int(ownerInfo.AccountID), int(ownerInfo.Id), boardID, "")
			fileName := fmt.Sprintf("%s%s", files[idx].Id.Hex(), files[idx].FileExt)
			f, err := storageService.GetUserFile(key, fileName)
			if err != nil {
				return nil, errors.Wrap(err, "unable to presign image")
			}
			files[idx].URL = f.Filename
		}

		// Fetch Thumbnails
		thumbKey := util.GetKeyForBoardMedia(int(board.OwnerInfo.AccountID), int(board.OwnerInfo.Id), boardID, "thumbs")
		thumbfileName := files[idx].Id.Hex() + ".png"
		files[idx].Thumbs, err = helper.GetThumbnails(storageService, thumbKey, thumbfileName, []string{})
		if err != nil {
			files[idx].Thumbs = model.Thumbnails{}
		}

		// reaction count
		if util.Contains(files[idx].Likes, profileStr) {
			files[idx].IsLiked = true
		} else {
			files[idx].IsLiked = false
		}
		files[idx].TotalComments = len(files[idx].Comments)
		files[idx].TotalLikes = len(files[idx].Likes)

		// get location
		loc, err := boardService.GetThingLocationOnBoard(files[idx].BoardID.Hex())
		if err != nil {
			continue
		}
		files[idx].Location = loc
	}

	return util.SetPaginationResponse(files, int(total), 1, "Files fetched successfully."), nil
}

func getFilesByPost(db *mongodatabase.DBConfig, mysql *database.Database, cache *cache.Cache,
	boardService board.Service, profileService peoplerpc.AccountServiceClient, storageService storage.Service, boardID, postID string, boardownerInfo *peoplerpc.ConciseProfileReply, profileID int, fileType, owner string, tagsArr []string, uploadDate string, limit int, page string, l string,
) (map[string]interface{}, error) {
	var files []*model.UploadedFile

	dbconn, err := db.New(consts.Board)
	if err != nil {
		return nil, err
	}
	boardCollection, boardClient := dbconn.Collection, dbconn.Client
	defer boardClient.Disconnect(context.TODO())

	profileStr := strconv.Itoa(profileID)
	isValid, err := permissions.CheckValidPermissions(profileStr, cache, boardCollection, boardID, []string{consts.Blocked}, true)
	if err != nil {
		return nil, err
	}
	if !isValid {
		return nil, errors.New("User does not have the access to the board!")
	}

	postObjID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return nil, err
	}

	dbconn2, err := db.New(consts.File)
	if err != nil {
		return nil, errors.Wrap(err, "unable to establish connection with File collection.")
	}

	fileCollection, fileClient := dbconn2.Collection, dbconn2.Client
	defer fileClient.Disconnect(context.TODO())

	findFilesFilter := bson.M{}

	findFilesFilter = bson.M{"$and": bson.A{
		bson.M{"postID": postObjID},
		bson.M{"state": bson.M{"$ne": consts.Hidden}},
		bson.M{"collectionID": primitive.NilObjectID},
	}}
	if owner != "" {
		findFilesFilter["owner"] = owner
	}
	if len(tagsArr) != 0 {
		findFilesFilter["tags"] = bson.M{"$all": tagsArr}
	}
	if fileType != "" {
		findFilesFilter["fileExt"] = "." + fileType
	}
	if uploadDate != "" {
		copyDate := uploadDate
		custom := "2006-01-02T15:04:05Z"

		start := copyDate + "T00:00:00Z"
		dayStart, _ := time.Parse(custom, start)

		uploadDate = uploadDate + "T11:59:59Z"
		dayEnd, _ := time.Parse(custom, uploadDate)

		findFilesFilter["createDate"] = bson.M{"$gte": dayStart, "$lte": dayEnd}
	}

	// total count
	total, err := fileCollection.CountDocuments(context.TODO(), findFilesFilter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find total count")
	}

	var opts *options.FindOptions
	var curr *mongo.Cursor

	findOptions := options.Find()
	findOptions.SetSort(bson.M{"createDate": -1})

	if limit != 0 {
		opts = options.Find().SetLimit(int64(limit))
		curr, err = fileCollection.Find(context.TODO(), findFilesFilter, opts, findOptions)
	} else {
		// pagination
		if l != "" && page != "" {
			pgInt, _ := strconv.Atoi(page)
			limitInt, _ := strconv.Atoi(l)
			offset := (pgInt - 1) * limitInt
			findOptions.SetSkip(int64(offset))
			findOptions.SetLimit(int64(limitInt))
		}
		curr, err = fileCollection.Find(context.TODO(), findFilesFilter, findOptions)
	}

	if err != nil {
		return nil, errors.Wrap(err, "unable to find file")
	}
	err = curr.All(context.TODO(), &files)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch files")
	}
	if len(files) == 0 {
		if l != "" {
			return util.SetPaginationResponse(nil, 0, 1, "Post has no files. Please add one."), nil
		} else {
			return util.SetResponse(nil, 1, "Post has no files. Please add one."), nil
		}
	}

	for idx := range files {
		fileOwner, _ := strconv.Atoi(files[idx].Owner)
		// ownerInfo, err := profileService.FetchConciseProfile(fileOwner)
		// if err != nil {
		// 	return nil, errors.Wrap(err, "unable to fetch basic info")
		// }

		cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(fileOwner)}
		ownerInfo, err := profileService.GetConciseProfile(context.TODO(), cpreq)
		if err != nil {
			return nil, errors.Wrap(err, "unable to fetch basic info")
		}

		ownerInfo.Id = int32(fileOwner)
		files[idx].OwnerInfo = ownerInfo

		// presign beforehand
		key := ""
		if files[idx].CollectionID == primitive.NilObjectID {
			key = util.GetKeyForBoardPostMedia(int(boardownerInfo.AccountID), int(boardownerInfo.Id), boardID, postID, "")
		} else {
			key = util.GetKeyForPostCollectionMedia(int(boardownerInfo.AccountID), int(boardownerInfo.Id), boardID, postID, files[idx].CollectionID.Hex(), "")
		}

		fileName := fmt.Sprintf("%s%s", files[idx].Id.Hex(), files[idx].FileExt)
		f, err := storageService.GetUserFile(key, fileName)
		if err != nil {
			return nil, errors.Wrap(err, "unable to presign image")
		}
		files[idx].URL = f.Filename

		// Fetch Thumbnails
		thumbKey := ""
		if files[idx].CollectionID == primitive.NilObjectID {
			thumbKey = util.GetKeyForBoardPostMedia(int(boardownerInfo.AccountID), int(boardownerInfo.Id), boardID, postID, "thumbs")
		} else {
			thumbKey = util.GetKeyForPostCollectionMedia(int(boardownerInfo.AccountID), int(boardownerInfo.Id), boardID, postID, files[idx].CollectionID.Hex(), "thumbs")
		}

		thumbfileName := files[idx].Id.Hex() + ".png"
		files[idx].Thumbs, err = helper.GetThumbnails(storageService, thumbKey, thumbfileName, []string{})
		if err != nil {
			files[idx].Thumbs = model.Thumbnails{}
		}

		// reaction count
		if util.Contains(files[idx].Likes, profileStr) {
			files[idx].IsLiked = true
		} else {
			files[idx].IsLiked = false
		}
		files[idx].TotalComments = len(files[idx].Comments)
		files[idx].TotalLikes = len(files[idx].Likes)

		// get location
		loc, err := boardService.GetThingLocationOnBoard(files[idx].BoardID.Hex())
		if err != nil {
			continue
		}
		files[idx].Location = loc
	}

	return util.SetPaginationResponse(files, int(total), 1, "Files fetched successfully."), nil
}

func getFileByMediaIDforPost(db *mongodatabase.DBConfig, mysql *database.Database, cache *cache.Cache,
	boardService board.Service, profileService peoplerpc.AccountServiceClient, storageService storage.Service, boardID, postID, mediaID string, boardownerInfo *peoplerpc.ConciseProfileReply, profileID int) (map[string]interface{}, error) {
	var file model.UploadedFile

	dbconn2, err := db.New(consts.File)
	if err != nil {
		return nil, errors.Wrap(err, "unable to establish connection with File collection.")
	}

	fileCollection, fileClient := dbconn2.Collection, dbconn2.Client
	defer fileClient.Disconnect(context.TODO())

	mediaObjID, err := primitive.ObjectIDFromHex(mediaID)
	if err != nil {
		return nil, errors.Wrap(err, "Error while converting object id")
	}

	filter := bson.M{"_id": mediaObjID}

	err = fileCollection.FindOne(context.TODO(), filter).Decode(&file)
	if err != nil {
		return nil, errors.Wrap(err, "File not found")
	}

	fileOwner, _ := strconv.Atoi(file.Owner)
	// ownerInfo, err := profileService.FetchConciseProfile(fileOwner)
	// if err != nil {
	// 	return nil, errors.Wrap(err, "unable to fetch basic info")
	// }

	cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(fileOwner)}
	ownerInfo, err := profileService.GetConciseProfile(context.TODO(), cpreq)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch basic info")
	}

	file.OwnerInfo = ownerInfo

	// presign beforehand
	key := ""
	if file.CollectionID == primitive.NilObjectID {
		key = util.GetKeyForBoardPostMedia(int(boardownerInfo.AccountID), int(boardownerInfo.Id), boardID, postID, "")
	} else {
		key = util.GetKeyForPostCollectionMedia(int(boardownerInfo.AccountID), int(boardownerInfo.Id), boardID, postID, file.CollectionID.Hex(), "")
	}

	fileName := fmt.Sprintf("%s%s", file.Id.Hex(), file.FileExt)
	f, err := storageService.GetUserFile(key, fileName)
	if err != nil {
		return nil, errors.Wrap(err, "unable to presign image")
	}
	file.URL = f.Filename

	// Fetch Thumbnails
	thumbKey := ""
	if file.CollectionID == primitive.NilObjectID {
		thumbKey = util.GetKeyForBoardPostMedia(int(boardownerInfo.AccountID), int(boardownerInfo.Id), boardID, postID, "thumbs")
	} else {
		thumbKey = util.GetKeyForPostCollectionMedia(int(boardownerInfo.AccountID), int(boardownerInfo.Id), boardID, postID, file.CollectionID.Hex(), "thumbs")
	}

	thumbfileName := file.Id.Hex() + ".png"
	file.Thumbs, err = helper.GetThumbnails(storageService, thumbKey, thumbfileName, []string{})
	if err != nil {
		file.Thumbs = model.Thumbnails{}
	}

	// reaction count
	if util.Contains(file.Likes, fmt.Sprint(profileID)) {
		file.IsLiked = true
	} else {
		file.IsLiked = false
	}
	file.TotalComments = len(file.Comments)
	file.TotalLikes = len(file.Likes)

	// get location
	loc, err := boardService.GetThingLocationOnBoard(file.BoardID.Hex())
	if err != nil {
		log.Println("error", err)
	}
	file.Location = loc
	return util.SetResponse(file, 1, "File fetched successfully."), nil
}

func getFileByName(db *mongodatabase.DBConfig, cache *cache.Cache, boardID, fileName string, profileID int) (map[string]interface{}, error) {
	dbconn1, err := db.New(consts.Board)
	if err != nil {
		return nil, err
	}

	boardCollection, boardClient := dbconn1.Collection, dbconn1.Client
	defer boardClient.Disconnect(context.TODO())

	profileStr := strconv.Itoa(profileID)
	isValid, err := permissions.CheckValidPermissions(profileStr, cache, boardCollection, boardID, []string{"blocked"}, true)
	if err != nil {
		return nil, err
	}
	if !isValid {
		return nil, errors.New("User does not have the access to the board!")
	}

	dbconn, err := db.New(consts.File)
	if err != nil {
		return nil, err
	}

	var file *model.UploadedFile

	fileCollection, fileClient := dbconn.Collection, dbconn.Client
	defer fileClient.Disconnect(context.TODO())
	err = fileCollection.FindOne(context.TODO(), bson.M{"fileName": fileName}).Decode(&file)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch file")
	}

	return util.SetResponse(file, 1, "File fetched successfully."), nil
}

func addFile(cache *cache.Cache, profileService peoplerpc.AccountServiceClient, storageService storage.Service, db *mongodatabase.DBConfig, boardID, postID string, file map[string]interface{}, profileID int) (map[string]interface{}, error) {
	dbconn, err := db.New(consts.Board)
	if err != nil {
		return nil, err
	}

	boardCollection, boardClient := dbconn.Collection, dbconn.Client
	defer boardClient.Disconnect(context.TODO())

	// checking if the user has valid permission on the board
	profileStr := strconv.Itoa(profileID)
	isValid, err := permissions.CheckValidPermissions(profileStr, cache, boardCollection, file["boardID"].(primitive.ObjectID).Hex(), []string{consts.Owner, consts.Admin, consts.Author}, false)
	if err != nil {
		return nil, err
	}
	if !isValid {
		return nil, errors.New("User does not have the access to the board!")
	}

	postObjID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert to objectID")
	}

	dbconn2, err := db.New(consts.File)
	if err != nil {
		return nil, errors.Wrap(err, "unable to connec to file collection.")
	}

	fileCollection, fileClient := dbconn2.Collection, dbconn2.Client
	defer fileClient.Disconnect(context.TODO())

	file["_id"] = primitive.NewObjectID()
	file["postID"] = postObjID
	file["createDate"] = time.Now()
	file["type"] = consts.FileType
	if file["state"] == "" {
		file["state"] = consts.Active
	}
	file["fileType"] = util.ReturnFileType(file["fileMime"].(string))

	_, err = fileCollection.InsertOne(context.TODO(), file)
	if err != nil {
		return nil, errors.Wrap(err, "unable to insert file metadata at mongo")
	}

	return util.SetResponse(file, 1, "File meta data inserted successfully."), nil
}

func updateFile(cache *cache.Cache, db *mongodatabase.DBConfig, payload map[string]interface{}, boardID, postID, thingID string, profileID int) (map[string]interface{}, error) {
	profileStr := strconv.Itoa(profileID)
	dbconn2, err := db.New(consts.File)
	if err != nil {
		return nil, err
	}

	fileCollection, fileClient := dbconn2.Collection, dbconn2.Client
	defer fileClient.Disconnect(context.TODO())

	fileObjID, err := primitive.ObjectIDFromHex(thingID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to ObjectID")
	}

	// file filter
	fileFilter := bson.M{"_id": fileObjID}

	dbconn, err := db.New(consts.Board)
	if err != nil {
		return nil, err
	}

	boardCollection, boardClient := dbconn.Collection, dbconn.Client
	defer boardClient.Disconnect(context.TODO())

	// check if the user has valid permissions on the board
	isValid, err := permissions.CheckValidPermissions(profileStr, cache, boardCollection, boardID, []string{consts.Owner, consts.Admin}, false)
	if err != nil {
		return nil, err
	}
	if !isValid {
		return nil, errors.New("User does not have the access to the board!")
	}

	postObjID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return nil, err
	}

	payload["_id"] = fileObjID
	payload["postID"] = postObjID
	payload["modifiedDate"] = time.Now()

	_, err = fileCollection.UpdateOne(context.TODO(), fileFilter, bson.M{"$set": payload})
	if err != nil {
		return nil, errors.Wrap(err, "unable to update file at mongo")
	}
	// presign URL??

	return util.SetResponse(payload, 1, "File metadate updated successfully."), nil
}

func deleteFile(cache *cache.Cache, db *mongodatabase.DBConfig, boardID, postID, mediaID string, profileID int) (map[string]interface{}, error) {
	dbconn, err := db.New(consts.File)
	if err != nil {
		return nil, err
	}

	fileCollection, fileClient := dbconn.Collection, dbconn.Client
	defer fileClient.Disconnect(context.TODO())

	fileObjID, err := primitive.ObjectIDFromHex(mediaID)
	if err != nil {
		return nil, err
	}

	dbconn1, err := db.New(consts.Board)
	if err != nil {
		return nil, err
	}

	boardCollection, boardClient := dbconn1.Collection, dbconn1.Client
	defer boardClient.Disconnect(context.TODO())

	// find file filter
	filter := bson.M{"_id": fileObjID}

	var fileToDelete model.UploadedFile
	err = fileCollection.FindOne(context.TODO(), filter).Decode(&fileToDelete)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to find file")
	}

	profileStr := strconv.Itoa(profileID)

	// check if the user has valid permission on the board
	isValid, err := permissions.CheckValidPermissions(profileStr, cache, boardCollection, boardID, []string{consts.Owner, consts.Admin}, false)
	if err != nil {
		return nil, err
	}
	if !isValid && fileToDelete.Owner != profileStr {
		return util.SetResponse(nil, 0, "User does not have the authority to delete this file"), nil
	}

	// find file filter
	deleteFileFilter := bson.M{"_id": fileObjID}

	_, err = fileCollection.DeleteOne(context.TODO(), deleteFileFilter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to delete file at mongo")
	}

	return util.SetResponse(nil, 1, "File metadata deleted successfully."), nil
}

func getFileByID(db *mongodatabase.DBConfig, boardService board.Service, profileService peoplerpc.AccountServiceClient,
	storageService storage.Service, fileID string, profileID int,
) (map[string]interface{}, error) {
	fileObjID, err := primitive.ObjectIDFromHex(fileID)
	if err != nil {
		return nil, err
	}
	var key string
	dbconn, err := db.New(consts.File)
	if err != nil {
		return nil, err
	}

	var file model.UploadedFile

	fileCollection, fileClient := dbconn.Collection, dbconn.Client
	defer fileClient.Disconnect(context.TODO())
	err = fileCollection.FindOne(context.TODO(), bson.M{"_id": fileObjID}).Decode(&file)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch file")
	}

	// If file state "HIDDEN" => it is a deleted file
	if file.State == consts.Hidden {
		return util.SetResponse(nil, 0, "The requested media has been already deleted"), nil
	}

	// get location
	loc, err := boardService.GetThingLocationOnBoard(file.BoardID.Hex())
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch things location")
	}
	file.Location = loc

	// get the basic info
	idInt, _ := strconv.Atoi(file.Owner)
	// cp, err := profileService.FetchConciseProfile(idInt)
	// if err != nil {
	// 	return nil, errors.Wrap(err, "unable to find basic info")
	// }

	cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(idInt)}
	cp, err := profileService.GetConciseProfile(context.TODO(), cpreq)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch basic info")
	}

	file.OwnerInfo = cp

	boardInfo, err := boardService.FetchBoardInfo(file.BoardID.Hex())
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch board info")
	}
	boardOwnerInt, err := strconv.Atoi(boardInfo["owner"].(string))
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to int")
	}
	// ownerInfo, err := profileService.FetchConciseProfile(boardOwnerInt)
	// if err != nil {
	// 	return nil, errors.Wrap(err, "unable to fetch concise profile")
	// }

	cpreq = &peoplerpc.ConciseProfileRequest{ProfileId: int32(boardOwnerInt)}
	ownerInfo, err := profileService.GetConciseProfile(context.TODO(), cpreq)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch basic info")
	}

	// presign beforehand
	if file.CollectionID == primitive.NilObjectID {
		key = util.GetKeyForBoardPostMedia(int(ownerInfo.AccountID), int(ownerInfo.Id), file.BoardID.Hex(), file.PostID.Hex(), "")
	} else {
		key = util.GetKeyForPostCollectionMedia(int(ownerInfo.AccountID), int(ownerInfo.Id), file.BoardID.Hex(), file.PostID.Hex(), file.CollectionID.Hex(), "")
	}

	fileName := fmt.Sprintf("%s%s", file.Id.Hex(), file.FileExt)
	f, err := storageService.GetUserFile(key, fileName)
	if err != nil {
		return nil, errors.Wrap(err, "unable to presign the file")
	}
	file.URL = f.Filename

	// getting the reactions
	if util.Contains(file.Likes, strconv.Itoa(profileID)) {
		file.IsLiked = true
	} else {
		file.IsLiked = false
	}
	file.TotalComments = len(file.Comments)
	file.TotalLikes = len(file.Likes)

	// Fetch Thumbnails
	thumbKey := ""
	if file.CollectionID == primitive.NilObjectID {
		thumbKey = util.GetKeyForBoardPostMedia(int(ownerInfo.AccountID), int(ownerInfo.Id), file.BoardID.Hex(), file.PostID.Hex(), "thumbs")
	} else {
		thumbKey = util.GetKeyForPostCollectionMedia(int(ownerInfo.AccountID), int(ownerInfo.Id), file.BoardID.Hex(), file.PostID.Hex(), file.CollectionID.Hex(), "thumbs")
	}

	thumbfileName := file.Id.Hex() + ".png"
	file.Thumbs, err = helper.GetThumbnails(storageService, thumbKey, thumbfileName, []string{})
	if err != nil {
		file.Thumbs = model.Thumbnails{}
	}

	return util.SetResponse(file, 1, "File info fetched successfully"), nil
}

func fetchFilesByProfile(cache *cache.Cache, db *mongodatabase.DBConfig, mysql *database.Database,
	boardID string, profileID, limit int, publicOnly bool,
) (map[string]interface{}, error) {
	profileIDStr := strconv.Itoa(profileID)
	dbConn, err := db.New(consts.File)
	if err != nil {
		return nil, err
	}
	fileCollection, fileClient := dbConn.Collection, dbConn.Client
	defer fileClient.Disconnect(context.TODO())
	var curr *mongo.Cursor
	var findFilter primitive.M
	allFiles := make(map[string][]*model.UploadedFile)
	var res interface{}
	boardObjID, _ := primitive.ObjectIDFromHex(boardID)

	totalGoroutines := 4
	errChan := make(chan error)
	if publicOnly {
		totalGoroutines = 1
	}
	go func(errChan chan<- error) {
		defer util.RecoverGoroutinePanic(errChan)
		findFilter = bson.M{"visible": "PUBLIC", "boardID": boardObjID}
		files, err := fetchFilesByFilter(fileCollection, mysql, curr, findFilter, limit)
		if err != nil {
			errChan <- errors.Wrap(err, "unable to fetch public files")
		}
		if len(files) > 0 {
			if publicOnly {
				res = files
			} else {
				allFiles["public"] = files
			}
		} else {
			if publicOnly {
				res = nil
			} else {
				allFiles["public"] = nil
			}
		}
		errChan <- nil
	}(errChan)
	if !publicOnly {
		go func(errChan chan<- error) {
			defer util.RecoverGoroutinePanic(errChan)
			findFilter = bson.M{"owner": profileIDStr, "boardID": boardObjID}
			files, err := fetchFilesByFilter(fileCollection, mysql, curr, findFilter, limit)
			if err != nil {
				errChan <- errors.Wrap(err, "unable to fetch  private files")
			}
			if len(files) > 0 {
				allFiles["private"] = files
			} else {
				allFiles["private"] = nil
			}
			errChan <- nil
		}(errChan)
		go func(errChan chan<- error) {
			defer util.RecoverGoroutinePanic(errChan)
			findFilter = bson.M{"visible": "MEMBERS", "boardID": boardObjID}
			files, err := fetchFilesByFilter(fileCollection, mysql, curr, findFilter, limit)
			if err != nil {
				errChan <- errors.Wrap(err, "unable to fetch members files")
			}
			if len(files) > 0 {
				allFiles["members"] = files
			} else {
				allFiles["members"] = nil
			}
			errChan <- nil
		}(errChan)
		go func(errChan chan<- error) {
			defer util.RecoverGoroutinePanic(errChan)
			// fetch profile connections
			dbConn, err := db.New(consts.Connection)
			if err != nil {
				errChan <- err
			}
			connCollection, connClient := dbConn.Collection, dbConn.Client
			defer connClient.Disconnect(context.TODO())

			findConnFilter := bson.M{"profileID": profileIDStr}
			cursor, err := connCollection.Find(context.TODO(), findConnFilter)
			if err != nil {
				errChan <- errors.Wrap(err, "unable to find boards")
			}
			connections := []model.BoardMemberRole{}
			err = cursor.All(context.TODO(), &connections)
			if err != nil {
				errChan <- errors.Wrap(err, "unable to find profile's connections.")
			}
			var connectionArr []string
			for _, member := range connections {
				connectionArr = append(connectionArr, member.ProfileID)
			}
			if len(connectionArr) > 0 {
				// fetch files filter
				findFilter = bson.M{"visible": "CONTACTS", "boardID": boardObjID, "owner": bson.M{"$in": connectionArr}}
				files, err := fetchFilesByFilter(fileCollection, mysql, curr, findFilter, limit)
				if err != nil {
					errChan <- errors.Wrap(err, "unable to fetch contact files")
				}
				if len(files) > 0 {
					allFiles["contacts"] = files
				} else {
					allFiles["contacts"] = nil
				}
			} else {
				allFiles["contacts"] = nil
			}
			errChan <- nil
		}(errChan)
	}
	for i := 0; i < totalGoroutines; i++ {
		if err := <-errChan; err != nil {
			return nil, errors.Wrap(err, "error fromfetchFilesByProfile go-routine")
		}
	}
	if publicOnly {
		return util.SetResponse(res, 1, "files fetched successfully."), nil
	}
	return util.SetResponse(allFiles, 1, "files fetched successfully."), nil
}

func fetchFilesByFilter(fileCollection *mongo.Collection, mysql *database.Database, curr *mongo.Cursor, findFilter primitive.M, limit int) (files []*model.UploadedFile, err error) {
	findOptions := options.Find()
	findOptions.SetSort(bson.M{"createDate": -1})
	if limit != 0 {
		opts := options.Find().SetLimit(int64(limit))
		curr, err = fileCollection.Find(context.TODO(), findFilter, opts, findOptions)
	} else {
		curr, err = fileCollection.Find(context.TODO(), findFilter, findOptions)
	}
	if err != nil {
		return nil, errors.Wrap(err, "unable to find files")
	}
	err = curr.All(context.TODO(), &files)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch files")
	}
	// map owner profile
	errChan := make(chan error)
	for index := range files {
		go func(i int, errChan chan<- error) {
			defer util.RecoverGoroutinePanic(errChan)

			ownerInfo := model.ConciseProfile{}
			stmt := `SELECT id, firstName, lastName,
							IFNULL(screenName, '') AS screenName,
							IFNULL(photo, '') AS photo FROM` + "`sidekiq-dev`.AccountProfile WHERE id = ?"
			itemOwner, _ := strconv.Atoi(files[i].Owner)
			err = mysql.Conn.Get(&ownerInfo, stmt, itemOwner)
			if err != nil {
				errChan <- errors.Wrap(err, "unable to map profile info")
			}
			files[i].OwnerInfo = &peoplerpc.ConciseProfileReply{
				Id:         int32(ownerInfo.Id),
				AccountID:  int32(ownerInfo.UserID),
				FirstName:  ownerInfo.FirstName,
				LastName:   ownerInfo.LastName,
				ScreenName: ownerInfo.ScreenName,
				Photo:      ownerInfo.Photo,
			}
			errChan <- nil
		}(index, errChan)
	}
	totalGoroutines := len(files)
	for i := 0; i < totalGoroutines; i++ {
		if err := <-errChan; err != nil {
			return nil, errors.Wrap(err, "error fromfetchFilesByFilter go-routine")
		}
	}
	return
}

func getFilesByPost2(db *mongodatabase.DBConfig, mysql *database.Database, cache *cache.Cache,
	boardService board.Service, profileService peoplerpc.AccountServiceClient, storageService storage.Service, boardID, postID string) ([]map[string]interface{}, error) {
	dbconn, err := db.New(consts.File)
	if err != nil {
		return nil, err
	}
	fileColl, fileClient := dbconn.Collection, dbconn.Client
	defer fileClient.Disconnect(context.TODO())

	postObjID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to objectID")
	}

	cur, err := fileColl.Find(context.TODO(), bson.M{"postID": postObjID, "collectionID": primitive.NilObjectID})
	if err != nil {
		return nil, errors.Wrap(err, "files of post not found")
	}

	var files []map[string]interface{}
	err = cur.All(context.TODO(), &files)
	if err != nil {
		return nil, errors.Wrap(err, "unable to unpack files")
	}
	// get the board owner
	boardInfo, err := boardService.FetchBoardInfo(boardID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find board")
	}
	boardOwnerInt, err := strconv.Atoi(boardInfo["owner"].(string))
	if err != nil {
		return nil, err
	}
	// boardownerInfo, err := profileService.FetchConciseProfile(boardOwnerInt)
	// if err != nil {
	// 	return nil, err
	// }

	cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(boardOwnerInt)}
	boardownerInfo, err := profileService.GetConciseProfile(context.TODO(), cpreq)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch basic info")
	}

	// presign the files
	for idx := range files {
		key := ""
		if files[idx]["collectionID"].(primitive.ObjectID) == primitive.NilObjectID {
			key = util.GetKeyForBoardPostMedia(int(boardownerInfo.AccountID), int(boardownerInfo.Id), boardID, postID, "")
		} else {
			key = util.GetKeyForPostCollectionMedia(int(boardownerInfo.AccountID), int(boardownerInfo.Id), boardID, postID, files[idx]["collectionID"].(primitive.ObjectID).Hex(), "")
		}

		fileName := fmt.Sprintf("%s%s", files[idx]["_id"].(primitive.ObjectID).Hex(), files[idx]["fileExt"].(string))
		f, err := storageService.GetUserFile(key, fileName)
		if err != nil {
			return nil, errors.Wrap(err, "unable to presign image")
		}
		files[idx]["url"] = f.Filename

		// Fetch Thumbnails
		thumbKey := ""
		if files[idx]["collectionID"].(primitive.ObjectID) == primitive.NilObjectID {
			thumbKey = util.GetKeyForBoardPostMedia(int(boardownerInfo.AccountID), int(boardownerInfo.Id), boardID, postID, "thumbs")
		} else {
			thumbKey = util.GetKeyForPostCollectionMedia(int(boardownerInfo.AccountID), int(boardownerInfo.Id), boardID, postID, files[idx]["collectionID"].(primitive.ObjectID).Hex(), "thumbs")
		}

		thumbfileName := files[idx]["_id"].(primitive.ObjectID).Hex() + ".png"
		files[idx]["thumbs"], err = helper.GetThumbnails(storageService, thumbKey, thumbfileName, []string{})
		if err != nil {
			files[idx]["thumbs"] = model.Thumbnails{}
		}
	}

	return files, nil
}
