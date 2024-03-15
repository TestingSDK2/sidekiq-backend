package collection

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	peoplerpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/storage"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/cache"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/consts"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/database"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/helper"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/mongodatabase"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/permissions"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/util"
	model "github.com/TestingSDK2/sidekiq-backend/sidekiq-models"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func addCollection(cache *cache.Cache, db *mongodatabase.DBConfig, mysql *database.Database, profileService peoplerpc.AccountServiceClient,
	storageService storage.Service, payload model.Collection, profileID int, boardID, postID string,
) (map[string]interface{}, error) {
	dbconn1, err := db.New(consts.Board)
	if err != nil {
		return nil, err
	}

	boardCollection, boardClient := dbconn1.Collection, dbconn1.Client
	defer boardClient.Disconnect(context.TODO())

	profileIDStr := strconv.Itoa(profileID)
	isValid, err := permissions.CheckValidPermissions(profileIDStr, cache, boardCollection, boardID, []string{consts.Owner, consts.Admin, consts.Author}, false)
	if err != nil {
		return nil, err
	}
	if !isValid {
		return util.SetResponse(nil, 0, "User does not have access to the board."), nil
	}

	dbconn, err := db.New(consts.Collection)
	if err != nil {
		return nil, err
	}

	collection, client := dbconn.Collection, dbconn.Client
	defer client.Disconnect(context.TODO())

	postObjID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return nil, err
	}

	// fixed
	payload.Id = primitive.NewObjectID()
	payload.PostID = postObjID
	payload.CreateDate = time.Now()
	payload.ModifiedDate = time.Now()
	payload.Type = "COLLECTION"
	payload.Owner = strconv.Itoa(profileID)
	if len(payload.Things) == 0 {
		payload.Things = []model.Things{}
	}

	// add owner info
	// cp, err := profileService.FetchConciseProfile(profileID)
	// if err != nil {
	// 	if errors.Is(err, sql.ErrNoRows) {
	// 		return util.SetResponse(nil, 0, "owner does not exist in mysql"), nil
	// 	}
	// 	return nil, errors.Wrap(err, "unable to find owner's info.")
	// }

	cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(profileID)}
	cp, err := profileService.GetConciseProfile(context.TODO(), cpreq)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return util.SetResponse(nil, 0, "owner does not exist in mysql"), nil
		}
		return nil, errors.Wrap(err, "unable to find owner's info.")
	}

	cp.Id = int32(profileID)

	_, err = collection.InsertOne(context.TODO(), payload)
	if err != nil {
		return nil, errors.Wrap(err, "unable to insert board at mongo")
	}

	payload.OwnerInfo = cp
	return util.SetResponse(payload, 1, "Collection added Successfully"), nil
}

