package post

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/board"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/file"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/note"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/storage"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/task"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/cache"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/consts"
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
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	peoplerpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"
	// peoplerpc "github.com/sidekiq-people/proto/people"
)

func getFirstPostThing(db *mongodatabase.DBConfig, boardService board.Service,
	profileService peoplerpc.AccountServiceClient, storageService storage.Service, postID, boardID string, profileID int) (map[string]interface{}, error) {
	colmap := make(map[string]*mongo.Collection)
	errChan := make(chan error)
	goRoutines := 0

	var wg sync.WaitGroup
	var dbconn1, dbconn2, dbconn3, dbconn4 *mongodatabase.MongoDBConn
	totalConnection := 4
	dbchainErr := make(chan error, totalConnection)
	dbconn1Chain := make(chan *mongodatabase.MongoDBConn)
	dbconn2Chain := make(chan *mongodatabase.MongoDBConn)
	dbconn3Chain := make(chan *mongodatabase.MongoDBConn)
	dbconn4Chain := make(chan *mongodatabase.MongoDBConn)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer util.RecoverGoroutinePanic(nil)
		defer wg.Done()
		dbconn1, err := db.New(consts.Note)
		if err != nil {
			dbchainErr <- errors.Wrap(err, "unable to connect to Note")
			return
		}
		dbchainErr <- nil
		dbconn1Chain <- dbconn1
	}(&wg)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer util.RecoverGoroutinePanic(nil)
		defer wg.Done()
		dbconn2, err := db.New(consts.Task)
		if err != nil {
			dbchainErr <- errors.Wrap(err, "unable to connect to Task")
			return
		}

		dbchainErr <- nil
		dbconn2Chain <- dbconn2
	}(&wg)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer util.RecoverGoroutinePanic(nil)
		defer wg.Done()
		dbconn3, err := db.New(consts.File)
		if err != nil {
			dbchainErr <- errors.Wrap(err, "unable to connect to File")
			return
		}

		dbchainErr <- nil
		dbconn3Chain <- dbconn3
	}(&wg)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		defer util.RecoverGoroutinePanic(nil)
		dbconn4, err := db.New(consts.Bookmark)
		if err != nil {
			dbchainErr <- errors.Wrap(err, "unable to connect to Bookmark")
			return
		}

		dbchainErr <- nil
		dbconn4Chain <- dbconn4
	}(&wg)

	for i := 1; i <= totalConnection; i++ {
		errdb := <-dbchainErr
		if errdb != nil {
			return nil, errors.Wrap(errdb, "unable to connect with DB")
		}
	}

	dbconn1 = <-dbconn1Chain
	dbconn2 = <-dbconn2Chain
	dbconn3 = <-dbconn3Chain
	dbconn4 = <-dbconn4Chain

	wg.Wait()

	defer dbconn1.Client.Disconnect(context.TODO())
	defer dbconn2.Client.Disconnect(context.TODO())
	defer dbconn3.Client.Disconnect(context.TODO())
	defer dbconn4.Client.Disconnect(context.TODO())

	colmap["NOTE"] = dbconn1.Collection
	colmap["TASK"] = dbconn2.Collection
	colmap["FILE"] = dbconn3.Collection

	postObjID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return nil, err
	}

	type Data struct {
		ID     primitive.ObjectID `bson:"_id"`
		PostID primitive.ObjectID `bson:"postID"`
		Type   string             `bson:"type"`
		Pos    int                `bson:"pos"`
	}

	filter := bson.M{"postID": postObjID}
	var opts options.FindOptions
	opts.SetLimit(int64(3))
	opts.SetProjection(bson.M{"_id": 1, "postID": 1, "type": 1, "pos": 1})

	var results []Data
	for _, col := range colmap {
		goRoutines++
		go func(col *mongo.Collection, errChan chan<- error) {
			defer util.RecoverGoroutinePanic(nil)
			var d []Data

			cur, err := col.Find(context.TODO(), filter, &opts)
			if err != nil {
				if err.Error() == mongo.ErrNoDocuments.Error() {
					errChan <- nil
					return
				} else {
					errChan <- errors.Wrap(err, "unable to find results")
					return
				}
			}
			defer cur.Close(context.TODO())

			err = cur.All(context.TODO(), &d)
			if err != nil {
				errChan <- errors.Wrap(err, "unable to unpack results")
				return
			}

			if len(d) > 0 {
				results = append(results, d...)
			}

			errChan <- nil
		}(col, errChan)
	}

	for goRoutines != 0 {
		if err := <-errChan; err != nil {
			return nil, errors.Wrap(err, "error from go routine 135")
		}
		goRoutines--
	}

	if len(results) > 0 {
		// sort as per pos
		sort.Slice(results, func(i, j int) bool {
			return results[i].Pos < results[j].Pos
		})
	} else {
		return map[string]interface{}{}, nil
	}

	var ret map[string]interface{}

	filter = bson.M{"_id": results[0].ID}
	switch results[0].Type {
	case "NOTE":
		err = colmap["NOTE"].FindOne(context.TODO(), filter).Decode(&ret)
		if err != nil {
			return nil, err
		}
	case "TASK":
		err = colmap["TASK"].FindOne(context.TODO(), filter).Decode(&ret)
		if err != nil {
			return nil, err
		}
	case consts.FileType:
		// get file from mongo
		err = colmap["FILE"].FindOne(context.TODO(), filter).Decode(&ret)
		if err != nil {
			return nil, err
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

		cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(boardOwnerInt)}
		boardownerInfo, err := profileService.GetConciseProfile(context.TODO(), cpreq)
		if err != nil {
			return nil, errors.Wrap(err, "unable to find basic info")
		}

		key := ""
		if ret["collectionID"].(primitive.ObjectID) == primitive.NilObjectID {
			key = util.GetKeyForBoardPostMedia(int(boardownerInfo.AccountID), int(boardownerInfo.Id), boardID, postID, "")
		} else {
			key = util.GetKeyForPostCollectionMedia(int(boardownerInfo.AccountID), int(boardownerInfo.Id), boardID, postID, ret["collectionID"].(primitive.ObjectID).Hex(), "")
		}

		fileName := fmt.Sprintf("%s%s", ret["_id"].(primitive.ObjectID).Hex(), ret["fileExt"].(string))
		f, err := storageService.GetUserFile(key, fileName)
		if err != nil {
			return nil, errors.Wrap(err, "unable to presign image")
		}
		ret["url"] = f.Filename

		// Fetch Thumbnails
		thumbKey := ""
		if ret["collectionID"].(primitive.ObjectID) == primitive.NilObjectID {
			thumbKey = util.GetKeyForBoardPostMedia(int(boardownerInfo.AccountID), int(boardownerInfo.Id), boardID, postID, "thumbs")
		} else {
			thumbKey = util.GetKeyForPostCollectionMedia(int(boardownerInfo.AccountID), int(boardownerInfo.Id), boardID, postID, ret["collectionID"].(primitive.ObjectID).Hex(), "thumbs")
		}

		thumbfileName := ret["_id"].(primitive.ObjectID).Hex() + ".png"
		thumbs, err := helper.GetThumbnails(storageService, thumbKey, thumbfileName, []string{})
		if err != nil {
			thumbs = model.Thumbnails{}
		}
		ret["thumbs"] = thumbs
	}

	var thingID string

	if id, ok := ret["_id"].(primitive.ObjectID); ok {
		thingID = id.Hex()
	} else if id, ok := ret["_id"].(string); ok {
		thingID = id
	}

	isbookmarked, bid, err := checkProfileBookmark(dbconn4.Collection, thingID, profileID)
	if err != nil {
		ret["isBookmarked"] = false
		ret["bookmarkID"] = ""
	} else {
		ret["isBookmarked"] = isbookmarked
		ret["bookmarkID"] = bid
	}

	return ret, nil
}

func getFirstPostThingV2(db *mongodatabase.DBConfig, boardService board.Service,
	profileService peoplerpc.AccountServiceClient, storageService storage.Service, postID, boardID string, profileID int) (map[string]interface{}, error) {

	var wg sync.WaitGroup
	var dbconn1, dbconn2 *mongodatabase.MongoDBConn
	totalConnection := 2
	dbchainErr := make(chan error, totalConnection)
	dbconn1Chain := make(chan *mongodatabase.MongoDBConn)
	dbconn2Chain := make(chan *mongodatabase.MongoDBConn)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer util.RecoverGoroutinePanic(nil)
		defer wg.Done()
		dbconn1, err := db.New(consts.Post)
		if err != nil {
			dbchainErr <- errors.Wrap(err, "unable to connect to post")
			return
		}
		dbchainErr <- nil
		dbconn1Chain <- dbconn1
	}(&wg)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer util.RecoverGoroutinePanic(nil)
		defer wg.Done()
		dbconn2, err := db.New(consts.Bookmark)
		if err != nil {
			dbchainErr <- errors.Wrap(err, "unable to connect to post")
			return
		}
		dbchainErr <- nil
		dbconn2Chain <- dbconn2
	}(&wg)

	for i := 1; i <= totalConnection; i++ {
		errdb := <-dbchainErr
		if errdb != nil {
			return nil, errors.Wrap(errdb, "unable to connect with DB")
		}
	}

	dbconn1 = <-dbconn1Chain
	dbconn2 = <-dbconn2Chain

	wg.Wait()

	defer dbconn1.Client.Disconnect(context.TODO())
	defer dbconn2.Client.Disconnect(context.TODO())

	postObjID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return nil, err
	}

	boardObjID, err := primitive.ObjectIDFromHex(boardID)
	if err != nil {
		return nil, err
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{
			{Key: "boardID", Value: boardObjID},
			{Key: "_id", Value: postObjID},
		}}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "thTask"},
			{Key: "localField", Value: "_id"},
			{Key: "foreignField", Value: "postID"},
			{Key: "as", Value: "tasks"},
		}}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "thFile"},
			{Key: "localField", Value: "_id"},
			{Key: "foreignField", Value: "postID"},
			{Key: "pipeline", Value: bson.A{
				bson.D{
					{Key: "$match", Value: bson.D{
						{Key: "collectionID", Value: primitive.NilObjectID},
					}},
				},
			}},
			{Key: "as", Value: "files"},
		}}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "thNote"},
			{Key: "localField", Value: "_id"},
			{Key: "foreignField", Value: "postID"},
			{Key: "as", Value: "notes"},
		}}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "thCollection"},
			{Key: "localField", Value: "_id"},
			{Key: "foreignField", Value: "postID"},
			{Key: "as", Value: "collections"},
		}}},
		{{Key: "$addFields", Value: bson.D{
			{Key: "things", Value: bson.D{
				{Key: "$concatArrays", Value: bson.A{
					bson.D{
						{Key: "$ifNull", Value: bson.A{"$tasks", bson.A{}}},
					},
					bson.D{
						{Key: "$ifNull", Value: bson.A{"$files", bson.A{}}},
					},
					bson.D{
						{Key: "$ifNull", Value: bson.A{"$notes", bson.A{}}},
					},
					bson.D{
						{Key: "$ifNull", Value: bson.A{"$collections", bson.A{}}},
					},
				}},
			}},
		}}},
		{{Key: "$unwind", Value: bson.D{
			{Key: "path", Value: "$things"},
		}}},
		{{Key: "$sort", Value: bson.D{
			{Key: "things.pos", Value: 1},
		}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$_id"},
			{Key: "firstThing", Value: bson.D{
				{Key: "$first", Value: "$things"},
			}},
		}}},
	}

	cursor, err := dbconn1.Collection.Aggregate(context.Background(), pipeline)
	if err != nil {
		return nil, errors.Wrap(err, "error executing aggregation pipeline")
	}
	defer cursor.Close(context.Background())
	var result map[string]interface{}
	// Iterate through the results
	for cursor.Next(context.Background()) {

		if err := cursor.Decode(&result); err != nil {
			return nil, errors.Wrap(err, "error Decode")
		}

	}

	if err := cursor.Err(); err != nil {
		return nil, errors.Wrap(err, "error executing cursor pipeline")
	}

	thing, ok := result["firstThing"].(map[string]interface{})
	if ok {
		switch strings.ToUpper(thing["type"].(string)) {
		case consts.FileType:
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

			cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(boardOwnerInt)}
			boardownerInfo, err := profileService.GetConciseProfile(context.TODO(), cpreq)
			if err != nil {
				return nil, errors.Wrap(err, "unable to find basic info")
			}

			key := util.GetKeyForBoardPostMedia(int(boardownerInfo.AccountID), int(boardownerInfo.Id), boardID, postID, "")

			fileName := fmt.Sprintf("%s%s", thing["_id"].(primitive.ObjectID).Hex(), thing["fileExt"].(string))
			f, err := storageService.GetUserFile(key, fileName)
			if err != nil {
				return nil, errors.Wrap(err, "unable to presign image")
			}
			thing["url"] = f.Filename

			// Fetch Thumbnails

			thumbKey := util.GetKeyForBoardPostMedia(int(boardownerInfo.AccountID), int(boardownerInfo.Id), boardID, postID, "thumbs")
			thumbfileName := thing["_id"].(primitive.ObjectID).Hex() + ".png"
			thumbs, err := helper.GetThumbnails(storageService, thumbKey, thumbfileName, []string{})
			if err != nil {
				thumbs = model.Thumbnails{}
			}

			thing["thumbs"] = thumbs

		case "COLLECTION":
			delete(thing, "things")
			isfound := false

			var collectionFile map[string]interface{}
			dbconn3, err := db.New(consts.File)
			if err != nil {
				return nil, errors.Wrap(err, "unable to connect to File")
			}

			err = dbconn3.Collection.FindOne(context.TODO(), bson.M{"collectionID": thing["_id"].(primitive.ObjectID), "pos": 0}).Decode(&collectionFile)
			if err == nil {
				isfound = true
			} else if errors.Is(err, mongo.ErrNoDocuments) {
				err = dbconn3.Collection.FindOne(context.TODO(), bson.M{"collectionID": thing["_id"].(primitive.ObjectID), "pos": 1}).Decode(&collectionFile)
				if err == nil {
					isfound = true
				}
			}

			if isfound {
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

				cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(boardOwnerInt)}
				boardownerInfo, err := profileService.GetConciseProfile(context.TODO(), cpreq)
				if err != nil {
					return nil, errors.Wrap(err, "unable to find basic info")
				}

				key := util.GetKeyForPostCollectionMedia(int(boardownerInfo.AccountID), int(boardownerInfo.Id), boardID, postID, thing["_id"].(primitive.ObjectID).Hex(), "")

				fileName := fmt.Sprintf("%s%s", collectionFile["_id"].(primitive.ObjectID).Hex(), collectionFile["fileExt"].(string))
				f, err := storageService.GetUserFile(key, fileName)
				if err != nil {
					return nil, errors.Wrap(err, "unable to presign image")
				}
				collectionFile["url"] = f.Filename

				// Fetch Thumbnails

				thumbKey := util.GetKeyForPostCollectionMedia(int(boardownerInfo.AccountID), int(boardownerInfo.Id), boardID, postID, thing["_id"].(primitive.ObjectID).Hex(), "thumbs")
				thumbfileName := collectionFile["_id"].(primitive.ObjectID).Hex() + ".png"
				thumbs, err := helper.GetThumbnails(storageService, thumbKey, thumbfileName, []string{})
				if err != nil {
					thumbs = model.Thumbnails{}
				}

				collectionFile["thumbs"] = thumbs
				thing["things"] = collectionFile
			}

		}

		var thingID string

		if id, ok := thing["_id"].(primitive.ObjectID); ok {
			thingID = id.Hex()
		} else if id, ok := thing["_id"].(string); ok {
			thingID = id
		}

		isbookmarked, bid, err := checkProfileBookmark(dbconn2.Collection, thingID, profileID)
		if err != nil {
			thing["isBookmarked"] = false
			thing["bookmarkID"] = ""
		} else {
			thing["isBookmarked"] = isbookmarked
			thing["bookmarkID"] = bid
		}

		return thing, nil
	}

	return nil, nil
}