func getCollection(cache *cache.Cache, db *mongodatabase.DBConfig, mysql *database.Database, profileService peoplerpc.AccountServiceClient,
	storageService storage.Service, boardID, postID string, profileID int, owner string, tagsArr []string, uploadDate string, limit int, page, l string,
) (map[string]interface{}, error) {
	dbconn1, err := db.New(consts.Board)
	if err != nil {
		return nil, err
	}
	boardCollection, boardClient := dbconn1.Collection, dbconn1.Client
	defer boardClient.Disconnect(context.TODO())

	profileIDStr := strconv.Itoa(profileID)
	isValid, err := permissions.CheckValidPermissions(profileIDStr, cache, boardCollection, boardID, []string{"blocked"}, true)
	if err != nil {
		return nil, err
	}
	if !isValid {
		return util.SetResponse(nil, 0, "User does not have access to the board."), nil
	}
	var collections []*model.Collection
	dbconn, err := db.New(consts.Collection)
	if err != nil {
		return nil, err
	}

	collection, client := dbconn.Collection, dbconn.Client
	defer client.Disconnect(context.TODO())

	var opts *options.FindOptions
	var curr *mongo.Cursor

	postObjID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert board id in get collection")
	}

	findFilter := bson.M{"$and": bson.A{
		bson.M{"postID": postObjID},
		bson.M{"state": bson.M{"$ne": consts.Hidden}},
	}}
	if owner != "" {
		findFilter["owner"] = owner
	}
	if len(tagsArr) != 0 {
		findFilter["tags"] = bson.M{"$all": tagsArr}
	}
	if uploadDate != "" {
		copyDate := uploadDate
		custom := "2006-01-02T15:04:05Z"

		start := copyDate + "T00:00:00Z"
		dayStart, _ := time.Parse(custom, start)

		uploadDate = uploadDate + "T11:59:59Z"
		dayEnd, _ := time.Parse(custom, uploadDate)

		findFilter["createDate"] = bson.M{"$gte": dayStart, "$lte": dayEnd}
	}

	findOptions := options.Find()
	findOptions.SetSort(bson.M{"createDate": -1})
	var isPaginated bool = false

	total, err := collection.CountDocuments(context.TODO(), findFilter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find total count")
	}
	// var allCurr *mongo.Cursor
	// var allCollections []*model.Collection
	if limit != 0 {
		opts = options.Find().SetLimit(int64(limit))
		curr, err = collection.Find(context.TODO(), findFilter, opts, findOptions)
		// allCurr, err = collection.Find(context.TODO(), findFilter, opts, findOptions)
		// if err != nil {
		// 	fmt.Println("unable to retrieve all collections")
		// }
	} else {
		// pagination
		if l != "" && page != "" {
			pgInt, _ := strconv.Atoi(page)
			limitInt, _ := strconv.Atoi(l)
			offset := (pgInt - 1) * limitInt
			findOptions.SetSkip(int64(offset))
			findOptions.SetLimit(int64(limitInt))
			isPaginated = true
		}
		curr, err = collection.Find(context.TODO(), findFilter, findOptions)
	}
	if err != nil {
		return nil, errors.Wrap(err, "unable to find collection")
	}
	err = curr.All(context.TODO(), &collections)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch collection")
	}
	for i := range collections {
		collectionOwner, _ := strconv.Atoi(collections[i].Owner)
		// ownerInfo, err := profileService.FetchConciseProfile(collectionOwner)
		// if err != nil {
		// 	return nil, errors.Wrap(err, "unable to find basic info")
		// }

		cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(collectionOwner)}
		ownerInfo, err := profileService.GetConciseProfile(context.TODO(), cpreq)
		if err != nil {
			return nil, errors.Wrap(err, "unable to find basic info")
		}

		ownerInfo.Id = int32(collectionOwner)
		collections[i].OwnerInfo = ownerInfo

		image, err := getCollectionCover(db, collections[i].Id, storageService)
		if err != nil {
			fmt.Println("error in fetching collection cover", err)
		}
		collections[i].CoverImage = *image

		if util.Contains(collections[i].Likes, fmt.Sprint(profileID)) {
			collections[i].IsLiked = true
		}
		collections[i].TotalLikes = len(collections[i].Likes)
		collections[i].TotalComments = len(collections[i].Comments)
	}
	if len(collections) == 0 {
		return util.SetPaginationResponse([]*model.Collection{}, 0, 1, "Board has no collections. Please add one."), nil
	}
	if isPaginated {
		return util.SetPaginationResponse(collections, int(total), 1, "Collection retrieved Successfully"), nil
	}
	return util.SetResponse(collections, 1, "Collection retrieved Successfully"), nil
}

func updateCollection(cache *cache.Cache, payload model.UpdateCollection, db *mongodatabase.DBConfig,
	boardID, postID, collectionID string, profileID int, storageService storage.Service,
) (map[string]interface{}, error) {
	profileIDStr := strconv.Itoa(profileID)
	var err error
	dbconn, err := db.New(consts.Collection)
	if err != nil {
		return nil, errors.Wrap(err, "unable to connect to collection")
	}
	dbconn2, err := db.New(consts.Board)
	if err != nil {
		return nil, errors.Wrap(err, "unable to connect to collection")
	}
	boardCollection, boardClient := dbconn2.Collection, dbconn2.Client
	defer boardClient.Disconnect(context.TODO())
	// check if the user has valid permissions on the board
	isValid, err := permissions.CheckValidPermissions(profileIDStr, cache, boardCollection, boardID, []string{consts.Owner, consts.Admin}, false)
	if err != nil {
		return nil, errors.Wrap(err, "unable to check permissions")
	}
	postObjID, _ := primitive.ObjectIDFromHex(postID)
	collectionObjID, _ := primitive.ObjectIDFromHex(collectionID)
	col, client := dbconn.Collection, dbconn.Client
	defer client.Disconnect(context.TODO())
	filter := bson.M{
		"$and": bson.A{
			bson.M{"postID": postObjID},
			bson.M{"_id": collectionObjID},
		},
	}
	var collection model.Collection
	err = col.FindOne(context.TODO(), filter).Decode(&collection)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find collection")
	}
	if !isValid && collection.Owner != profileIDStr {
		return util.SetResponse(nil, 0, "User does not have access to the collection."), nil
	}
	collection.Title = payload.Title
	collection.Tags = payload.Tags
	collection.ModifiedDate = time.Now()
	collection.PostID = postObjID

	res, err := col.UpdateOne(context.TODO(), filter, bson.M{"$set": collection})
	if err != nil || res.ModifiedCount == 0 {
		return nil, errors.Wrap(err, "unable to update board at mongo")
	}
	image, err := getCollectionCover(db, collection.Id, storageService)
	if err != nil {
		fmt.Println("error in fetching collection cover", err)
	}
	collection.CoverImage = *image

	return util.SetResponse(&collection, 1, "Collection updated successfully."), nil
}