func getThumbnailAndImageforPostThing(db *mongodatabase.DBConfig, boardService board.Service,
	profileService peoplerpc.AccountServiceClient, storageService storage.Service, postID, boardID string, profileID int, reqThing map[string]interface{}) (map[string]interface{}, error) {
	var wg sync.WaitGroup
	var dbconn1 *mongodatabase.MongoDBConn
	totalConnection := 1
	dbchainErr := make(chan error, totalConnection)
	dbconn1Chain := make(chan *mongodatabase.MongoDBConn)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer util.RecoverGoroutinePanic(nil)
		defer wg.Done()
		dbconn2, err := db.New(consts.Bookmark)
		if err != nil {
			dbchainErr <- errors.Wrap(err, "unable to connect to Bookmark")
			return
		}
		dbchainErr <- nil
		dbconn1Chain <- dbconn2
	}(&wg)

	for i := 1; i <= totalConnection; i++ {
		errdb := <-dbchainErr
		if errdb != nil {
			return nil, errors.Wrap(errdb, "unable to connect with DB")
		}
	}

	dbconn1 = <-dbconn1Chain

	wg.Wait()

	defer dbconn1.Client.Disconnect(context.TODO())

	switch strings.ToUpper(reqThing["type"].(string)) {
	case consts.FileType:
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

		cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(boardOwnerInt)}
		boardownerInfo, err := profileService.GetConciseProfile(context.TODO(), cpreq)
		if err != nil {
			return nil, errors.Wrap(err, "unable to find basic info")
		}

		key := util.GetKeyForBoardPostMedia(int(boardownerInfo.AccountID), int(boardownerInfo.Id), boardID, postID, "")

		fileName := fmt.Sprintf("%s%s", reqThing["_id"].(primitive.ObjectID).Hex(), reqThing["fileExt"].(string))
		f, err := storageService.GetUserFile(key, fileName)
		if err != nil {
			return nil, errors.Wrap(err, "unable to presign image")
		}
		reqThing["url"] = f.Filename

		// Fetch Thumbnaill
		thumbKey := util.GetKeyForBoardPostMedia(int(boardownerInfo.AccountID), int(boardownerInfo.Id), boardID, postID, "thumbs")
		thumbfileName := reqThing["_id"].(primitive.ObjectID).Hex() + ".png"
		thumbs, err := helper.GetThumbnails(storageService, thumbKey, thumbfileName, []string{})
		if err != nil {
			thumbs = model.Thumbnails{}
		}

		reqThing["thumbs"] = thumbs

	case "COLLECTION":
		delete(reqThing, "things")

		isfound := false

		var collectionFile map[string]interface{}
		dbconn3, err := db.New(consts.File)
		if err != nil {
			return nil, errors.Wrap(err, "unable to connect to File")
		}

		defer dbconn3.Client.Disconnect(context.TODO())

		err = dbconn3.Collection.FindOne(context.TODO(), bson.M{"collectionID": reqThing["_id"].(primitive.ObjectID), "pos": 0}).Decode(&collectionFile)
		if err == nil {
			isfound = true
		} else if errors.Is(err, mongo.ErrNoDocuments) {
			err = dbconn3.Collection.FindOne(context.TODO(), bson.M{"collectionID": reqThing["_id"].(primitive.ObjectID), "pos": 1}).Decode(&collectionFile)
			if err == nil {
				isfound = true
			}
		}

		if isfound {
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

			cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(boardOwnerInt)}
			boardownerInfo, err := profileService.GetConciseProfile(context.TODO(), cpreq)
			if err != nil {
				return nil, errors.Wrap(err, "unable to find basic info")
			}

			key := util.GetKeyForPostCollectionMedia(int(boardownerInfo.AccountID), int(boardownerInfo.Id), boardID, postID, reqThing["_id"].(primitive.ObjectID).Hex(), "")

			fileName := fmt.Sprintf("%s%s", collectionFile["_id"].(primitive.ObjectID).Hex(), collectionFile["fileExt"].(string))
			f, err := storageService.GetUserFile(key, fileName)
			if err != nil {
				return nil, errors.Wrap(err, "unable to presign image")
			}
			collectionFile["url"] = f.Filename

			// Fetch Thumbnails
			thumbKey := util.GetKeyForPostCollectionMedia(int(boardownerInfo.AccountID), int(boardownerInfo.Id), boardID, postID, reqThing["_id"].(primitive.ObjectID).Hex(), "thumbs")
			thumbfileName := collectionFile["_id"].(primitive.ObjectID).Hex() + ".png"
			thumbs, err := helper.GetThumbnails(storageService, thumbKey, thumbfileName, []string{})
			if err != nil {
				thumbs = model.Thumbnails{}
			}

			collectionFile["thumbs"] = thumbs
			reqThing["things"] = collectionFile
		}

	}

	var thingID string

	if id, ok := reqThing["_id"].(primitive.ObjectID); ok {
		thingID = id.Hex()
	} else if id, ok := reqThing["_id"].(string); ok {
		thingID = id
	}

	isbookmarked, bid, err := checkProfileBookmark(dbconn1.Collection, thingID, profileID)
	if err != nil {
		reqThing["isBookmarked"] = false
		reqThing["bookmarkID"] = ""
	} else {
		reqThing["isBookmarked"] = isbookmarked
		reqThing["bookmarkID"] = bid
	}

	if reqThing["likes"] != nil {
		reqThing["totalLikes"] = len(reqThing["likes"].(primitive.A))
		var likes []string
		for _, value := range reqThing["likes"].(primitive.A) {
			likes = append(likes, value.(string))
		}
		if util.Contains(likes, fmt.Sprint(profileID)) {
			reqThing["isLiked"] = true
		} else {
			reqThing["isLiked"] = false
		}
	} else {
		reqThing["totalLikes"] = 0
	}

	if reqThing["comments"] != nil {
		reqThing["totalComments"] = len(reqThing["comments"].(primitive.A))
	} else {
		reqThing["totalComments"] = 0
	}

	fmt.Println("return from here ***********************")
	return reqThing, nil
}

func getPostsOfBoard(db *mongodatabase.DBConfig, profileServie peoplerpc.AccountServiceClient, cache *cache.Cache, boardService board.Service, profileID int, boardID, limit, page, filterBy, sortBy string) (map[string]interface{}, error) {
	var boardObj *model.Board
	boardInfo, err := boardService.FetchBoardInfo(boardID, "admins", "viewers", "authors", "subscribers", "blocked", "guests", "followers", "isDefaultBoard")
	if err != nil {
		return nil, errors.Wrap(err, "unable to find board")
	}

	jsonBody, err := json.Marshal(boardInfo)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(jsonBody, &boardObj)
	if err != nil {
		return nil, err
	}

	profileKey := fmt.Sprintf("boards:%s", strconv.Itoa(profileID))
	ownerBoardPermission := permissions.GetBoardPermissionsNew(profileKey, cache, boardObj, strconv.Itoa(profileID))
	role := ownerBoardPermission[boardID]

	fmt.Println("role", role)

	postConn, err := db.New(consts.Post)
	if err != nil {
		return nil, err
	}

	postColl, postClient := postConn.Collection, postConn.Client
	defer postClient.Disconnect(context.TODO())

	// convert string to ObjectID
	boardObjID, err := primitive.ObjectIDFromHex(boardID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to ObjectID")
	}

	// post count
	total, err := postColl.CountDocuments(context.TODO(), bson.M{"boardID": boardObjID})
	if err != nil {
		return nil, errors.Wrap(err, "unable to find posts count")
	}

	var sortKey string
	sortFilter := bson.M{}
	filter := bson.M{"boardID": boardObjID}
	msg := "No%sposts found"
	// sort
	if sortBy != "" {
		switch sortBy {
		case "new":
			sortKey = "createDate"
			sortFilter[sortKey] = -1
		case "old":
			sortKey = "createDate"
			sortFilter[sortKey] = 1
		case "lmd":
			sortKey = "lastModifiedDate"
			sortFilter[sortKey] = -1
		case "alp_asc":
			sortKey = "title"
			sortFilter[sortKey] = 1
		case "alp_dsc":
			sortKey = "title"
			sortFilter[sortKey] = -1
		}
	} else { // default
		sortKey = "createDate"
		sortFilter[sortKey] = -1
	}

	// filter
	if filterBy == "" || filterBy == "all" {
		filter["state"] = bson.M{"$in": []string{consts.Active, consts.Archive, consts.Hidden, consts.Draft}}
		// if boardObj.IsDefaultBoard {
		// 	filter["state"] = bson.M{"$in": []string{consts.Active, consts.Archive, consts.Hidden, consts.Draft}}
		// } else {
		// 	filter["state"] = consts.Active
		// }
	} else if filterBy == "actv" {
		filter["state"] = consts.Active
	} else if filterBy == "arch" {
		filter["state"] = consts.Archive
	} else if filterBy == "hidn" {
		if role == "owner" || role == "admin" {
			filter["state"] = consts.Hidden
		} else if role == "author" {
			filter["owner"] = strconv.Itoa(profileID)
			filter["state"] = consts.Hidden
		} else { // not a member
			filter["state"] = consts.Hidden
			return util.SetPaginationResponse([]model.Post{}, int(total),
					0,
					fmt.Sprintf(msg, cases.Title(language.English).String(filter["state"].(string))),
				),
				nil
		}
	} else if filterBy == "drft" {
		filter["owner"] = strconv.Itoa(profileID)
		filter["state"] = consts.Draft
	}

	// pagination
	findOptions := options.Find()
	findOptions.SetSort(sortFilter)
	pgInt, _ := strconv.Atoi(page)
	limitInt, _ := strconv.Atoi(limit)
	offset := (pgInt - 1) * limitInt
	findOptions.SetSkip(int64(offset))
	findOptions.SetLimit(int64(limitInt))

	var posts []model.Post
	cur, err := postColl.Find(context.TODO(), filter, findOptions)
	if err != nil {
		return nil, errors.Wrap(err, "posts not found")
	}

	err = cur.All(context.TODO(), &posts)
	if err != nil {
		return nil, errors.Wrap(err, "unable to unpack posts")
	}

	if len(posts) == 0 {
		state := " "
		if reflect.TypeOf(filter["state"]).Name() == "string" {
			state = " " + cases.Title(language.English).String((filter["state"].(string))) + " "
		}
		return util.SetPaginationResponse(posts, int(total),
			0,
			fmt.Sprintf(msg, state),
		), nil
	}
	return util.SetPaginationResponse(posts, int(total), 1, "Posts fetched successfully"), nil
}

func getPostsOfBoardV2(db *mongodatabase.DBConfig, profileServie peoplerpc.AccountServiceClient, cache *cache.Cache, boardService board.Service, profileID int, boardID, limit, page, filterBy, sortBy string) (map[string]interface{}, error) {
	var boardObj *model.Board
	boardInfo, err := boardService.FetchBoardInfo(boardID, "admins", "viewers", "authors", "subscribers", "blocked", "guests", "followers", "isDefaultBoard")
	if err != nil {
		return nil, errors.Wrap(err, "unable to find board")
	}

	jsonBody, err := json.Marshal(boardInfo)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(jsonBody, &boardObj)
	if err != nil {
		return nil, err
	}

	profileKey := fmt.Sprintf("boards:%s", strconv.Itoa(profileID))
	ownerBoardPermission := permissions.GetBoardPermissionsNew(profileKey, cache, boardObj, strconv.Itoa(profileID))
	role := ownerBoardPermission[boardID]

	fmt.Println("role", role)

	postConn, err := db.New(consts.Post)
	if err != nil {
		return nil, err
	}

	postColl, postClient := postConn.Collection, postConn.Client
	defer postClient.Disconnect(context.TODO())

	// convert string to ObjectID
	boardObjID, err := primitive.ObjectIDFromHex(boardID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to ObjectID")
	}

	// post count
	total, err := postColl.CountDocuments(context.TODO(), bson.M{"boardID": boardObjID})
	if err != nil {
		return nil, errors.Wrap(err, "unable to find posts count")
	}

	var sortKey string
	sortFilter := bson.M{}
	filter := bson.M{}
	msg := "No%sposts found"
	// sort
	if sortBy != "" {
		switch sortBy {
		case "new":
			sortKey = "createDate"
			sortFilter[sortKey] = -1
		case "old":
			sortKey = "createDate"
			sortFilter[sortKey] = 1
		case "lmd":
			sortKey = "lastModifiedDate"
			sortFilter[sortKey] = -1
		case "alp_asc":
			sortKey = "title"
			sortFilter[sortKey] = 1
		case "alp_dsc":
			sortKey = "title"
			sortFilter[sortKey] = -1
		}
	} else { // default
		sortKey = "createDate"
		sortFilter[sortKey] = -1
	}

	// filter
	if filterBy == "" || filterBy == "all" {
		filter["state"] = bson.M{"$in": []string{consts.Active, consts.Archive, consts.Hidden, consts.Draft}}
	} else if filterBy == "actv" {
		filter["state"] = consts.Active
	} else if filterBy == "arch" {
		filter["state"] = consts.Archive
	} else if filterBy == "hidn" {
		if role == "owner" || role == "admin" {
			filter["state"] = consts.Hidden
		} else if role == "author" {
			filter["owner"] = strconv.Itoa(profileID)
			filter["state"] = consts.Hidden
		} else { // not a member
			filter["state"] = consts.Hidden
			return util.SetPaginationResponse([]model.Post{}, int(total),
					0,
					fmt.Sprintf(msg, cases.Title(language.English).String(filter["state"].(string))),
				),
				nil
		}
	} else if filterBy == "drft" {
		filter["owner"] = strconv.Itoa(profileID)
		filter["state"] = consts.Draft
	}

	// pagination
	findOptions := options.Find()
	findOptions.SetSort(sortFilter)
	pgInt, _ := strconv.Atoi(page)
	limitInt, _ := strconv.Atoi(limit)
	offset := (pgInt - 1) * limitInt
	findOptions.SetSkip(int64(offset))
	findOptions.SetLimit(int64(limitInt))

	// Aggregation pipeline stages
	pipeline := mongo.Pipeline{
		{
			{Key: "$match", Value: bson.M{"boardID": boardObjID}},
		},
		{
			{Key: "$match", Value: filter},
		},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "thTask"},
			{Key: "localField", Value: "_id"},
			{Key: "foreignField", Value: "postID"},
			{Key: "as", Value: "tasks"},
		}}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "thFile"},
			{Key: "localField", Value: "_id"},
			{Key: "foreignField", Value: "postID"},
			{Key: "pipeline", Value: bson.A{
				bson.D{
					{Key: "$match", Value: bson.D{
						{Key: "collectionID", Value: primitive.NilObjectID},
					}},
				},
			}},
			{Key: "as", Value: "files"},
		}}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "thNote"},
			{Key: "localField", Value: "_id"},
			{Key: "foreignField", Value: "postID"},
			{Key: "as", Value: "notes"},
		}}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "thCollection"},
			{Key: "localField", Value: "_id"},
			{Key: "foreignField", Value: "postID"},
			{Key: "as", Value: "collections"},
		}}},
		{{Key: "$addFields", Value: bson.D{
			{Key: "things", Value: bson.D{
				{Key: "$concatArrays", Value: bson.A{
					bson.D{
						{Key: "$ifNull", Value: bson.A{"$tasks", bson.A{}}},
					},
					bson.D{
						{Key: "$ifNull", Value: bson.A{"$files", bson.A{}}},
					},
					bson.D{
						{Key: "$ifNull", Value: bson.A{"$notes", bson.A{}}},
					},
					bson.D{
						{Key: "$ifNull", Value: bson.A{"$collections", bson.A{}}},
					},
				}},
			}},
		}}},
		{{Key: "$unwind", Value: bson.D{
			{Key: "path", Value: "$things"},
			{Key: "preserveNullAndEmptyArrays", Value: true},
		}}},
		{{Key: "$sort", Value: bson.D{
			{Key: "things.pos", Value: 1},
		}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$_id"},
			{Key: "boardID", Value: bson.D{
				{Key: "$first", Value: "$boardID"},
			}},
			{Key: "thingOptSettings", Value: bson.D{
				{Key: "$first", Value: "$thingOptSettings"},
			}},
			{Key: "tags", Value: bson.D{
				{Key: "$first", Value: "$tags"},
			}},
			{Key: "title", Value: bson.D{
				{Key: "$first", Value: "$title"},
			}},
			{Key: "description", Value: bson.D{
				{Key: "$first", Value: "$description"},
			}},
			{Key: "state", Value: bson.D{
				{Key: "$first", Value: "$state"},
			}},
			{Key: "owner", Value: bson.D{
				{Key: "$first", Value: "$owner"},
			}},
			{Key: "priority", Value: bson.D{
				{Key: "$first", Value: "$priority"},
			}},
			{Key: "postStartDate", Value: bson.D{
				{Key: "$first", Value: "$postStartDate"},
			}},
			{Key: "postEndDate", Value: bson.D{
				{Key: "$first", Value: "$postEndDate"},
			}},
			{Key: "lockedDate", Value: bson.D{
				{Key: "$first", Value: "$lockedDate"},
			}},
			{Key: "createDate", Value: bson.D{
				{Key: "$first", Value: "$createDate"},
			}},
			{Key: "modifiedDate", Value: bson.D{
				{Key: "$first", Value: "$modifiedDate"},
			}},
			{Key: "sortDate", Value: bson.D{
				{Key: "$first", Value: "$sortDate"},
			}},
			{Key: "deleteDate", Value: bson.D{
				{Key: "$first", Value: "$deleteDate"},
			}},
			{Key: "sequence", Value: bson.D{
				{Key: "$first", Value: "$sequence"},
			}},
			{Key: "publicStartDate", Value: bson.D{
				{Key: "$first", Value: "$publicStartDate"},
			}},
			{Key: "publicEndDate", Value: bson.D{
				{Key: "$first", Value: "$publicEndDate"},
			}},
			{Key: "shareable", Value: bson.D{
				{Key: "$first", Value: "$shareable"},
			}},
			{Key: "searchable", Value: bson.D{
				{Key: "$first", Value: "$searchable"},
			}},
			{Key: "bookmark", Value: bson.D{
				{Key: "$first", Value: "$bookmark"},
			}},
			{Key: "visible", Value: bson.D{
				{Key: "$first", Value: "$visible"},
			}},
			{Key: "reactions", Value: bson.D{
				{Key: "$first", Value: "$reactions"},
			}},
			{Key: "comments", Value: bson.D{
				{Key: "$first", Value: "$comments"},
			}},
			{Key: "likes", Value: bson.D{
				{Key: "$first", Value: "$likes"},
			}},
			{Key: "viewCount", Value: bson.D{
				{Key: "$first", Value: "$viewCount"},
			}},
			{Key: "type", Value: bson.D{
				{Key: "$first", Value: "$type"},
			}},
			{Key: "totalComments", Value: bson.D{
				{Key: "$first", Value: "$totalComments"},
			}},
			{Key: "totallikes", Value: bson.D{
				{Key: "$first", Value: "$totallikes"},
			}},
			{Key: "isliked", Value: bson.D{
				{Key: "$first", Value: "$isliked"},
			}},
			{Key: "isReactions", Value: bson.D{
				{Key: "$first", Value: "$isReactions"},
			}},
			{Key: "ownerInfo", Value: bson.D{
				{Key: "$first", Value: "$ownerInfo"},
			}},
			{Key: "taggedPeople", Value: bson.D{
				{Key: "$first", Value: "$taggedPeople"},
			}},
			{Key: "isBookmarked", Value: bson.D{
				{Key: "$first", Value: "$isBookmarked"},
			}},
			{Key: "bookmarkID", Value: bson.D{
				{Key: "$first", Value: "$bookmarkID"},
			}},
			{Key: "location", Value: bson.D{
				{Key: "$first", Value: "$location"},
			}},
			{Key: "fileExt", Value: bson.D{
				{Key: "$first", Value: "$fileExt"},
			}},
			{Key: "hidden", Value: bson.D{
				{Key: "$first", Value: "$hidden"},
			}},
			{Key: "isCoverImage", Value: bson.D{
				{Key: "$first", Value: "$isCoverImage"},
			}},
			{Key: "things", Value: bson.D{
				{Key: "$first", Value: "$things"},
			}},
		}}},
		{
			{Key: "$sort", Value: sortFilter},
		},
		{
			{Key: "$skip", Value: int64(offset)},
		},
		{
			{Key: "$limit", Value: int64(limitInt)},
		},
	}

	// Aggregate using the constructed pipeline
	cur, err := postColl.Aggregate(context.TODO(), pipeline)
	if err != nil {
		return nil, errors.Wrap(err, "error executing aggregation pipeline")
	}
	defer cur.Close(context.TODO())

	result := make([]map[string]interface{}, 0)
	// Iterate through the results
	err = cur.All(context.TODO(), &result)
	if err != nil {
		return nil, errors.Wrap(err, "unable to unpack posts")
	}

	if err := cur.Err(); err != nil {
		return nil, errors.Wrap(err, "error executing cursor pipeline")
	}

	if len(result) == 0 {
		state := " "
		if reflect.TypeOf(filter["state"]).Name() == "string" {
			state = " " + cases.Title(language.English).String((filter["state"].(string))) + " "
		}
		return util.SetPaginationResponse(result, int(total),
			0,
			fmt.Sprintf(msg, state),
		), nil
	}
	return util.SetPaginationResponse(result, int(total), 1, "Posts fetched successfully"), nil
}