func getCollectionCover(db *mongodatabase.DBConfig, collectionID primitive.ObjectID, storageService storage.Service) (*string, error) {
	dbconn, err := db.New(consts.File)
	if err != nil {
		return nil, err
	}

	fileCollection, client := dbconn.Collection, dbconn.Client
	defer client.Disconnect(context.TODO())
	imageLink := ""
	opts := options.FindOne()
	opts.SetSort(bson.M{"createDate": -1})

	filter := bson.M{"$and": bson.A{
		bson.M{"collectionID": collectionID},
		bson.M{"fileType": "image"},
		bson.M{"state": "ACTIVE"},
	}}

	var document model.UploadedFile
	err = fileCollection.FindOne(context.Background(), filter, opts).Decode(&document)
	if err != nil {
		return &imageLink, errors.Wrap(err, "no images found for this collection")
	}

	// key := util.GetKeyForPostCollectionMedia(document.BoardID.Hex(), document.PostID.Hex(), collectionID.Hex(), "")
	key := ""
	fileName := fmt.Sprintf("%s%s", document.Id.Hex(), document.FileExt)
	fmt.Println(key, fileName)
	f, err := storageService.GetUserFile(key, fileName)
	if err != nil {
		return &imageLink, errors.Wrap(err, "unable to presign image")
	}
	return &f.Filename, nil
}

func getCollectionByID(cache *cache.Cache, db *mongodatabase.DBConfig, mysql *database.Database, profileService peoplerpc.AccountServiceClient,
	storageService storage.Service, boardID, postID, collectionID string, profileID int,
) (map[string]interface{}, error) {
	// profileIDStr := strconv.Itoa(profileID)
	// isValid := permissions.CheckValidPermissions(profileIDStr, cache, boardID, []string{"blocked"}, true)
	// if !isValid {
	// 	return util.SetResponse(nil, 0, "User does not have access to the board."), nil
	// }
	var collectionRes *model.Collection
	dbconn, err := db.New(consts.Collection)
	if err != nil {
		return nil, err
	}

	collection, client := dbconn.Collection, dbconn.Client
	defer client.Disconnect(context.TODO())

	// boardObjID, err := primitive.ObjectIDFromHex(boardID)
	// if err != nil {
	// 	return nil, errors.Wrap(err, "unable to convert board id in get collection")
	// }
	collectionObjID, err := primitive.ObjectIDFromHex(collectionID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert board id in get collection")
	}
	// findFilter := bson.M{"boardID": boardObjID}
	findFilter := bson.M{"_id": collectionObjID}
	err = collection.FindOne(context.TODO(), findFilter).Decode(&collectionRes)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find collection from the board.")
	}

	if util.Contains(collectionRes.Likes, fmt.Sprint(profileID)) {
		collectionRes.IsLiked = true
	}
	collectionRes.TotalLikes = len(collectionRes.Likes)
	collectionRes.TotalComments = len(collectionRes.Comments)

	// fetch owner info
	ownerID, _ := strconv.Atoi(collectionRes.Owner)
	// owner, err := profileService.FetchConciseProfile(ownerID)

	cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(ownerID)}
	owner, err := profileService.GetConciseProfile(context.TODO(), cpreq)
	if err != nil {
		if err == sql.ErrNoRows {
			collectionRes.OwnerInfo = nil
		} else {
			return nil, errors.Wrap(err, "unable find fetch basic info")
		}
	}

	collectionRes.OwnerInfo = owner
	return util.SetResponse(collectionRes, 1, "Collection retrieved Successfully"), nil
}