func findPost(db *mongodatabase.DBConfig, boardID, postID string) (map[string]interface{}, error) {
	dbconn, err := db.New(consts.Board)
	if err != nil {
		return nil, err
	}

	boardCollection, boardClient := dbconn.Collection, dbconn.Client
	defer boardClient.Disconnect(context.TODO())

	postConn, err := db.New(consts.Post)
	if err != nil {
		return nil, err
	}

	postColl, postClient := postConn.Collection, postConn.Client
	defer postClient.Disconnect(context.TODO())

	// convert string to ObjectID
	boardObjID, err := primitive.ObjectIDFromHex(boardID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to ObjectID")
	}
	postObjID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to ObjectID")
	}

	// check if board exists
	c, err := boardCollection.CountDocuments(context.TODO(), bson.M{"_id": boardObjID})
	if err != nil {
		return nil, errors.Wrap(err, "board not found")
	}
	if int(c) == 0 {
		fmt.Println("here")
		return nil, nil
	}

	// find post
	var post model.Post
	err = postColl.FindOne(context.TODO(), bson.M{"_id": postObjID, "boardID": boardObjID}).Decode(&post)
	if err != nil {
		return nil, errors.Wrap(err, "post not found")
	}

	return util.SetResponse(post, 1, "Post things fetched successfully"), nil
}

func findPostByPostID(db *mongodatabase.DBConfig, postID string) (map[string]interface{}, error) {
	postConn, err := db.New(consts.Post)
	if err != nil {
		return nil, err
	}

	postColl, postClient := postConn.Collection, postConn.Client
	defer postClient.Disconnect(context.TODO())

	postObjID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to ObjectID")
	}

	// find post
	var post model.Post
	err = postColl.FindOne(context.TODO(), bson.M{"_id": postObjID}).Decode(&post)
	if err != nil {
		return nil, errors.Wrap(err, "post not found")
	}

	return util.SetResponse(post, 1, "Post fetched successfully"), nil
}

func addPost(db *mongodatabase.DBConfig, cache *cache.Cache, profileService peoplerpc.AccountServiceClient, storageService storage.Service, profileID int, boardID string, post model.Post) (map[string]interface{}, error) {
	dbconn, err := db.New(consts.Board)
	if err != nil {
		return nil, err
	}
	boardCollection, boardClient := dbconn.Collection, dbconn.Client
	defer boardClient.Disconnect(context.TODO())

	// check the profileID's permissions
	profileIDStr := strconv.Itoa(profileID)
	isValid, err := permissions.CheckValidPermissions(profileIDStr, cache, boardCollection, boardID, []string{consts.Owner, consts.Admin, consts.Author}, false)
	if err != nil {
		return nil, err
	}
	if !isValid {
		return util.SetResponse(nil, 0, "User does not have access to the board."), nil
	}

	dbconn, err = db.New(consts.Post)
	if err != nil {
		return nil, err
	}
	postColl, postClient := dbconn.Collection, dbconn.Client
	defer postClient.Disconnect(context.TODO())

	boardObjID, err := primitive.ObjectIDFromHex(boardID)
	if err != nil {
		return nil, err
	}

	post.Id = primitive.NewObjectID()
	post.BoardID = boardObjID
	post.CreateDate = time.Now()
	post.ModifiedDate = time.Now()
	post.Owner = profileIDStr
	post.Type = cases.Upper(language.English).String(consts.Post)
	if post.State == "" {
		post.State = strings.ToUpper(consts.Active)
	}

	// cp, err := profileService.FetchConciseProfile(profileID)

	cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(profileID)}
	cp, err := profileService.GetConciseProfile(context.TODO(), cpreq)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find basic info")
	}

	_, err = postColl.InsertOne(context.TODO(), post)
	if err != nil {
		return nil, errors.Wrap(err, "unable to add Post")
	}

	post.OwnerInfo = cp
	post.OwnerInfo.Id = cp.Id

	return util.SetResponse(post, 1, "post inserted succesfully"), nil
}

func getPostThings(db *mongodatabase.DBConfig, cache *cache.Cache, profileService peoplerpc.AccountServiceClient, noteService note.Service, taskService task.Service, fileService file.Service, boardID, postID string, profileID int) ([]map[string]interface{}, error) {
	var err error
	goroutines := 0
	errChan := make(chan error)
	var notesRes, tasksRes, filesRes, colletionRes []map[string]interface{}

	dbconn, err := db.New(consts.Bookmark)
	if err != nil {
		return nil, errors.Wrap(err, "unable to connect to Bookmark")
	}

	defer dbconn.Client.Disconnect(context.TODO())

	// fetch things
	goroutines += 1
	go func(errChan chan<- error) {
		notesRes, err = noteService.FetchNotesByPost(boardID, postID)
		if err != nil {
			errChan <- errors.Wrap(err, "unable to find post")
		}
		errChan <- nil
	}(errChan)

	// fetch collection
	goroutines += 1
	go func(errChan chan<- error) {
		colletionRes, err = noteService.FetchCollectionByPost(boardID, postID, profileID)
		if err != nil {
			errChan <- errors.Wrap(err, "unable to find post")
		}
		errChan <- nil
	}(errChan)

	goroutines += 1
	go func(errChan chan<- error) {
		tasksRes, err = taskService.FetchTasksByPost(boardID, postID)
		if err != nil {
			errChan <- errors.Wrap(err, "unable to find post")
		}
		errChan <- nil
	}(errChan)

	goroutines += 1
	go func(errChan chan<- error) {
		filesRes, err = fileService.FetchFilesByPost2(boardID, postID)
		if err != nil {
			errChan <- errors.Wrap(err, "unable to find post")
		}
		errChan <- nil
	}(errChan)

	// waiting for goroutines to finish
	for goroutines != 0 {
		goroutines--
		if err := <-errChan; err != nil {
			return nil, errors.Wrap(err, "error from go routine")
		}
	}

	// sort them as per "pos" value
	var allThings []map[string]interface{}
	allThings = append(allThings, notesRes...)
	allThings = append(allThings, tasksRes...)
	allThings = append(allThings, filesRes...)
	allThings = append(allThings, colletionRes...)

	for index, thing := range allThings {
		if posvalue, ok := allThings[index]["pos"]; ok {
			strposvalue := fmt.Sprint(posvalue)
			if posvalue != "" {
				allThings[index]["pos"], _ = strconv.Atoi(strposvalue)
			}
		} else {
			allThings[index]["pos"], _ = strconv.Atoi("3")
		}
		_, ok := allThings[index]["editBy"]
		if ok {
			if edibystr := allThings[index]["editBy"].(string); ok {
				if edibystr != "" {
					edibyint, err := strconv.Atoi(edibystr)
					if err != nil {
						allThings[index]["editByInfo"] = ""
						allThings[index]["editBy"] = ""
					} else {
						// editorinfo, _ := profileService.FetchConciseProfile(edibyint)

						cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(edibyint)}
						editorinfo, err := profileService.GetConciseProfile(context.TODO(), cpreq)
						if err != nil {
							return nil, errors.Wrap(err, "unable to find basic info")
						}

						allThings[index]["editByInfo"] = editorinfo
					}
				} else {
					allThings[index]["editByInfo"] = ""
				}

			} else {
				allThings[index]["editByInfo"] = ""
			}

		} else {
			allThings[index]["editByInfo"] = ""
		}

		var thingID string

		if id, ok := allThings[index]["_id"].(primitive.ObjectID); ok {
			thingID = id.Hex()
		} else if id, ok := allThings[index]["_id"].(string); ok {
			thingID = id
		}

		isbookmarked, bid, err := checkProfileBookmark(dbconn.Collection, thingID, profileID)
		if err != nil {
			allThings[index]["isBookmarked"] = false
			allThings[index]["bookmarkID"] = ""
		} else {
			allThings[index]["isBookmarked"] = isbookmarked
			allThings[index]["bookmarkID"] = bid
		}

		_, islike := thing["likes"]
		if islike && thing["likes"] != nil {
			thing["totalLikes"] = len(thing["likes"].(primitive.A))

			var likes []string
			for _, value := range thing["likes"].(primitive.A) {
				likes = append(likes, value.(string))
			}
			if util.Contains(likes, fmt.Sprint(profileID)) {
				thing["isLiked"] = true
			} else {
				thing["isLiked"] = false
			}
		}

		_, isComment := thing["comments"]
		if isComment && thing["comments"] != nil {
			thing["totalComments"] = len(thing["comments"].(primitive.A))
		}

		allThings[index] = thing
	}

	sort.Slice(allThings, func(i, j int) bool {
		return allThings[i]["pos"].(int) < allThings[j]["pos"].(int)
	})

	return allThings, nil
}

func checkProfileBookmark(bmColl *mongo.Collection, thingID string, profileID int) (bool, string, error) {
	var bm model.Bookmark
	err := bmColl.FindOne(context.TODO(), bson.M{"thingID": thingID, "profileID": profileID}).Decode(&bm)
	if err != nil {
		return false, "", nil
	}
	return true, bm.ID.Hex(), nil
}

func deletePost(db *mongodatabase.DBConfig, postID string) error {
	dbconn, err := db.New(consts.Post)
	if err != nil {
		return err
	}
	postColl, postClient := dbconn.Collection, dbconn.Client
	defer postClient.Disconnect(context.TODO())

	postObjID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return errors.Wrap(err, "unable to convert string to ObjectID")
	}

	_, err = postColl.DeleteOne(context.TODO(), bson.M{"_id": postObjID})
	if err != nil {
		return err
	}

	return nil
}

func movePost(db *mongodatabase.DBConfig, post model.Post, postID, trgtBoard string, storageService storage.Service, boardService board.Service, profileService peoplerpc.AccountServiceClient) error {
	postObjID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return errors.Wrap(err, "unable to convert string to ObjectID")
	}

	oldBoardID := post.BoardID

	trgtBoardObj, err := primitive.ObjectIDFromHex(trgtBoard)
	if err != nil {
		return errors.Wrap(err, "unable to convert string to ObjectID")
	}

	post.BoardID = trgtBoardObj

	dbconn, err := db.New(consts.Post)
	if err != nil {
		return err
	}
	postColl, postClient := dbconn.Collection, dbconn.Client
	defer postClient.Disconnect(context.TODO())

	dbconn1, err := db.New(consts.File)
	if err != nil {
		return err
	}
	fileColl, fileClient := dbconn1.Collection, dbconn1.Client
	defer fileClient.Disconnect(context.TODO())

	var uploadedfiles []model.UploadedFile

	totalFile, err := fileColl.CountDocuments(context.TODO(), bson.M{"postID": postObjID})
	if err != nil {
		return errors.Wrap(err, "Unable to get count of post files.")
	}

	if totalFile > 0 {
		cursor, err := fileColl.Find(context.TODO(), bson.M{"postID": postObjID})
		if err != nil {
			return errors.Wrap(err, "unable to find files")
		}

		err = cursor.All(context.TODO(), &uploadedfiles)
		if err != nil {
			return errors.Wrap(err, "unable to decode files")
		}

		// get the old board owner
		oldboardInfo, err := boardService.FetchBoardInfo(oldBoardID.Hex())
		if err != nil {
			return errors.Wrap(err, "unable to find board")
		}
		oldboardOwnerInt, err := strconv.Atoi(oldboardInfo["owner"].(string))
		if err != nil {
			return err
		}
		// oldboardownerInfo, err := profileService.FetchConciseProfile(oldboardOwnerInt)
		// if err != nil {
		// 	return err
		// }

		cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(oldboardOwnerInt)}
		oldboardownerInfo, err := profileService.GetConciseProfile(context.TODO(), cpreq)
		if err != nil {
			return errors.Wrap(err, "unable to find basic info")
		}

		// get the new board owner
		newboardInfo, err := boardService.FetchBoardInfo(trgtBoard)
		if err != nil {
			return errors.Wrap(err, "unable to find board")
		}
		newboardOwnerInt, err := strconv.Atoi(newboardInfo["owner"].(string))
		if err != nil {
			return err
		}
		// newboardownerInfo, err := profileService.FetchConciseProfile(newboardOwnerInt)
		// if err != nil {
		// 	return err
		// }

		cpreq = &peoplerpc.ConciseProfileRequest{ProfileId: int32(newboardOwnerInt)}
		newboardownerInfo, err := profileService.GetConciseProfile(context.TODO(), cpreq)
		if err != nil {
			return errors.Wrap(err, "unable to find basic info")
		}

		oldkey := util.MovePostKey(int(oldboardownerInfo.AccountID), int(oldboardownerInfo.Id), oldBoardID.Hex())
		newkey := util.MovePostKey(int(newboardownerInfo.AccountID), int(newboardownerInfo.Id), trgtBoard)

		err = storageService.MoveFile(oldkey, newkey)
		if err != nil {
			fmt.Println("****************** orginal file can not move *********")
			return err
		}

		updatePayload := bson.M{"boardID": trgtBoardObj}
		_, err = fileColl.UpdateMany(context.TODO(), bson.M{"postID": postObjID}, bson.M{"$set": updatePayload})
		if err != nil {
			return errors.Wrap(err, "unable to move post")
		}
	}

	updatePayloadforPost := bson.M{"boardID": trgtBoardObj}
	_, err = postColl.UpdateOne(context.TODO(), bson.M{"_id": postObjID}, bson.M{"$set": updatePayloadforPost})
	if err != nil {
		return errors.Wrap(err, "unable to move post")
	}

	return nil
}

func updatePostSettings(db *mongodatabase.DBConfig, cache *cache.Cache, profileID int, postID string, post model.Post, payload map[string]interface{}) (map[string]interface{}, error) {
	dbconn, err := db.New(consts.Board)
	if err != nil {
		return nil, err
	}
	bColl, bc := dbconn.Collection, dbconn.Client
	defer bc.Disconnect(context.TODO())

	isValid, err := permissions.CheckValidPermissions(strconv.Itoa(profileID), cache, bColl, post.BoardID.Hex(), []string{consts.Owner, consts.Admin, consts.Author}, true)
	if err != nil {
		return nil, err
	}

	if !isValid || strconv.Itoa(profileID) != post.Owner {
		return util.SetResponse(nil, 0, "You do not have the permission to update the settings."), nil
	}

	dbconn, err = db.New(consts.Post)
	if err != nil {
		return nil, err
	}
	postColl, postClient := dbconn.Collection, dbconn.Client
	defer postClient.Disconnect(context.TODO())

	postObjID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to ObjectID")
	}
	// get post by id
	var postInfo model.Post
	err = postColl.FindOne(context.TODO(), bson.M{"_id": postObjID}).Decode(&postInfo)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find post")
	}

	if val, ok := payload["state"]; ok {
		if postInfo.Hidden && val.(string) != consts.Hidden { // unhide
			payload["hidden"] = false
		}
		if !postInfo.Hidden && val.(string) == consts.Hidden { // hide
			payload["hidden"] = true
		}
	}

	_, err = postColl.UpdateByID(context.TODO(), postObjID, bson.M{"$set": payload})
	if err != nil {
		return nil, errors.Wrap(err, "unable to update post settings")
	}

	// get the updated post
	err = postColl.FindOne(context.TODO(), bson.M{"_id": postObjID}).Decode(&postInfo)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find post")
	}

	return util.SetResponse(postInfo, 1, "Post settings updated successfully"), nil
}