func updateCollecitonStatusByID(cache *cache.Cache, db *mongodatabase.DBConfig, mysql *database.Database, profileService peoplerpc.AccountServiceClient,
	storageService storage.Service, boardID, postID, collectionID, status string, profileID int,
) (map[string]interface{}, error) {
	collectionObjID, err := primitive.ObjectIDFromHex(collectionID)
	if err != nil {
		return nil, err
	}

	dbconn, err := db.New(consts.Collection)
	if err != nil {
		return nil, err
	}

	Collection, Client := dbconn.Collection, dbconn.Client
	defer Client.Disconnect(context.TODO())

	filter := bson.M{"_id": collectionObjID}
	update := bson.M{"$set": bson.M{"fileProcStatus": status}}
	_, err = Collection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		return nil, err
	}

	var collectionRes *model.Collection

	// findFilter := bson.M{"boardID": boardObjID}
	findFilter := bson.M{"_id": collectionObjID}
	err = Collection.FindOne(context.TODO(), findFilter).Decode(&collectionRes)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find collection from the board.")
	}

	if util.Contains(collectionRes.Likes, fmt.Sprint(profileID)) {
		collectionRes.IsLiked = true
	}
	collectionRes.TotalLikes = len(collectionRes.Likes)
	collectionRes.TotalComments = len(collectionRes.Comments)

	// fetch owner info
	ownerID, _ := strconv.Atoi(collectionRes.Owner)
	// owner, err := profileService.FetchConciseProfile(ownerID)

	cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(ownerID)}
	owner, err := profileService.GetConciseProfile(context.TODO(), cpreq)
	if err != nil {
		if err == sql.ErrNoRows {
			collectionRes.OwnerInfo = nil
		} else {
			return nil, errors.Wrap(err, "unable find fetch basic info")
		}
	}

	collectionRes.OwnerInfo = owner
	return util.SetResponse(collectionRes, 1, "Collection status updated Successfully"), nil
}

func appendThingInCollection(db *mongodatabase.DBConfig, profileService peoplerpc.AccountServiceClient,
	storageService storage.Service, payload model.Collection, boardID string, profileID int,
) (map[string]interface{}, error) {
	dbconn, err := db.New(consts.Collection)
	if err != nil {
		return nil, err
	}

	collection, client := dbconn.Collection, dbconn.Client
	defer client.Disconnect(context.TODO())

	filter := bson.M{"_id": payload.Id}
	update := bson.M{"$addToSet": bson.M{"things": payload.Things[0]}}
	_, err = collection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		return nil, errors.Wrap(err, "unable to insert things in collection")
	}

	return util.SetResponse(payload, 1, "Things added Successfully"), nil
}

func getFilesByCollection(db *mongodatabase.DBConfig, mysql *database.Database, cache *cache.Cache,
	profileService peoplerpc.AccountServiceClient, storageService storage.Service, boardID, postID, collectionID, fileName string, profileID, limit int, page string, l string, ownerInfo *peoplerpc.ConciseProfileReply,
) (map[string]interface{}, error) {
	dbconn1, err := db.New(consts.Board)
	if err != nil {
		return nil, err
	}
	boardCollection, boardClient := dbconn1.Collection, dbconn1.Client
	defer boardClient.Disconnect(context.TODO())

	var files []*model.UploadedFile
	profileStr := strconv.Itoa(profileID)
	isValid, err := permissions.CheckValidPermissions(profileStr, cache, boardCollection, boardID, []string{consts.Blocked}, true)
	if err != nil {
		return nil, err
	}
	if !isValid {
		return nil, errors.New("User does not have the access to the board!")
	}

	dbconn, err := db.New(consts.Collection)
	if err != nil {
		return nil, err
	}
	collectionCollection, collectionClient := dbconn.Collection, dbconn.Client
	defer collectionClient.Disconnect(context.TODO())

	// boardObjID, err := primitive.ObjectIDFromHex(boardID)
	// if err != nil {
	// 	return nil, err
	// }
	collectionObjID, err := primitive.ObjectIDFromHex(collectionID)
	if err != nil {
		return nil, err
	}
	var collection *model.Collection
	// colfilter := bson.M{"$and": bson.A{
	// 	bson.M{"_id": collectionObjID},
	// 	bson.M{"boardID": boardObjID},
	// }}
	colfilter := bson.M{"_id": collectionObjID}
	err = collectionCollection.FindOne(context.TODO(), colfilter).Decode(&collection)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find board")
	}

	dbconn2, err := db.New(consts.File)
	if err != nil {
		return nil, errors.Wrap(err, "unable to establish connection with File collection.")
	}

	fileCollection, fileClient := dbconn2.Collection, dbconn2.Client
	defer fileClient.Disconnect(context.TODO())

	// regexoptions:
	// i: To match both lower case and upper case pattern in the string.
	// m: To include ^ and $ in the pattern in the match i.e.
	// to specifically search for ^ and $ inside the string. Without this option, these anchors match at the beginning or end of the string.
	findFilesFilter := bson.M{
		"$and": bson.A{
			bson.M{
				"$and": bson.A{
					bson.M{"state": bson.M{"$ne": consts.Hidden}},
					bson.M{"title": bson.M{"$regex": fileName, "$options": "im"}},
				},
			},
			bson.M{"collectionID": collectionObjID},
		},
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
		response := map[string]interface{}{
			"data": map[string]interface{}{
				"info":            []int{},
				"total":           0,
				"collectionTitle": collection.Title,
			},
			"status":  1,
			"message": "Collection has no files. Please add one.",
		}
		return response, nil
	}

	for idx := range files {
		fileOwner, _ := strconv.Atoi(files[idx].Owner)
		// ownerInfo, err := profileService.FetchConciseProfile(fileOwner)

		cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(fileOwner)}
		ownerInfo, err := profileService.GetConciseProfile(context.TODO(), cpreq)
		if err != nil {
			return nil, errors.Wrap(err, "unable to fetch basic info")
		}

		ownerInfo.Id = int32(fileOwner)
		files[idx].OwnerInfo = ownerInfo

		// presign beforehand
		key := util.GetKeyForPostCollectionMedia(int(ownerInfo.AccountID), int(ownerInfo.Id), boardID, postID, collectionID, "")
		fileName := fmt.Sprintf("%s%s", files[idx].Id.Hex(), files[idx].FileExt)
		f, err := storageService.GetUserFile(key, fileName)
		if err != nil {
			return nil, errors.Wrap(err, "unable to presign image")
		}
		files[idx].URL = f.Filename

		// reaction count
		if util.Contains(files[idx].Likes, profileStr) {
			files[idx].IsLiked = true
		} else {
			files[idx].IsLiked = false
		}
		files[idx].TotalComments = len(files[idx].Comments)
		files[idx].TotalLikes = len(files[idx].Likes)

		// Fetch Thumbnails
		thumbKey := util.GetKeyForPostCollectionMedia(int(ownerInfo.AccountID), int(ownerInfo.Id), boardID, postID, collectionID, "thumbs")
		thumbfileName := files[idx].Id.Hex() + ".png"
		thumbs, err := helper.GetThumbnails(storageService, thumbKey, thumbfileName, []string{})
		if err != nil {
			thumbs = model.Thumbnails{}
		}

		files[idx].Thumbs = thumbs

	}
	response := map[string]interface{}{
		"data": map[string]interface{}{
			"info":            files,
			"total":           int(total),
			"collectionTitle": collection.Title,
		},
		"status":  1,
		"message": "Files fetched successfully.",
	}
	return response, nil
}

func deleteCollectionMedia(db *mongodatabase.DBConfig, colId, thingID string) (map[string]interface{}, error) {
	var collectionRes *model.Collection
	dbconn, err := db.New(consts.Collection)
	if err != nil {
		return nil, err
	}
	collection, client := dbconn.Collection, dbconn.Client
	defer client.Disconnect(context.TODO())

	collectionObjID, err := primitive.ObjectIDFromHex(colId)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert board id in get collection")
	}
	thingObjID, err := primitive.ObjectIDFromHex(thingID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert board id in get collection")
	}
	findFilter := bson.M{"_id": collectionObjID}
	err = collection.FindOne(context.TODO(), findFilter).Decode(&collectionRes)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find task from the board.")
	}
	var filtered []model.Things
	for _, thing := range collectionRes.Things {
		if thing.ThingID != thingObjID {
			filtered = append(filtered, thing)
		}
	}
	collectionRes.Things = filtered
	_, err = collection.UpdateOne(context.TODO(), findFilter, bson.M{"$set": collectionRes})
	if err != nil {
		return nil, errors.Wrap(err, "unable to delete things in collection")
	}
	return util.SetResponse(nil, 1, "Collection media removed successfully"), nil
}