func updatePostThingUnblocked(db *mongodatabase.DBConfig, postObjId primitive.ObjectID, profileID string) error {
	var wg sync.WaitGroup
	var err error
	var dbconn, dbconn1, dbconn2, dbconn3 *mongodatabase.MongoDBConn
	dbchainErr := make(chan error, 3)
	dbconnChain := make(chan *mongodatabase.MongoDBConn)
	dbconn1Chain := make(chan *mongodatabase.MongoDBConn)
	dbconn2Chain := make(chan *mongodatabase.MongoDBConn)
	dbconn3Chain := make(chan *mongodatabase.MongoDBConn)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		defer util.RecoverGoroutinePanic(nil)
		dbconn, err := db.New(consts.Collection)
		if err != nil {
			dbchainErr <- errors.Wrap(err, "unable to connect to Collection")
			return
		}
		dbchainErr <- nil
		dbconnChain <- dbconn
	}(&wg)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		defer util.RecoverGoroutinePanic(nil)
		dbconn1, err := db.New(consts.Task)
		if err != nil {
			dbchainErr <- errors.Wrap(err, "unable to connect to Task")
			return
		}

		dbchainErr <- nil
		dbconn1Chain <- dbconn1
	}(&wg)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		defer util.RecoverGoroutinePanic(nil)
		dbconn2, err := db.New(consts.Note)
		if err != nil {
			dbchainErr <- errors.Wrap(err, "unable to connect to Note")
			return
		}
		dbchainErr <- nil
		dbconn2Chain <- dbconn2
	}(&wg)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		defer util.RecoverGoroutinePanic(nil)
		dbconn3, err := db.New(consts.File)
		if err != nil {
			dbchainErr <- errors.Wrap(err, "unable to connect to File")
			return
		}
		dbchainErr <- nil
		dbconn3Chain <- dbconn3
	}(&wg)

	for i := 1; i <= 4; i++ {
		errdb := <-dbchainErr
		if errdb != nil {
			return errors.Wrap(errdb, "unable to connect with DB")
		}
	}

	dbconn = <-dbconnChain
	dbconn1 = <-dbconn1Chain
	dbconn2 = <-dbconn2Chain
	dbconn3 = <-dbconn3Chain

	wg.Wait()

	collecitoncoll, colclient := dbconn.Collection, dbconn.Client
	defer colclient.Disconnect(context.TODO())

	taskColl, taskClient := dbconn1.Collection, dbconn1.Client
	defer taskClient.Disconnect(context.TODO())

	noteColl, noteClient := dbconn2.Collection, dbconn2.Client
	defer noteClient.Disconnect(context.TODO())

	fileColl, fileClient := dbconn3.Collection, dbconn3.Client
	defer fileClient.Disconnect(context.TODO())

	_, err = collecitoncoll.UpdateMany(context.TODO(), bson.M{"postID": postObjId, "editBy": profileID}, bson.M{"$set": bson.M{"editBy": "", "editDate": nil}})
	if err != nil {
		return err
	}

	_, err = taskColl.UpdateMany(context.TODO(), bson.M{"postID": postObjId, "editBy": profileID}, bson.M{"$set": bson.M{"editBy": "", "editDate": nil}})
	if err != nil {
		return err
	}

	_, err = noteColl.UpdateMany(context.TODO(), bson.M{"postID": postObjId, "editBy": profileID}, bson.M{"$set": bson.M{"editBy": "", "editDate": nil}})
	if err != nil {
		return err
	}

	_, err = fileColl.UpdateMany(context.TODO(), bson.M{"postID": postObjId, "editBy": profileID}, bson.M{"$set": bson.M{"editBy": "", "editDate": nil}})
	if err != nil {
		return err
	}

	return nil
}

func updatePostThing(db *mongodatabase.DBConfig, reqThings []map[string]interface{}, postObjId primitive.ObjectID, profileID string) error {
	var wg sync.WaitGroup
	var dbconn, dbconn1, dbconn2 *mongodatabase.MongoDBConn
	dbchainErr := make(chan error, 3)
	dbconnChain := make(chan *mongodatabase.MongoDBConn)
	dbconn1Chain := make(chan *mongodatabase.MongoDBConn)
	dbconn2Chain := make(chan *mongodatabase.MongoDBConn)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		defer util.RecoverGoroutinePanic(nil)
		dbconn, err := db.New(consts.Collection)
		if err != nil {
			dbchainErr <- errors.Wrap(err, "unable to connect to Collection")
			return
		}
		dbchainErr <- nil
		dbconnChain <- dbconn
	}(&wg)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		defer util.RecoverGoroutinePanic(nil)
		dbconn1, err := db.New(consts.Task)
		if err != nil {
			dbchainErr <- errors.Wrap(err, "unable to connect to Task")
			return
		}

		dbchainErr <- nil
		dbconn1Chain <- dbconn1
	}(&wg)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		defer util.RecoverGoroutinePanic(nil)
		dbconn2, err := db.New(consts.Note)
		if err != nil {
			dbchainErr <- errors.Wrap(err, "unable to connect to Note")
			return
		}
		dbchainErr <- nil
		dbconn2Chain <- dbconn2
	}(&wg)

	for i := 1; i <= 3; i++ {
		errdb := <-dbchainErr
		if errdb != nil {
			return errors.Wrap(errdb, "unable to connect with DB")
		}
	}

	dbconn = <-dbconnChain
	dbconn1 = <-dbconn1Chain
	dbconn2 = <-dbconn2Chain

	wg.Wait()

	collecitoncoll, colclient := dbconn.Collection, dbconn.Client
	defer colclient.Disconnect(context.TODO())

	taskColl, taskClient := dbconn1.Collection, dbconn1.Client
	defer taskClient.Disconnect(context.TODO())

	noteColl, noteClient := dbconn2.Collection, dbconn2.Client
	defer noteClient.Disconnect(context.TODO())

	for _, thing := range reqThings {
		thingTypeValue, ok1 := thing["type"]
		if !ok1 {
			return errors.New("type not found in objects")
		}

		thingTypeStr := thingTypeValue.(string)
		if thingTypeStr == "" {
			return errors.New("type can not be empty objects")
		}

		var thingObjectId primitive.ObjectID
		var err error
		var isInsert bool

		thingIdValue, ok1 := thing["_id"]
		if !ok1 {
			isInsert = true
			thingObjectId = primitive.NewObjectID()
		} else {
			thingIdStr := thingIdValue.(string)
			if thingIdStr == "" {
				return errors.New("_id can not be empty objects")
			}

			thingObjectId, err = primitive.ObjectIDFromHex(thingIdStr)
			if err != nil {
				return err
			}
		}

		switch strings.ToUpper(thingTypeStr) {
		case "NOTE":
			if isInsert {
				err = ToInsertThing(noteColl, thingObjectId, postObjId, thing)
				if err != nil {
					return errors.Wrap(err, "Unable to insert NOTE with Id: "+thingObjectId.Hex())
				}
			} else {
				err = ToUpdateThing(noteColl, thingObjectId, postObjId, thing, profileID)
				if err != nil {
					return errors.Wrap(err, "Unable to update NOTE with Id: "+thingObjectId.Hex())
				}
			}

		case "TASK":
			if isInsert {
				err = ToInsertThing(taskColl, thingObjectId, postObjId, thing)
				if err != nil {
					return errors.Wrap(err, "Unable to insert TASK with Id: "+thingObjectId.Hex())
				}
			} else {
				err = ToUpdateThing(taskColl, thingObjectId, postObjId, thing, profileID)
				if err != nil {
					return errors.Wrap(err, "Unable to update TASK with Id: "+thingObjectId.Hex())
				}
			}

		case "COLLECTION":
			if isInsert {
				err = ToInsertThing(collecitoncoll, thingObjectId, postObjId, thing)
				if err != nil {
					return errors.Wrap(err, "Unable to insert COLLECTION with Id: "+thingObjectId.Hex())
				}
			} else {
				err = ToUpdateThing(collecitoncoll, thingObjectId, postObjId, thing, profileID)
				if err != nil {
					return errors.Wrap(err, "Unable to update COLLECTION with Id: "+thingObjectId.Hex())
				}
			}

		default:
			return errors.New("type can be NOTE,TASK, COLLECTION")
		}
	}

	return nil
}