func removeArrayElem(things []model.Things, targetID primitive.ObjectID) []model.Things {
	var filtered []model.Things
	for _, thing := range things {
		if thing.ThingID != targetID {
			filtered = append(filtered, thing)
		}
	}
	return filtered
}

func editCollectionMedia(cache *cache.Cache, db *mongodatabase.DBConfig, payload model.UpdateCollection, profileID, thingID, thingType, boardID, postID string) (map[string]interface{}, error) {
	// profileIDStr := strconv.Itoa(profileID)
	var err error
	var collection string
	switch strings.ToLower(thingType) {
	case "note":
		collection = consts.Note
	case "task":
		collection = consts.Task
	case "file":
		collection = consts.File
	default:
		return nil, errors.New("collection type invalid")
	}
	dbconn1, err := db.New(consts.Board)
	if err != nil {
		return nil, err
	}

	boardCollection, boardClient := dbconn1.Collection, dbconn1.Client
	defer boardClient.Disconnect(context.TODO())

	// check if the user has valid permissions on the board
	isValid, err := permissions.CheckValidPermissions(profileID, cache, boardCollection, boardID, []string{"owner", "admin"}, false)
	if err != nil {
		return nil, err
	}
	// Update for task and note to be handled later

	dbconn, err := db.New(collection)
	if err != nil {
		return nil, errors.Wrap(err, "unable to connect to media collection")
	}
	thingObjID, _ := primitive.ObjectIDFromHex(thingID)
	thingCollection, client := dbconn.Collection, dbconn.Client
	defer client.Disconnect(context.TODO())
	var fileData model.UploadedFile
	filter := bson.M{"_id": thingObjID}
	err = thingCollection.FindOne(context.TODO(), filter).Decode(&fileData)
	if err != nil {
		return nil, errors.Wrap(err, "unable to retrieve "+thingType+" collection")
	}
	if !isValid && fileData.Owner != profileID {
		return util.SetResponse(nil, 0, "User don't have permission to edit the file"), nil
	}
	fileData.Title = payload.Title
	fileData.Tags = payload.Tags
	// Perform the update operation
	_, err = thingCollection.UpdateOne(context.TODO(), filter, bson.M{"$set": fileData})
	if err != nil {
		return nil, errors.Wrap(err, "unable to update "+thingType+" collection")
	}
	return util.SetResponse(fileData, 1, "Collection media updated successfully."), nil
}

func deleteCollection(db *mongodatabase.DBConfig, cache *cache.Cache, boardID, postID, collectionID, profileID string) (map[string]interface{}, error) {
	dbconn, err := db.New(consts.Collection)
	if err != nil {
		return nil, err
	}

	colCollection, colClient := dbconn.Collection, dbconn.Client
	defer colClient.Disconnect(context.TODO())

	collectionObjID, err := primitive.ObjectIDFromHex(collectionID)
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
	filter := bson.M{"_id": collectionObjID}
	// filter := bson.M{"_id": collectionObjID, "state": "ACTIVE"}

	// filter := bson.M{"$and": bson.A{
	// 	bson.M{"_id": collectionObjID},
	// 	bson.M{"state": "ACTIVE"},
	// }}

	var colToDelete model.Collection
	err = colCollection.FindOne(context.TODO(), filter).Decode(&colToDelete)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find collection")
	}

	// check if the user has valid permission on the board
	isValid, err := permissions.CheckValidPermissions(profileID, cache, boardCollection, boardID, []string{"owner", "admin"}, false)
	if err != nil {
		return nil, err
	}
	if !isValid && colToDelete.Owner != profileID {
		return util.SetResponse(nil, 0, "User don't have permission to delete the file"), nil
	}

	deleteFileFilter := bson.M{"_id": collectionObjID}
	_, err = colCollection.DeleteOne(context.TODO(), deleteFileFilter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to delete file at mongo")
	}
	return util.SetResponse(nil, 1, "Collection deleted successfully"), nil
}

func updateCollectionById(db *mongodatabase.DBConfig, collectionobjectID primitive.ObjectID, payload map[string]interface{}) error {
	dbconn, err := db.New(consts.Collection)
	if err != nil {
		return err
	}

	colCollection, colClient := dbconn.Collection, dbconn.Client
	defer colClient.Disconnect(context.TODO())

	_, err = colCollection.UpdateOne(context.TODO(), bson.M{"_id": collectionobjectID}, bson.M{"$set": payload})
	if err != nil {
		return err
	}

	return nil
}