func deleteSelectedPostThing(db *mongodatabase.DBConfig, reqThings []map[string]interface{}, postObjId primitive.ObjectID) error {
	var wg sync.WaitGroup
	totalConnection := 6

	var dbconn, dbconn1, dbconn2, dbconn3, dbconn4, dbconn5 *mongodatabase.MongoDBConn
	dbchainErr := make(chan error, totalConnection)
	dbconnChain := make(chan *mongodatabase.MongoDBConn)
	dbconn1Chain := make(chan *mongodatabase.MongoDBConn)
	dbconn2Chain := make(chan *mongodatabase.MongoDBConn)
	dbconn3Chain := make(chan *mongodatabase.MongoDBConn)
	dbconn4Chain := make(chan *mongodatabase.MongoDBConn)
	dbconn5Chain := make(chan *mongodatabase.MongoDBConn)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		defer util.RecoverGoroutinePanic(nil)
		dbconn, err := db.New(consts.Collection)
		if err != nil {
			dbchainErr <- errors.Wrap(err, "unable to connect to Collection")
			return
		}
		dbchainErr <- nil
		dbconnChain <- dbconn
	}(&wg)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		defer util.RecoverGoroutinePanic(nil)
		dbconn1, err := db.New(consts.Task)
		if err != nil {
			dbchainErr <- errors.Wrap(err, "unable to connect to Task")
			return
		}

		dbchainErr <- nil
		dbconn1Chain <- dbconn1
	}(&wg)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		defer util.RecoverGoroutinePanic(nil)
		dbconn2, err := db.New(consts.Note)
		if err != nil {
			dbchainErr <- errors.Wrap(err, "unable to connect to Note")
			return
		}
		dbchainErr <- nil
		dbconn2Chain <- dbconn2
	}(&wg)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		defer util.RecoverGoroutinePanic(nil)
		dbconn3, err := db.New(consts.File)
		if err != nil {
			dbchainErr <- errors.Wrap(err, "unable to connect to File")
			return
		}
		dbchainErr <- nil
		dbconn3Chain <- dbconn3
	}(&wg)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		defer util.RecoverGoroutinePanic(nil)
		dbconn4, err := db.New(consts.Bookmark)
		if err != nil {
			dbchainErr <- errors.Wrap(err, "unable to connect to Bookmark")
			return
		}
		dbchainErr <- nil
		dbconn4Chain <- dbconn4
	}(&wg)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		defer util.RecoverGoroutinePanic(nil)
		dbconn5, err := db.New(consts.Recent)
		if err != nil {
			dbchainErr <- errors.Wrap(err, "unable to connect to Recent")
			return
		}
		dbchainErr <- nil
		dbconn5Chain <- dbconn5
	}(&wg)

	for i := 1; i <= totalConnection; i++ {
		errdb := <-dbchainErr
		if errdb != nil {
			return errors.Wrap(errdb, "unable to connect with DB")
		}
	}

	dbconn = <-dbconnChain
	dbconn1 = <-dbconn1Chain
	dbconn2 = <-dbconn2Chain
	dbconn3 = <-dbconn3Chain
	dbconn4 = <-dbconn4Chain
	dbconn5 = <-dbconn5Chain

	wg.Wait()

	collecitoncoll, colclient := dbconn.Collection, dbconn.Client
	defer colclient.Disconnect(context.TODO())

	taskColl, taskClient := dbconn1.Collection, dbconn1.Client
	defer taskClient.Disconnect(context.TODO())

	noteColl, noteClient := dbconn2.Collection, dbconn2.Client
	defer noteClient.Disconnect(context.TODO())

	fileColl, fileClient := dbconn3.Collection, dbconn3.Client
	defer fileClient.Disconnect(context.TODO())

	bookmarkColl, bookmarkClient := dbconn4.Collection, dbconn4.Client
	defer bookmarkClient.Disconnect(context.TODO())

	recentColl, recentClient := dbconn5.Collection, dbconn5.Client
	defer recentClient.Disconnect(context.TODO())

	for _, thing := range reqThings {
		thingTypeValue, ok1 := thing["type"]
		if !ok1 {
			return errors.New("type not found in objects")
		}

		thingTypeStr := thingTypeValue.(string)
		if thingTypeStr == "" {
			return errors.New("type can not be empty objects")
		}

		thingIdValue, ok1 := thing["_id"]
		if !ok1 {
			return errors.New("_id can not be found in request.")
		}

		thingIdStr := thingIdValue.(string)
		if thingIdStr == "" {
			return errors.New("_id can not be empty objects")
		}

		thingObjectId, err := primitive.ObjectIDFromHex(thingIdStr)
		if err != nil {
			return err
		}

		switch strings.ToUpper(thingTypeStr) {
		case "NOTE":
			err = ToDeleteThing(noteColl, thingObjectId, postObjId)
			if err != nil {
				return errors.Wrap(err, "Unable to delete NOTE with Id: "+thingObjectId.Hex())
			}
		case "TASK":

			err = ToDeleteThing(taskColl, thingObjectId, postObjId)
			if err != nil {
				return errors.Wrap(err, "Unable to delete TASK with Id: "+thingObjectId.Hex())
			}

		case "COLLECTION":

			err = ToDeleteThing(collecitoncoll, thingObjectId, postObjId)
			if err != nil {
				return errors.Wrap(err, "Unable to delete COLLECTION with Id: "+thingObjectId.Hex())
			}

			err = ToDeleteCollectionFiles(fileColl, thingObjectId, postObjId)
			if err != nil {
				return errors.Wrap(err, "Unable to delete COLLECTION with Id: "+thingObjectId.Hex())
			}

		case "FILE":

			err = ToDeleteThing(fileColl, thingObjectId, postObjId)
			if err != nil {
				return errors.Wrap(err, "Unable to delete FILE with Id: "+thingObjectId.Hex())
			}

		default:
			return errors.New("type can be NOTE,TASK, COLLECTION")
		}

		err = flagBookmarkAndRecentDelete(bookmarkColl, recentColl, thingObjectId, time.Now())
		if err != nil {
			return errors.Wrap(err, "Unable update to bookmark and recent with Id: "+thingObjectId.Hex())
		}
	}

	return nil
}

func ToUpdateThing(col *mongo.Collection, thingObjectId primitive.ObjectID, postID primitive.ObjectID, payload map[string]interface{}, profileID string) error {
	delete(payload, "_id")
	payload["postID"] = postID

	var temp map[string]interface{}
	err := col.FindOne(context.TODO(), bson.M{"_id": thingObjectId}).Decode(&temp)
	if err != nil {
		return err
	}

	_, ok := temp["editBy"]
	if ok {
		if temp["editBy"].(string) == profileID {
			payload["editBy"] = ""
			payload["editDate"] = nil
		}
	}

	_, err = col.UpdateOne(context.TODO(), bson.M{"_id": thingObjectId}, bson.M{"$set": payload})
	if err != nil {
		return err
	}

	return nil
}

func ToInsertThing(col *mongo.Collection, thingObjectId, postID primitive.ObjectID, payload map[string]interface{}) error {
	payload["postID"] = postID
	payload["_id"] = thingObjectId
	payload["comments"] = nil
	payload["likes"] = nil
	payload["state"] = consts.Active
	payload["createDate"] = time.Now()
	payload["editBy"] = ""
	payload["editDate"] = nil

	_, err := col.InsertOne(context.TODO(), payload)
	if err != nil {
		return err
	}

	return nil
}

func ToDeleteThing(col *mongo.Collection, thingObjectId primitive.ObjectID, postID primitive.ObjectID) error {
	_, err := col.DeleteOne(context.TODO(), bson.M{"_id": thingObjectId, "postID": postID})
	if err != nil {
		return err
	}

	return nil
}

func ToDeleteCollectionFiles(col *mongo.Collection, collectionID primitive.ObjectID, postID primitive.ObjectID) error {
	total, err := col.CountDocuments(context.TODO(), bson.M{"collectionID": collectionID, "postID": postID})
	if err != nil {
		return err
	}

	if total > 0 {
		_, err := col.DeleteMany(context.TODO(), bson.M{"collectionID": collectionID, "postID": postID})
		if err != nil {
			return err
		}
	}

	return nil
}

func flagBookmarkAndRecentDelete(bookmarkColl, recentColl *mongo.Collection, thingID primitive.ObjectID, deleteDate time.Time) error {
	filter := bson.M{"thingID": thingID.Hex()}

	// count
	count, err := bookmarkColl.CountDocuments(context.TODO(), filter)
	if err != nil {
		return errors.Wrap(err, "unable to find bookmark's count")
	}

	if int(count) > 0 {
		update := bson.M{
			"$set": bson.M{
				"deleteDate": deleteDate,
				"flagged":    true,
			},
		}

		// Perform the update
		updateResult, err := bookmarkColl.UpdateMany(context.TODO(), filter, update)
		if err != nil {
			return errors.Wrap(err, "Unable to update bookmark documents")
		}

		fmt.Printf("Updated %v bookmark documents\n", updateResult.ModifiedCount)
	}

	recentfilter := bson.M{"thingID": thingID}

	// count
	recentcount, err := recentColl.CountDocuments(context.TODO(), recentfilter)
	if err != nil {
		return errors.Wrap(err, "unable to find recent's count")
	}

	if int(recentcount) > 0 {

		// Perform the update
		updateResult, err := recentColl.DeleteMany(context.TODO(), recentfilter)
		if err != nil {
			return errors.Wrap(err, "Unable to delete recent documents")
		}

		fmt.Printf("Deleted %v recent documents\n", updateResult.DeletedCount)
	}

	return nil
}
