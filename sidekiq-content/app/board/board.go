package board

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/profile"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/storage"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/cache"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/consts"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/database"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/helper"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/mongodatabase"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/permissions"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/util"
	model "github.com/TestingSDK2/sidekiq-backend/sidekiq-models"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/pkg/errors"

	// peoplerpc "github.com/sidekiq-people/proto/people"
	peoplerpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func paginate(arr []model.BoardMemberRole, pageNo, limit int) (ret []model.BoardMemberRole) {
	var startIdx, endIdx int
	startIdx = limit * (pageNo - 1)
	endIdx = limit * pageNo

	if len(arr) == limit || len(arr) < limit {
		return arr
	}
	if endIdx < len(arr) {
		ret = arr[startIdx:endIdx]
	} else {
		ret = arr[startIdx:]
	}
	return
}

func paginate2(arr []model.ConciseProfile, pageNo, limit int) (ret []model.ConciseProfile) {
	var startIdx, endIdx int
	startIdx = limit * (pageNo - 1)
	endIdx = limit * pageNo

	if len(arr) == limit || len(arr) < limit {
		return arr
	}
	if endIdx < len(arr) {
		ret = arr[startIdx:endIdx]
	} else {
		ret = arr[startIdx:]
	}
	return
}

func getProfileImage(mysql *database.Database, storageService storage.Service, userID, profileID int) (string, error) {
	var err error
	if userID == 0 {
		stmt := `SELECT accountID FROM` + "`sidekiq-dev`.AccountProfile WHERE id = ?"
		err = mysql.Conn.Get(&userID, stmt, profileID)
		if err != nil {
			return "", err
		}
	}

	key := util.GetKeyForProfileImage(userID, profileID, "")
	fileName := fmt.Sprintf("%d.png", profileID)
	fileData, err := storageService.GetUserFile(key, fileName)
	if err != nil {
		return "", err
	}
	return fileData.Filename, nil
}

func getProfileImageThumb(mysql *database.Database, storageService storage.Service, accountID, profileID int) (model.Thumbnails, error) {

	thumbTypes := []string{"sm", "ic"}
	thumbKey := util.GetKeyForProfileImage(accountID, profileID, "thumbs")
	thumbfileName := fmt.Sprintf("%d.png", profileID)
	thumbs, err := helper.GetThumbnails(storageService, thumbKey, thumbfileName, thumbTypes)
	if err != nil {
		thumbs = model.Thumbnails{}
	}

	return thumbs, nil
}

func getBoardsByProfile(db *mongodatabase.DBConfig, mysql *database.Database, cache *cache.Cache, profileID int,
	profileService peoplerpc.AccountServiceClient, storageService storage.Service, fetchSubBoards bool, page, limit string,
) (map[string]interface{}, error) {
	var boards, accessibleBoards []*model.Board

	dbconn, err := db.New(consts.Board)
	if err != nil {
		return nil, err
	}
	collection, client := dbconn.Collection, dbconn.Client
	defer client.Disconnect(context.TODO())

	profileIDStr := strconv.Itoa(profileID)

	filter := bson.M{
		"$or": bson.A{
			bson.M{"owner": profileIDStr},
			bson.M{"subscribers": profileIDStr},
			bson.M{"admins": profileIDStr},
			bson.M{"authors": profileIDStr},
			bson.M{"guests": profileIDStr},
		},
	}

	// get the board count
	total, err := collection.CountDocuments(context.TODO(), filter)
	if err != nil {
		return nil, err
	}

	findOptions := options.Find()
	findOptions.SetSort(bson.M{"createDate": -1})
	pgInt, _ := strconv.Atoi(page)
	limitInt, _ := strconv.Atoi(limit)
	offset := (pgInt - 1) * limitInt
	findOptions.SetSkip(int64(offset))
	findOptions.SetLimit(int64(limitInt))

	cursor, err := collection.Find(context.TODO(), filter, findOptions)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find boards")
	}
	defer cursor.Close(context.TODO())

	err = cursor.All(context.TODO(), &boards)
	if err != nil {
		return nil, err
	}

	ownerKey := fmt.Sprintf("boards:%s", profileIDStr)
	ownerBoardPermission := permissions.GetBoardPermissions(ownerKey, cache)

	for idx := range boards {
		boards[idx].IsBoardFollower = util.Contains(boards[idx].Followers, profileIDStr)
		boards[idx].IsPassword = boards[idx].Password != ""
		// get total board members
		var members []string
		members = append(members, boards[idx].Admins...)
		members = append(members, boards[idx].Authors...)
		members = append(members, boards[idx].Subscribers...)
		members = append(members, boards[idx].Followers...)
		distinctMembers := util.RemoveArrayDuplicate(members)
		if len(members) > 0 {
			for _, member := range distinctMembers {
				memberID, _ := strconv.Atoi(member)
				// info, err := profileService.FetchConciseProfile(memberID)

				cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(memberID)}
				info, err := profileService.GetConciseProfile(context.TODO(), cpreq)
				if err != nil {
					if errors.Is(err, sql.ErrNoRows) {
						continue
					}
					fmt.Println("ERROR COMMING FROM 173", memberID, boards[idx].Id)
					return nil, errors.Wrap(err, "unable in fetching concise profile")
				}

				if len(boards[idx].BoardMembers) < 2 {
					if info != nil {
						boards[idx].BoardMembers = append(boards[idx].BoardMembers, info)
					}
				} else {
					continue
				}
			}
			boards[idx].TotalMembers = len(distinctMembers)

			if util.Contains(boards[idx].Likes, fmt.Sprint(profileID)) {
				boards[idx].IsLiked = true
			} else {
				boards[idx].IsLiked = false
			}

			boards[idx].TotalLikes = len(boards[idx].Likes)
			boards[idx].TotalComments = len(boards[idx].Comments)
		}

		// fetch basic profile info
		boardOwnerIDInt, _ := strconv.Atoi(boards[idx].Owner)
		// cp, err := profileService.FetchConciseProfile(boardOwnerIDInt)

		cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(boardOwnerIDInt)}
		cp, err := profileService.GetConciseProfile(context.TODO(), cpreq)
		if err != nil {
			return nil, err
		}

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				log.Println(err)
				return nil, errors.Wrap(err, "board owner data not present in sql")
			} else {
				return nil, errors.Wrap(err, "error in fetching concise profile")
			}
		}
		cp.Id = int32(boardOwnerIDInt)
		boards[idx].OwnerInfo = cp
		role := ownerBoardPermission[boards[idx].Id.Hex()]
		if role != "owner" {
			boards[idx].Password = ""
		}
		if role != "blocked" {
			if fetchSubBoards { // fetch boards along with sub boards
				accessibleBoards = append(accessibleBoards, boards[idx])
			} else {
				if boards[idx].ParentID == "" {
					accessibleBoards = append(accessibleBoards, boards[idx])
				}
			}
		}
	}

	if len(accessibleBoards) == 0 {
		return util.SetPaginationResponse([]*model.Board{}, int(total), 1, "User has no boards. Please add one."), nil
	}
	return util.SetPaginationResponse(accessibleBoards, int(total), 1, "Boards fetched successfully"), nil
}

func getFollowedBoardsByProfile(db *mongodatabase.DBConfig, mysql *database.Database, cache *cache.Cache, search string, profileID int,
	profileService peoplerpc.AccountServiceClient, storageService storage.Service, limitInt, pgInt int, sortBy, orderBy string,
) (map[string]interface{}, error) {

	var followedboards []*model.Board

	dbconn, err := db.New(consts.Board)
	if err != nil {
		return nil, err
	}
	collection, client := dbconn.Collection, dbconn.Client
	defer client.Disconnect(context.TODO())

	var FollowBoardID []string

	stmt := "SELECT DISTINCT(b.boardID) FROM  `sidekiq-dev`.BoardsFollowed AS b WHERE b.profileID=?;"

	err = mysql.Conn.Select(&FollowBoardID, stmt, profileID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch boards followed")
	}

	if len(FollowBoardID) == 0 {
		return util.SetPaginationResponse([]model.Board{}, 0, 1, "User has no followed boards. Please add one."), nil
	}

	var objectIDs []primitive.ObjectID
	for _, id := range FollowBoardID {
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			message := fmt.Sprintf("Error in getFollowedBoardsByProfile converting %s to ObjectID: %s", id, err)
			return nil, errors.Wrap(err, message)
		}
		objectIDs = append(objectIDs, objectID)
	}

	var filterorderBy int64
	filter := bson.M{"_id": bson.M{"$in": objectIDs}, "state": consts.Active}
	// pagination
	findOptions := options.Find()
	if sortBy == "" {
		sortBy = "createDate"
	}

	if search != "" {
		filter["title"] = bson.M{"$regex": primitive.Regex{Pattern: search, Options: "i"}}
	}

	if orderBy == "" || orderBy == "Desc" || orderBy == "desc" || orderBy == "DESC" {
		filterorderBy = -1
	} else {
		filterorderBy = 1
	}
	findOptions.SetSort(bson.M{sortBy: filterorderBy})
	if pgInt > 0 {
		offset := (pgInt - 1) * limitInt
		findOptions.SetSkip(int64(offset))
		findOptions.SetLimit(int64(limitInt))
	}

	total, err := collection.CountDocuments(context.TODO(), filter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find board count")
	}
	if int(total) == 0 {
		return util.SetPaginationResponse([]model.Board{}, 0, 1, "User has no board found."), nil
	}

	cursor, err := collection.Find(context.TODO(), filter, findOptions)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find boards")
	}
	defer cursor.Close(context.TODO())

	err = cursor.All(context.TODO(), &followedboards)
	if err != nil {
		return nil, err
	}

	if len(followedboards) == 0 {
		return util.SetResponse(nil, 1, "User has no followed boards. Please add one."), nil
	}

	for idx := range followedboards {
		// getting the role
		profileKey := fmt.Sprintf("boards:%s", strconv.Itoa(profileID))
		ownerBoardPermission := permissions.GetBoardPermissionsNew(profileKey, cache, followedboards[idx], strconv.Itoa(profileID))
		followedboards[idx].Role = ownerBoardPermission[followedboards[idx].Id.Hex()]

		followedboards[idx].IsPassword = followedboards[idx].Password != ""
		// get total board members
		var members []string
		members = append(members, followedboards[idx].Admins...)
		members = append(members, followedboards[idx].Authors...)
		members = append(members, followedboards[idx].Subscribers...)
		members = append(members, followedboards[idx].Followers...)
		distinctMembers := util.RemoveArrayDuplicate(members)
		if len(members) > 0 {
			for _, member := range distinctMembers {
				memberID, _ := strconv.Atoi(member)
				// info, err := profileService.FetchConciseProfile(memberID)

				cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(memberID)}
				info, err := profileService.GetConciseProfile(context.TODO(), cpreq)

				if err != nil {
					if errors.Is(err, sql.ErrNoRows) {
						continue
					}
					fmt.Println("ERROR COMMING FROM 173", memberID, followedboards[idx].Id)
					return nil, errors.Wrap(err, "unable in fetching concise profile")
				}
				if len(followedboards[idx].BoardMembers) < 2 {
					if info != nil {
						followedboards[idx].BoardMembers = append(followedboards[idx].BoardMembers, info)
					}
				} else {
					continue
				}
			}
			followedboards[idx].TotalMembers = len(distinctMembers)

			if util.Contains(followedboards[idx].Likes, fmt.Sprint(profileID)) {
				followedboards[idx].IsLiked = true
			} else {
				followedboards[idx].IsLiked = false
			}

			followedboards[idx].TotalLikes = len(followedboards[idx].Likes)
			followedboards[idx].TotalComments = len(followedboards[idx].Comments)

			var members []string
			members = append(members, followedboards[idx].Admins...)
			members = append(members, followedboards[idx].Guests...)
			members = append(members, followedboards[idx].Subscribers...)
			members = append(members, followedboards[idx].Viewers...)

			if util.Contains(members, fmt.Sprint(profileID)) {
				followedboards[idx].IsBoardShared = true
			} else {
				followedboards[idx].IsBoardShared = false
			}
		}

		// fetch basic profile info
		boardOwnerIDInt, _ := strconv.Atoi(followedboards[idx].Owner)
		// cp, err := profileService.FetchConciseProfile(boardOwnerIDInt)

		cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(boardOwnerIDInt)}
		cp, err := profileService.GetConciseProfile(context.TODO(), cpreq)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				log.Println(err)
				return nil, errors.Wrap(err, "board owner data not present in sql")
			} else {
				return nil, errors.Wrap(err, "error in fetching concise profile")
			}
		}

		cp.Id = int32(boardOwnerIDInt)
		followedboards[idx].OwnerInfo = cp
		role := ownerBoardPermission[followedboards[idx].Id.Hex()]
		if role != "owner" {
			followedboards[idx].Password = ""
		}
	}

	if strings.ToLower(sortBy) == "owner" {
		sort.Slice(followedboards, func(i, j int) bool {
			name1 := fmt.Sprintf("%s %s", followedboards[i].OwnerInfo.FirstName, followedboards[i].OwnerInfo.LastName)
			name2 := fmt.Sprintf("%s %s", followedboards[j].OwnerInfo.FirstName, followedboards[j].OwnerInfo.LastName)

			if strings.ToLower(orderBy) == "asc" {
				return strings.ToLower(name1) < strings.ToLower(name2)
			}
			return strings.ToLower(name1) > strings.ToLower(name2)
		})
	}

	return util.SetPaginationResponse(followedboards, int(total), 1, "Followed Boards fetched successfully"), nil
}

func getFirstPostThing(db *mongodatabase.DBConfig, profileService peoplerpc.AccountServiceClient, storageService storage.Service, postID, boardID string, profileID int) (map[string]interface{}, error) {
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
			field := make([]string, 0)
			boardInfo, err := fetchBoardInfo(db, boardID, field)
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
				return nil, errors.Wrap(err, "unable to fetch concise profile")
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
				field := make([]string, 0)
				boardInfo, err := fetchBoardInfo(db, boardID, field)
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
					return nil, errors.Wrap(err, "unable to fetch concise profile")
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

func checkProfileBookmark(bmColl *mongo.Collection, thingID string, profileID int) (bool, string, error) {
	var bm model.Bookmark
	err := bmColl.FindOne(context.TODO(), bson.M{"thingID": thingID, "profileID": profileID}).Decode(&bm)
	if err != nil {
		return false, "", nil
	}
	return true, bm.ID.Hex(), nil
}

func getBoardsAndPostByState(db *mongodatabase.DBConfig, mysql *database.Database, cache *cache.Cache, profileID int,
	profileService peoplerpc.AccountServiceClient, storageService storage.Service, state string, limitInt, pgInt int, sortBy, orderBy string, fetchPost bool, searchKeyword string,
) (map[string]interface{}, error) {

	var boards []*model.Board
	var posts []*model.Post

	dbconn, err := db.New(consts.Board)
	if err != nil {
		return nil, err
	}
	collection, client := dbconn.Collection, dbconn.Client
	defer client.Disconnect(context.TODO())

	profileIDStr := strconv.Itoa(profileID)
	ownerKey := fmt.Sprintf("boards:%s", profileIDStr)
	filter := make(bson.M)
	filter["isDefaultBoard"] = false
	filter["$or"] = bson.A{bson.M{"owner": profileIDStr}}
	if state == consts.Hidden {
		filter["hidden"] = true
	} else {
		filter["state"] = state
	}

	if searchKeyword != "" {
		filter["title"] = bson.M{"$regex": primitive.Regex{Pattern: searchKeyword, Options: "i"}}
	}

	var filterorderBy int64
	cps := make(map[int]*peoplerpc.ConciseProfileReply)
	bpms := make(map[string]model.BoardPermission)

	// pagination
	findOptions := options.Find()
	if sortBy == "" {
		sortBy = "createDate"
	}

	if orderBy == "" || orderBy == "Desc" || orderBy == "desc" || orderBy == "DESC" {
		filterorderBy = -1
	} else {
		filterorderBy = 1
	}

	findOptions.SetSort(bson.M{sortBy: filterorderBy})
	if pgInt > 0 {
		offset := (pgInt - 1) * limitInt
		findOptions.SetSkip(int64(offset))
		findOptions.SetLimit(int64(limitInt))
	}

	cursor, err := collection.Find(context.TODO(), filter, findOptions)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find boards")
	}
	defer cursor.Close(context.TODO())

	err = cursor.All(context.TODO(), &boards)
	if err != nil {
		return nil, err
	}

	var cp *peoplerpc.ConciseProfileReply
	if val, ok := cps[profileID]; !ok {
		// cp, err = profileService.FetchConciseProfile(profileID)
		// if err != nil {
		// 	return nil, errors.Wrap(err, "unable to fetch concise profile")
		// }
		cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(profileID)}
		cp, err := profileService.GetConciseProfile(context.TODO(), cpreq)
		if err != nil {
			return nil, errors.Wrap(err, "unable to fetch concise profile")
		}

		cps[profileID] = cp
	} else {
		cp = val
	}

	var merged []interface{}

	for idx := range boards {
		boards[idx].IsBoardFollower = util.Contains(boards[idx].Followers, strconv.Itoa(profileID))
		boards[idx].IsPassword = boards[idx].Password != ""
		// get total board members
		var members []string
		members = append(members, boards[idx].Admins...)
		members = append(members, boards[idx].Authors...)
		members = append(members, boards[idx].Subscribers...)
		members = append(members, boards[idx].Followers...)
		distinctMembers := util.RemoveArrayDuplicate(members)
		if len(members) > 0 {
			for _, member := range distinctMembers {
				memberID, _ := strconv.Atoi(member)
				// info, err := profileService.FetchConciseProfile(memberID)
				// if err != nil {
				// 	if errors.Is(err, sql.ErrNoRows) {
				// 		continue
				// 	}
				// }

				cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(memberID)}
				info, err := profileService.GetConciseProfile(context.TODO(), cpreq)
				if errors.Is(err, sql.ErrNoRows) {
					continue
				}

				if len(boards[idx].BoardMembers) < 2 {
					if info != nil {
						boards[idx].BoardMembers = append(boards[idx].BoardMembers, info)
					}
				} else {
					continue
				}
			}
			boards[idx].TotalMembers = len(distinctMembers)

			if util.Contains(boards[idx].Likes, fmt.Sprint(profileID)) {
				boards[idx].IsLiked = true
			} else {
				boards[idx].IsLiked = false
			}

			boards[idx].TotalLikes = len(boards[idx].Likes)
			boards[idx].TotalComments = len(boards[idx].Comments)

			var members []string
			members = append(members, boards[idx].Admins...)
			members = append(members, boards[idx].Guests...)
			members = append(members, boards[idx].Subscribers...)
			members = append(members, boards[idx].Viewers...)

			if util.Contains(members, fmt.Sprint(profileID)) {
				boards[idx].IsBoardShared = true
			} else {
				boards[idx].IsBoardShared = false
			}
		}

		cp.Id = int32(profileID)
		boards[idx].OwnerInfo = cp
		var bpm model.BoardPermission
		if v, ok := bpms[boards[idx].Id.Hex()+profileIDStr]; !ok {
			bpm = permissions.GetBoardPermissionsNew(ownerKey, cache, boards[idx], profileIDStr)
			bpms[boards[idx].Id.Hex()+profileIDStr] = bpm
		} else {
			bpm = v
		}
		boards[idx].Role = bpm[boards[idx].Id.Hex()]
		if boards[idx].Role != "owner" {
			boards[idx].Password = ""
		}

		merged = append(merged, boards[idx])
	}

	if fetchPost {
		dbconn1, err := db.New(consts.Post)
		fmt.Println("posts", posts)
		if err != nil {
			return nil, err
		}
		postCollection, client1 := dbconn1.Collection, dbconn1.Client
		defer client1.Disconnect(context.TODO())

		filter := make(bson.M)
		filter["$or"] = bson.A{bson.M{"owner": profileIDStr}}
		if state == consts.Hidden {
			filter["hidden"] = true
		} else {
			filter["state"] = state
		}

		if searchKeyword != "" {
			filter["title"] = bson.M{"$regex": primitive.Regex{Pattern: searchKeyword, Options: "i"}}
		}

		cursor, err = postCollection.Find(context.TODO(), filter, findOptions)
		if err != nil {
			return nil, errors.Wrap(err, "unable to find boards")
		}
		defer cursor.Close(context.TODO())
		err = cursor.All(context.TODO(), &posts)
		if err != nil {
			return nil, err
		}

		for _, post := range posts {
			if util.Contains(post.Likes, fmt.Sprint(profileID)) {
				post.IsLiked = true
			} else {
				post.IsLiked = false
			}

			post.Things, _ = getFirstPostThing(db, profileService, storageService, post.Id.Hex(), post.BoardID.Hex(), profileID)

			post.TotalLikes = len(post.Likes)
			post.TotalComments = len(post.Comments)

			post.OwnerInfo = cp
			merged = append(merged, post)
		}
	}

	if strings.ToLower(sortBy) == "owner" {
		sort.Slice(merged, func(i, j int) bool {
			var name1 string
			var name2 string

			if board, ok := merged[i].(*model.Board); ok {
				name1 = fmt.Sprintf("%s %s", board.OwnerInfo.FirstName, board.OwnerInfo.LastName)
			} else if post, ok := merged[i].(*model.Post); ok {
				name1 = fmt.Sprintf("%s %s", post.OwnerInfo.FirstName, post.OwnerInfo.LastName)
			}

			if board, ok := merged[j].(*model.Board); ok {
				name2 = fmt.Sprintf("%s %s", board.OwnerInfo.FirstName, board.OwnerInfo.LastName)
			} else if post, ok := merged[j].(*model.Post); ok {
				name2 = fmt.Sprintf("%s %s", post.OwnerInfo.FirstName, post.OwnerInfo.LastName)
			}

			if strings.ToLower(orderBy) == "asc" {
				return strings.ToLower(name1) < strings.ToLower(name2)
			}
			return strings.ToLower(name1) > strings.ToLower(name2)
		})
	} else if strings.ToLower(sortBy) == "title" {
		sort.Slice(merged, func(i, j int) bool {
			var title1 string
			var title2 string

			if board, ok := merged[i].(*model.Board); ok {
				title1 = board.Title
			} else if post, ok := merged[i].(*model.Post); ok {
				title1 = post.Title
			}

			if board, ok := merged[j].(*model.Board); ok {
				title2 = board.Title
			} else if post, ok := merged[j].(*model.Post); ok {
				title2 = post.Title
			}

			if strings.ToLower(orderBy) == "asc" {
				return strings.ToLower(title1) < strings.ToLower(title2)
			}
			return strings.ToLower(title1) > strings.ToLower(title2)
		})
	}

	if fetchPost {
		return util.SetPaginationResponse(merged, len(merged), 1, "Boards and posts fetched successfully"), nil
	}

	return util.SetPaginationResponse(merged, len(merged), 1, "Boards fetched successfully"), nil

}

func searchBoardsByProfile(db *mongodatabase.DBConfig, mysql *database.Database, cache *cache.Cache, profileID int,
	profileService profile.Service, storageService storage.Service, boardName string, fetchSubBoards bool, page, limit string,
) (map[string]interface{}, error) {
	var (
		boards []*model.Board
		res    []*model.BoardSearch
	)

	dbconn, err := db.New(consts.Board)
	if err != nil {
		return nil, err
	}
	collection, client := dbconn.Collection, dbconn.Client
	defer client.Disconnect(context.TODO())

	profileIDStr := strconv.Itoa(profileID)

	filter := bson.M{
		"$and": bson.A{
			bson.M{
				"$or": bson.A{
					bson.M{"owner": profileIDStr},
					bson.M{"viewers": profileIDStr},
					bson.M{"subscribers": profileIDStr},
					bson.M{"admins": profileIDStr},
					// bson.M{"parentID": bson.M{"$eq": ""}},
				},
			},
			// bson.M{"title": bson.M{"$regex": boardName, "$options": "im"}},
		},
	}

	findOptions := options.Find()
	findOptions.SetSort(bson.M{"createDate": -1})

	// pagination
	// pgInt, _ := strconv.Atoi(page)
	// limitInt, _ := strconv.Atoi(limit)
	// offset := (pgInt - 1) * limitInt
	// findOptions.SetSkip(int64(offset))
	// findOptions.SetLimit(int64(limitInt))

	cursor, err := collection.Find(context.TODO(), filter, findOptions)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find boards")
	}
	defer cursor.Close(context.TODO())

	err = cursor.All(context.TODO(), &boards)
	if err != nil {
		return nil, err
	}

	for _, b := range boards {
		if fuzzy.Match(boardName, b.Title) || fuzzy.MatchFold(boardName, b.Title) {
			temp := model.BoardSearch{
				Id:         b.Id,
				Title:      b.Title,
				CreateDate: b.CreateDate,
				Tags:       b.Tags,
			}
			res = append(res, &temp)
		}
	}

	// for _, value := range boards {
	// 	temp := model.BoardSearch{
	// 		Id:         value.Id,
	// 		Title:      value.Title,
	// 		CreateDate: value.CreateDate,
	// 		Tags:       value.Tags,
	// 	}
	// 	res = append(res, &temp)
	// }

	return util.SetResponse(res, 1, "Boards fetched successfully"), nil
}

func fetchSubBoardsOfProfile(db *mongodatabase.DBConfig, profileService peoplerpc.AccountServiceClient, storageService storage.Service, profileID int, page, limit string) (map[string]interface{}, error) {
	var err error
	profileIDStr := strconv.Itoa(profileID)

	dbconn, err := db.New(consts.Board)
	if err != nil {
		return nil, err
	}
	boardCollection, boardClient := dbconn.Collection, dbconn.Client
	defer boardClient.Disconnect(context.TODO())

	var curr *mongo.Cursor
	findOptions := options.Find()

	filter := bson.M{
		"$and": bson.A{
			bson.M{"$or": bson.A{
				bson.M{"owner": profileIDStr},
				bson.M{"viewers": profileIDStr},
				bson.M{"subscribers": profileIDStr},
				bson.M{"admins": profileIDStr},
			}},
			bson.M{"parentID": bson.M{"$ne": bson.M{"parentID": ""}}},
		},
	}

	total, err := boardCollection.CountDocuments(context.TODO(), filter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find total count")
	}

	findOptions.SetSort(bson.M{"createDate": -1})

	// pagination
	pgInt, _ := strconv.Atoi(page)
	limitInt, _ := strconv.Atoi(limit)
	offset := (pgInt - 1) * limitInt
	findOptions.SetSkip(int64(offset))
	findOptions.SetLimit(int64(limitInt))

	curr, err = boardCollection.Find(context.TODO(), filter, findOptions)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find sub-boards")
	}

	var boards []*model.Board
	err = curr.All(context.TODO(), &boards)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find sub-boards")
	}

	for idx := range boards {
		if util.Contains(boards[idx].Likes, fmt.Sprint(profileID)) {
			boards[idx].IsLiked = true
		} else {
			boards[idx].IsLiked = false
		}

		boards[idx].TotalLikes = len(boards[idx].Likes)
		boards[idx].TotalComments = len(boards[idx].Comments)
	}

	if len(boards) == 0 {
		return util.SetPaginationResponse([]model.Board{}, 0, 1, "You have no boards shared with you."), nil
	}
	return util.SetPaginationResponse(boards, int(total), 1, "Boards fetched successfully."), nil
}

func fetchSubBoardsOfBoard(db *mongodatabase.DBConfig, profileService peoplerpc.AccountServiceClient, storageService storage.Service, profileID int, boardID, page, limit string) (map[string]interface{}, error) {
	var err error
	dbconn, err := db.New(consts.Board)
	if err != nil {
		return nil, err
	}
	boardCollection, boardClient := dbconn.Collection, dbconn.Client
	defer boardClient.Disconnect(context.TODO())

	// boardObjID, _ := primitive.ObjectIDFromHex(boardID)
	findOptions := options.Find()
	var curr *mongo.Cursor

	filter := bson.M{"parentID": boardID}

	total, err := boardCollection.CountDocuments(context.TODO(), bson.M{"parentID": boardID})
	if err != nil {
		return nil, errors.Wrap(err, "unable to find board count")
	}
	if int(total) == 0 {
		return util.SetPaginationResponse([]model.Board{}, 0, 1, "Board contains no sub-boards"), nil
	}

	findOptions.SetSort(bson.M{"createDate": -1})

	// pagination
	pgInt, _ := strconv.Atoi(page)
	limitInt, _ := strconv.Atoi(limit)
	offset := (pgInt - 1) * limitInt
	findOptions.SetSkip(int64(offset))
	findOptions.SetLimit(int64(limitInt))

	curr, err = boardCollection.Find(context.TODO(), filter, findOptions)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find sub-boards")
	}

	var boards []*model.Board
	err = curr.All(context.TODO(), &boards)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find sub-boards")
	}

	// fetch basic info
	for idx := range boards {
		boards[idx].IsBoardFollower = util.Contains(boards[idx].Followers, strconv.Itoa(profileID))
		idInt, _ := strconv.Atoi(boards[idx].Owner)
		// cp, err := profileService.FetchConciseProfile(idInt)
		// if err != nil {
		// 	return nil, errors.Wrap(err, "unable to find basic info")
		// }

		cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(idInt)}
		cp, err := profileService.GetConciseProfile(context.TODO(), cpreq)
		if err != nil {
			return nil, errors.Wrap(err, "unable to find basic info")
		}

		boards[idx].OwnerInfo = cp

		if util.Contains(boards[idx].Likes, fmt.Sprint(profileID)) {
			boards[idx].IsLiked = true
		} else {
			boards[idx].IsLiked = false
		}

		boards[idx].TotalLikes = len(boards[idx].Likes)
		boards[idx].TotalComments = len(boards[idx].Comments)
	}

	return util.SetPaginationResponse(boards, int(total), 1, "Sub-boards fetched successfully"), nil
}

func fetchSubBoards(cache *cache.Cache, profileService peoplerpc.AccountServiceClient, storageService storage.Service, db *mongodatabase.DBConfig, mysql *database.Database, boardID string, profileID, limit int) (map[string]interface{}, error) {
	dbconn, err := db.New(consts.Board)
	if err != nil {
		return nil, err
	}
	boardCollection, boardClient := dbconn.Collection, dbconn.Client
	defer boardClient.Disconnect(context.TODO())

	profileIDStr := strconv.Itoa(profileID)
	isValid, err := permissions.CheckValidPermissions(profileIDStr, cache, boardCollection, boardID, []string{"blocked"}, true)
	if err != nil {
		return nil, err
	}
	if !isValid {
		return util.SetResponse(nil, 0, "User does not have access to the board."), nil
	}

	var curr *mongo.Cursor
	var opts *options.FindOptions

	findFilter := bson.M{"parentID": boardID}

	findOptions := options.Find()
	findOptions.SetSort(bson.M{"createDate": -1})

	if limit != 0 {
		opts = options.Find().SetLimit(int64(limit))
		curr, err = boardCollection.Find(context.TODO(), findFilter, opts, findOptions)
	} else {
		curr, err = boardCollection.Find(context.TODO(), findFilter, findOptions)
	}

	if err != nil {
		return nil, errors.Wrap(err, "unable to find sub boards")
	}

	var subBoards []*model.Board

	err = curr.All(context.TODO(), &subBoards)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch sub boards")
	}

	errChan := make(chan error)

	var wg sync.WaitGroup
	for index := range subBoards {
		wg.Add(1)
		go func(i int) error {
			defer wg.Done()
			subBoards[i].IsBoardFollower = util.Contains(subBoards[i].Followers, strconv.Itoa(profileID))
			subBoards[i].IsPassword = subBoards[i].Password != ""
			itemOwner, _ := strconv.Atoi(subBoards[i].Owner)
			// ownerInfo, err := profileService.FetchConciseProfile(itemOwner)
			// if err != nil {
			// 	return errors.Wrap(err, "unable to map profile info")
			// }

			cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(itemOwner)}
			ownerInfo, err := profileService.GetConciseProfile(context.TODO(), cpreq)
			if err != nil {
				return errors.Wrap(err, "unable to map profile info")
			}

			subBoards[i].OwnerInfo = ownerInfo

			// add location
			loc, err := fetchThingLocationOnBoard(db, subBoards[i].Id.Hex())
			if err != nil {
				errChan <- errors.Wrap(err, "unable to fetch board location")
			}
			subBoards[i].Location = loc

			if util.Contains(subBoards[i].Likes, fmt.Sprint(profileID)) {
				subBoards[i].IsLiked = true
			} else {
				subBoards[i].IsLiked = false
			}

			subBoards[i].TotalLikes = len(subBoards[i].Likes)
			subBoards[i].TotalComments = len(subBoards[i].Comments)

			errChan <- nil
			return nil
		}(index)
	}
	// wg.Wait()
	for i := 0; i < len(subBoards); i++ {
		if err := <-errChan; err != nil {
			return nil, errors.Wrap(err, "error from go routine from ftsOnDashboard()")
		}
	}

	if len(subBoards) == 0 {
		return util.SetResponse(nil, 1, "No boards are shared with you."), nil
	}

	return util.SetResponse(subBoards, 1, "Boards fetched successfully."), nil
}

func addBoard(cache *cache.Cache, db *mongodatabase.DBConfig, profileService peoplerpc.AccountServiceClient, storageService storage.Service, board model.Board, profileID int) (map[string]interface{}, error) {
	// checking if the user has valid permissions in the parent board
	dbconn, err := db.New(consts.Board)
	if err != nil {
		return nil, err
	}

	collection, client := dbconn.Collection, dbconn.Client
	defer client.Disconnect(context.TODO())

	profileIDStr := strconv.Itoa(profileID)
	if board.ParentID != "" {
		isValid, err := permissions.CheckValidPermissions(profileIDStr, cache, collection, board.ParentID, []string{consts.Owner, consts.Admin, consts.Author}, false)
		if err != nil {
			return nil, err
		}
		if !isValid {
			return util.SetResponse(nil, 0, "User does not have access to the board."), nil
		}
	}

	// fixed
	board.Id = primitive.NewObjectID()
	board.CreateDate = time.Now()
	board.ModifiedDate = time.Now()
	board.Type = cases.Upper(language.English).String(consts.Board)
	if board.Owner == "" {
		board.Owner = strconv.Itoa(profileID)
	}
	if board.State == "" {
		board.State = consts.Active
	}
	if board.Visible == "" {
		board.Visible = consts.Public
	}

	// add owner info
	// cp, err := profileService.FetchConciseProfile(profileID)
	// if err != nil {
	// 	return nil, errors.Wrap(err, "unable to find owner's info.")
	// }

	cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(profileID)}
	cp, err := profileService.GetConciseProfile(context.TODO(), cpreq)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find owner's info.")
	}

	cp.Id = int32(profileID)
	board.OwnerInfo = cp

	_, err = collection.InsertOne(context.TODO(), board)
	if err != nil {
		return nil, errors.Wrap(err, "unable to insert board at mongo")
	}

	// add board things Tags
	dbconn2, err := db.New(consts.BoardThingsTags)
	if err != nil {
		return nil, err
	}

	collection2, client2 := dbconn2.Collection, dbconn2.Client
	defer client2.Disconnect(context.TODO())

	_, err = collection2.InsertOne(context.TODO(), bson.M{"boardID": board.Id, "tags": nil})
	if err != nil {
		return nil, errors.Wrap(err, "unable to add board things document")
	}

	return util.SetResponse(board, 1, "Board added Successfully"), nil
}

func getBoardByID(mysql *database.Database, db *mongodatabase.DBConfig, boardIDStr string, profileService peoplerpc.AccountServiceClient, storageService storage.Service, args ...string) (map[string]interface{}, error) {
	var board *model.Board
	dbconn, err := db.New(consts.Board)
	if err != nil {
		return nil, err
	}
	collection, client := dbconn.Collection, dbconn.Client
	defer client.Disconnect(context.TODO())

	boardID, _ := primitive.ObjectIDFromHex(boardIDStr)
	filter := bson.M{"_id": boardID}
	err = collection.FindOne(context.TODO(), filter).Decode(&board)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.Wrap(err, "board does not exist")
		}
		return nil, errors.Wrap(err, "unable to fetch board")
	}
	if err != nil {
		return nil, err
	}
	board.IsPassword = board.Password != ""
	if args[0] != "owner" {
		board.Password = ""
	}

	// fetch board location
	board.Location, err = fetchThingLocationOnBoard(db, boardIDStr)
	if err != nil {
		log.Println("unable to find board location")
	}

	// fetch basic info
	boardOwnerIDInt, _ := strconv.Atoi(board.Owner)
	// ownerInfo, err := profileService.FetchConciseProfile(boardOwnerIDInt)
	// if err != nil {
	// 	return nil, errors.Wrap(err, "unable to fetch basic info")
	// }

	cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(boardOwnerIDInt)}
	ownerInfo, err := profileService.GetConciseProfile(context.TODO(), cpreq)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find owner's info.")
	}

	board.OwnerInfo = ownerInfo

	// check count in db
	stmt := "SELECT COUNT(DISTINCT p.id) FROM `sidekiq-dev`.AccountProfile as p INNER JOIN `sidekiq-dev`.BoardsFollowed as b on p.id = b.profileID AND b.boardID = ?"
	err = mysql.Conn.Get(&board.TotalFollowers, stmt, board.Id.Hex())
	if err != nil {
		return nil, errors.Wrap(err, "unable to get record's existence")
	}

	// fetch board followers
	var boardFollowersID []int
	stmt = "SELECT p.id FROM `sidekiq-dev`.AccountProfile as p INNER JOIN `sidekiq-dev`.BoardsFollowed as b on p.id = b.profileID AND b.boardID = ? LIMIT 10"
	err = mysql.Conn.Select(&boardFollowersID, stmt, board.Id.Hex())
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch board followers")
	}

	board.TotalTags = len(board.Tags)
	admins := len(board.Admins)
	authors := len(board.Authors)
	subscribers := len(board.Subscribers)
	viewers := len(board.Viewers)

	// fetch admin, author, subscriber and viewers profiles
	errChan := make(chan error)
	totalRoutines := 0
	var mutex sync.Mutex
	if admins > 0 {
		totalRoutines += 1
		go func(errChan chan<- error) {
			defer util.RecoverGoroutinePanic(nil)
			for i := range board.Admins {
				idInt, _ := strconv.Atoi(board.Admins[i])
				// memberProfile, err := profileService.FetchConciseProfile(idInt)
				// if err != nil {
				// 	errChan <- errors.Wrap(err, "unable to find owner's info.")
				// }

				cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(idInt)}
				memberProfile, err := profileService.GetConciseProfile(context.TODO(), cpreq)
				if err != nil {
					errChan <- errors.Wrap(err, "unable to find owner's info.")
				}

				if memberProfile != nil {
					memberProfile.Type = "admin"
					mutex.Lock()
					board.BoardMembers = append(board.BoardMembers, memberProfile)
					mutex.Unlock()
				}
			}
			errChan <- nil
		}(errChan)
	}

	if authors > 0 {
		totalRoutines += 1
		go func(errChan chan<- error) {
			defer util.RecoverGoroutinePanic(nil)
			for i := range board.Authors {
				idInt, _ := strconv.Atoi(board.Authors[i])

				cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(idInt)}
				memberProfile, err := profileService.GetConciseProfile(context.TODO(), cpreq)
				if err != nil {
					errChan <- errors.Wrap(err, "unable to find owner's info.")
				}

				if memberProfile != nil {
					memberProfile.Type = "author"
					mutex.Lock()
					board.BoardMembers = append(board.BoardMembers, memberProfile)
					mutex.Unlock()
				}
			}
			errChan <- nil
		}(errChan)
	}

	if subscribers > 0 {
		totalRoutines += 1
		go func(errChan chan<- error) {
			defer util.RecoverGoroutinePanic(nil)
			for i := range board.Subscribers {
				idInt, _ := strconv.Atoi(board.Subscribers[i])
				// memberProfile, err := profileService.FetchConciseProfile(idInt)

				cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(idInt)}
				memberProfile, err := profileService.GetConciseProfile(context.TODO(), cpreq)
				if err != nil {
					errChan <- errors.Wrap(err, "unable to find owner's info.")
				}

				if memberProfile != nil {

					memberProfile.Type = "subscriber"
					mutex.Lock()
					board.BoardMembers = append(board.BoardMembers, memberProfile)
					mutex.Unlock()
				}
			}
			errChan <- nil
		}(errChan)
	}

	if viewers > 0 {
		totalRoutines += 1
		go func(errChan chan<- error) {
			defer util.RecoverGoroutinePanic(nil)
			for i := range board.Viewers {
				idInt, _ := strconv.Atoi(board.Viewers[i])
				// memberProfile, err := profileService.FetchConciseProfile(idInt)

				cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(idInt)}
				memberProfile, err := profileService.GetConciseProfile(context.TODO(), cpreq)
				if err != nil {
					errChan <- errors.Wrap(err, "unable to find owner's info.")
				}

				if memberProfile != nil {
					memberProfile.Type = "viewer"
					mutex.Lock()
					board.BoardMembers = append(board.BoardMembers, memberProfile)
					mutex.Unlock()
				}
			}
			errChan <- nil
		}(errChan)
	}

	go func(errChan chan<- error) {
		defer util.RecoverGoroutinePanic(nil)
		for i := range boardFollowersID {
			// followersProfile, err := profileService.FetchConciseProfile(boardFollowersID[i])

			cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(boardFollowersID[i])}
			followersProfile, err := profileService.GetConciseProfile(context.TODO(), cpreq)
			if err != nil {
				errChan <- errors.Wrap(err, "unable to find owner's info.")
			}

			mutex.Lock()
			board.BoardFollowers = append(board.BoardFollowers, followersProfile)
			mutex.Unlock()
		}
		errChan <- nil
	}(errChan)

	for i := 0; i < totalRoutines+1; i++ {
		if err := <-errChan; err != nil {
			return nil, errors.Wrap(err, "error from go fetch members profile go routine")
		}
	}
	board.TotalMembers = len(board.BoardMembers)
	if board.TotalMembers > 10 {
		board.BoardMembers = board.BoardMembers[:10]
	}

	board.TotalMembers = board.TotalMembers + 1

	return util.SetResponse(board, 1, "Board fetched successfully."), nil
}

func getBoardDetailsByID(db *mongodatabase.DBConfig, boardIDStr string) (map[string]interface{}, error) {
	var board *model.Board
	dbconn, err := db.New(consts.Board)
	if err != nil {
		return nil, err
	}
	collection, client := dbconn.Collection, dbconn.Client
	defer client.Disconnect(context.TODO())

	boardID, _ := primitive.ObjectIDFromHex(boardIDStr)
	filter := bson.M{"_id": boardID}
	err = collection.FindOne(context.TODO(), filter).Decode(&board)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch board")
	}

	return util.SetResponse(board, 1, "Board fetched successfully."), nil
}

func addViewerInBoard(db *mongodatabase.DBConfig, boardIDStr, profileID string) (map[string]interface{}, error) {
	dbconn, err := db.New(consts.Board)
	if err != nil {
		return nil, err
	}
	collection, client := dbconn.Collection, dbconn.Client
	defer client.Disconnect(context.TODO())

	boardID, _ := primitive.ObjectIDFromHex(boardIDStr)
	filter := bson.M{"_id": boardID}
	update := bson.M{"$addToSet": bson.M{"viewers": profileID}}
	_, err = collection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		return nil, errors.Wrap(err, "unable to add profile in viewer list")
	}

	return util.SetResponse(nil, 1, "profile added to viewer successfully"), nil
}

func fetchThingLocationOnBoard(mongoDb *mongodatabase.DBConfig, boardID string) (location string, err error) {
	err = nil
	location = ""
	var filter primitive.D
	dbconn, err := mongoDb.New(consts.Board)
	if err != nil {
		return "", err
	}
	var info model.Board
	info.ParentID = boardID
	var boardObjID primitive.ObjectID
	boardCollection, client := dbconn.Collection, dbconn.Client
	defer client.Disconnect(context.TODO())

	for info.ParentID != "" {
		boardObjID, err = primitive.ObjectIDFromHex(info.ParentID)
		if err != nil {
			return "", errors.Wrap(err, "unable to map boardID")
		}
		filter = bson.D{{Key: "_id", Value: boardObjID}}
		err = boardCollection.FindOne(context.TODO(), filter).Decode(&info)
		if err != nil {
			return "", errors.Wrap(err, "unable to find board")
		}

		if info.ParentID == "" {
			location += info.Title
		} else {
			location += info.Title + "/"
		}
	}
	return
}

func updateBoard(cache *cache.Cache, db *mongodatabase.DBConfig, payload map[string]interface{}, boardID string, profileID int) (map[string]interface{}, error) {
	profileIDStr := strconv.Itoa(profileID)
	var err error

	dbconn, err := db.New(consts.Board)
	if err != nil {
		return nil, errors.Wrap(err, "unable to connect to Board")
	}

	boardObjID, _ := primitive.ObjectIDFromHex(boardID)
	collection, client := dbconn.Collection, dbconn.Client
	defer client.Disconnect(context.TODO())

	fmt.Println("boardObjID", boardObjID)
	filter := bson.M{"_id": boardObjID}

	var board model.Board
	err = collection.FindOne(context.TODO(), filter).Decode(&board)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find board")
	}
	if board.Owner != profileIDStr {
		return util.SetResponse(nil, 0, "User does not have access to the board."), nil
	}

	payload["modifiedDate"] = time.Now()
	payload["_id"] = boardObjID
	payload["owner"] = board.Owner
	payload["createDate"] = board.CreateDate

	res, err := collection.UpdateOne(context.TODO(), filter, bson.M{"$set": payload})
	if err != nil || res.ModifiedCount == 0 {
		return nil, errors.Wrap(err, "unable to update board at mongo")
	}

	// get the update board
	err = collection.FindOne(context.TODO(), filter).Decode(&board)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find updated Board")
	}

	return util.SetResponse(&board, 1, "Board updated successfully."), nil
}

func deleteBoard(cache *cache.Cache, db *mongodatabase.DBConfig, boardID string, profileID int) (map[string]interface{}, error) {
	// Board collection
	dbconn, err := db.New(consts.Board)
	if err != nil {
		return nil, errors.Wrap(err, "unable to connect to Board")
	}
	boardCollection, boardClient := dbconn.Collection, dbconn.Client
	defer boardClient.Disconnect(context.TODO())

	// Trash collection
	trashConn, err := db.New(consts.Trash)
	if err != nil {
		return nil, errors.Wrap(err, "unable to connect to Trash")
	}
	trashColl, trashClient := trashConn.Collection, trashConn.Client
	defer trashClient.Disconnect(context.TODO())

	profileIDStr := strconv.Itoa(profileID)
	boardObjID, err := primitive.ObjectIDFromHex(boardID)
	if err != nil {
		return nil, err
	}

	filter := bson.M{"$and": bson.A{
		bson.M{"_id": boardID},
		bson.M{"owner": profileIDStr},
	}}

	var prevBoard *model.Board
	err = boardCollection.FindOne(context.TODO(), filter).Decode(&prevBoard)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find board")
	}
	if prevBoard.Owner != profileIDStr {
		return util.SetResponse(nil, 0, "User does not have access to the board."), nil
	}

	// check if the board contains any child board or not
	childBoardsArr, err := getChildBoards(boardCollection, boardObjID)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("unable to find child board of %s", boardObjID.Hex()))
	}
	childBoards := childBoardsArr[0]
	if len(childBoards["childBoards"].(primitive.A)) != 0 {
		return util.SetResponse(nil, 0, "Cannot delete board, contains another Board."), nil
	}

	// trashing and deleting all the things
	for _, col := range []string{consts.Note, consts.Task, consts.File} {
		conn, err := db.New(col)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("unable to connect to to %s", col))
		}
		coll, client := conn.Collection, conn.Client
		defer client.Disconnect(context.TODO())

		filter := bson.M{"boardID": boardObjID}

		// trashing
		var docs []interface{}
		curr, err := coll.Find(context.TODO(), filter)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("unable to find documents from %s", col))
		}
		curr.All(context.TODO(), &docs)
		for _, doc := range docs {
			m := doc.(map[string]interface{})
			m["deleteDate"] = time.Now()
			m["expiryDate"] = m["deleteDate"].(time.Time).AddDate(0, 0, 30)
		}
		_, err = trashColl.InsertMany(context.TODO(), docs)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("unable to insert into Trash from to %s", col))
		}

		// deleting
		_, err = coll.DeleteMany(context.TODO(), filter)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("unable to delete things from %s", col))
		}
	}

	// trash the board
	prevBoard.Id = boardObjID
	prevBoard.DeleteDate = time.Now()
	prevBoard.ExpiryDate = prevBoard.DeleteDate.AddDate(0, 0, 30)

	_, err = trashColl.InsertOne(context.TODO(), prevBoard)
	if err != nil {
		return nil, errors.Wrap(err, "unable to trash the board")
	}

	// delete the board
	_, err = boardCollection.DeleteOne(context.TODO(), bson.M{"_id": boardID})
	if err != nil {
		return nil, errors.Wrap(err, "unable to delete the board")
	}

	return util.SetResponse(prevBoard, 1, "Board and all its things trashed successfully"), nil
}

func getChildBoards(coll *mongo.Collection, boardObjID primitive.ObjectID) ([]map[string]interface{}, error) {
	var childBoards []map[string]interface{}
	var err error

	match := bson.M{"$match": bson.M{"_id": boardObjID}}
	graphLookup := bson.M{"$graphLookup": bson.M{
		"from":             "Board",
		"startWith":        "$boardID",
		"connectFromField": "boardID",
		"connectToField":   "parentID",
		"as":               "childBoards",
	}}
	boardsCursor, bmerror := coll.Aggregate(context.TODO(), bson.A{match, graphLookup})
	if bmerror != nil {
		return nil, err
	}
	defer boardsCursor.Close(context.TODO())

	err = boardsCursor.All(context.TODO(), &childBoards)
	if err != nil {
		return nil, err
	}
	return childBoards, nil
}

func findBoardMappings(db *mongodatabase.DBConfig, boards []*model.Board) ([]map[string]interface{}, error) {
	dbconn, err := db.New("Board")
	if err != nil {
		return nil, err
	}
	boardsCollection, boardsClient := dbconn.Collection, dbconn.Client
	defer boardsClient.Disconnect(context.TODO())

	var childBoards []map[string]interface{}
	var bsonBoards []interface{}

	if len(boards) == 0 {
		return childBoards, nil
	}

	for _, board := range boards {
		bsonBoards = append(bsonBoards, bson.M{"boardID": board.Id.Hex()})
	}

	// graphlookup: find child boards
	match := bson.D{{Key: "$match", Value: bson.D{{Key: "$or", Value: bsonBoards}}}}
	graphLookup := bson.D{{Key: "$graphLookup", Value: bson.D{
		{Key: "from", Value: "Boards"},
		{Key: "startWith", Value: "$boardID"},
		{Key: "connectFromField", Value: "boardID"},
		{Key: "connectToField", Value: "parentID"},
		{Key: "as", Value: "childBoards"},
	}}}
	boardsCursor, bmerror := boardsCollection.Aggregate(context.TODO(), mongo.Pipeline{match, graphLookup})
	if bmerror != nil {
		return nil, err
	}
	defer boardsCursor.Close(context.TODO())

	err = boardsCursor.All(context.TODO(), &childBoards)
	if err != nil {
		return nil, err
	}
	return childBoards, nil
}

func getBoardPermissionsByProfile(db *mongodatabase.DBConfig, boards []*model.Board, profileID int) (*model.BoardPermission, error) {
	strProfile := strconv.Itoa(profileID)
	boardPermissions := make(model.BoardPermission)
	parentBoardPermissions := make(model.BoardPermission)

	// searching permission in all boards
	for _, board := range boards {
		if board.Owner == strProfile {
			parentBoardPermissions[board.Id.Hex()] = "owner"
		} else if util.Contains(board.Admins, strProfile) {
			parentBoardPermissions[board.Id.Hex()] = "admin"
		} else if util.Contains(board.Authors, strProfile) {
			parentBoardPermissions[board.Id.Hex()] = "author"
		} else if util.Contains(board.Subscribers, strProfile) {
			parentBoardPermissions[board.Id.Hex()] = "subscriber"
		} else if util.Contains(board.Viewers, strProfile) {
			parentBoardPermissions[board.Id.Hex()] = "viewer"
		}
	}

	childBoards, childBoardsErr := findBoardMappings(db, boards)
	if childBoardsErr != nil {
		return nil, errors.Wrap(childBoardsErr, "unable to find child boards")
	}

	// setting parent permission in child boards
	for _, b := range childBoards {
		for _, cb := range b["childBoards"].(primitive.A) {
			if parentBoardPermissions[b["boardID"].(string)] == "owner" {
				boardPermissions[cb.(map[string]interface{})["boardID"].(string)] = "admin"
			} else {
				boardPermissions[cb.(map[string]interface{})["boardID"].(string)] = parentBoardPermissions[b["boardID"].(string)]
			}
		}
	}
	for _, board := range boards {
		if board.Owner == strProfile {
			boardPermissions[board.Id.Hex()] = "owner"
		} else if util.Contains(board.Admins, strProfile) {
			boardPermissions[board.Id.Hex()] = "admin"
		} else if util.Contains(board.Authors, strProfile) {
			boardPermissions[board.Id.Hex()] = "author"
		} else if util.Contains(board.Subscribers, strProfile) {
			boardPermissions[board.Id.Hex()] = "subscriber"
		} else if util.Contains(board.Viewers, strProfile) {
			boardPermissions[board.Id.Hex()] = "viewer"
		}
	}

	return &boardPermissions, nil
}

func addBoardMapping(db *mongodatabase.DBConfig, boardMapping *model.BoardMapping) error {
	dbconn, err := db.New("BoardMapping")
	if err != nil {
		return err
	}
	boardMappingCollection, boardMappingClient := dbconn.Collection, dbconn.Client
	defer boardMappingClient.Disconnect(context.TODO())

	_, err = boardMappingCollection.InsertOne(context.TODO(), boardMapping)
	if err != nil {
		return err
	}

	return nil
}

func getParentBoards(db *mongodatabase.DBConfig, boardID primitive.ObjectID) ([]map[string]interface{}, error) {
	dbconn, err := db.New("Board")
	if err != nil {
		return nil, err
	}
	boardsCollection, boardsClient := dbconn.Collection, dbconn.Client
	defer boardsClient.Disconnect(context.TODO())

	var parentBoards []map[string]interface{}
	var bsonBoards []interface{}
	bsonBoards = append(bsonBoards, bson.M{"boardID": boardID.Hex()})

	// graphlookup: find parent boards
	match := bson.D{{Key: "$match", Value: bson.D{{Key: "$or", Value: bsonBoards}}}}
	graphLookup := bson.D{{Key: "$graphLookup", Value: bson.D{
		{Key: "from", Value: "Boards"},
		{Key: "startWith", Value: "$parentID"},
		{Key: "connectFromField", Value: "parentID"},
		{Key: "connectToField", Value: "boardID"},
		{Key: "as", Value: "parentBoards"},
	}}}
	boardsCursor, bmerror := boardsCollection.Aggregate(context.TODO(), mongo.Pipeline{match, graphLookup})
	if bmerror != nil {
		return nil, err
	}
	defer boardsCursor.Close(context.TODO())

	err = boardsCursor.All(context.TODO(), &parentBoards)
	if err != nil {
		return nil, err
	}

	return parentBoards, nil
}

func boardUnfollow(db *database.Database, mongoDB *mongodatabase.DBConfig, cache *cache.Cache, boardID string, profileID int) (map[string]interface{}, error) {
	var count int
	var stmt string
	stmt = "SELECT COUNT(*) FROM `sidekiq-dev`.BoardsFollowed WHERE profileID = ? AND boardID = ?"
	err := db.Conn.Get(&count, stmt, profileID, boardID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch count from DB")
	}
	if count == 0 {
		return util.SetResponse(nil, 0, "You're not following this board"), nil
	}

	stmt = "DELETE FROM `sidekiq-dev`.BoardsFollowed WHERE boardID = ? and profileID = ?;"
	_, err = db.Conn.Exec(stmt, boardID, profileID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to delete from database")
	}

	// remove from mongo
	dbconn, err := mongoDB.New(consts.Board)
	if err != nil {
		return nil, err
	}
	boardCollection, boardClient := dbconn.Collection, dbconn.Client
	defer boardClient.Disconnect(context.TODO())

	boardObjectID, err := primitive.ObjectIDFromHex(boardID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to decode objectID")
	}

	var board *model.Board
	err = boardCollection.FindOne(context.TODO(), bson.M{"_id": boardObjectID}).Decode(&board)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find board")
	}

	var role string

	if util.Contains(board.Followers, strconv.Itoa(profileID)) {
		board.Followers = util.Remove(board.Followers, strconv.Itoa(profileID))
	}
	if util.Contains(board.Admins, strconv.Itoa(profileID)) {
		role = "admin"
	} else if util.Contains(board.Subscribers, strconv.Itoa(profileID)) {
		role = "subscriber"
	} else if util.Contains(board.Authors, strconv.Itoa(profileID)) {
		role = "author"
	} else if board.Owner == strconv.Itoa(profileID) {
		role = "owner"
	}

	_, err = boardCollection.UpdateOne(context.TODO(), bson.M{"_id": boardObjectID}, bson.M{"$set": board})
	if err != nil {
		return nil, errors.Wrap(err, "unable to update board at mongo")
	}

	// check if the member is still on the board keep the cache intact or else remove from cache
	key := fmt.Sprintf("boards:%s", strconv.Itoa(profileID))
	bp := permissions.GetBoardPermissionsNew(key, cache, board, strconv.Itoa(profileID))
	if role != "" {
		bp[boardID] = role
	} else {
		delete(bp, boardID)
	}
	cache.SetValue(key, bp.ToJSON())
	return util.SetResponse(nil, 1, "Board unfollowed successfully"), nil
}

func boardFollow(mongoDB *mongodatabase.DBConfig, sql *database.Database, cache *cache.Cache, payload model.BoardFollowInfo) (map[string]interface{}, error) {
	var count int
	stmt := "SELECT COUNT(*) FROM `sidekiq-dev`.BoardsFollowed WHERE profileID = ? AND boardID = ?"
	err := sql.Conn.Get(&count, stmt, payload.ProfileID, payload.BoardID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch count from DB")
	}
	if count > 0 {
		return util.SetResponse(nil, 0, "You're already following this board"), nil
	}

	dbconn, err := mongoDB.New(consts.Board)
	if err != nil {
		return nil, err
	}
	boardCollection, boardClient := dbconn.Collection, dbconn.Client
	defer boardClient.Disconnect(context.TODO())

	boardObjectID, err := primitive.ObjectIDFromHex(payload.BoardID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to decode objectID")
	}

	var board *model.Board
	err = boardCollection.FindOne(context.TODO(), bson.M{"_id": boardObjectID, "state": consts.Active}).Decode(&board)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find board")
	}

	profileIDStr := strconv.Itoa(payload.ProfileID)

	// check profile not blocked by board owner profile
	connectionID, _ := strconv.Atoi(board.Owner)
	dbconn, err = mongoDB.New(consts.Connection)
	if err != nil {
		return nil, errors.Wrap(err, "error in connecting to mongo 'Connection'")
	}
	coll, client := dbconn.Collection, dbconn.Client
	defer client.Disconnect(context.TODO())
	var result struct {
		Count int `bson:"count"`
	}
	countPipeline := mongo.Pipeline{
		bson.D{
			{Key: "$match", Value: bson.M{
				"profileID":    payload.ProfileID,
				"connectionID": connectionID,
				"blocked":      true,
			}},
		},
		bson.D{
			{Key: "$count", Value: "count"},
		},
	}
	cursor, err := coll.Aggregate(context.TODO(), countPipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	if cursor.Next(context.TODO()) {
		err = cursor.Decode(&result)
		if err != nil {
			return nil, err
		}
	}
	if result.Count > 0 {
		return util.SetResponse(nil, 0, "Profile blocked by board-owner profile"), nil
	}

	if board.Visible != consts.Public {
		return util.SetResponse(nil, 0, "Board is not public"), nil
	}

	// check if the follower is already a member on the board or not
	msg := "User is already %s on the board"
	if util.Contains(board.Blocked, profileIDStr) {
		return util.SetResponse(nil, 0, fmt.Sprintf(msg, "blocked")), nil
	}

	// add profile as a follower in board
	if !util.Contains(board.Followers, profileIDStr) {
		board.Followers = append(board.Followers, profileIDStr)
	}

	_, err = boardCollection.UpdateOne(context.TODO(), bson.M{"_id": board.Id}, bson.M{"$set": board})
	if err != nil {
		return nil, errors.Wrap(err, "unable to update board at mongo")
	}

	// all checks passed so insert in MySQL
	payload.OwnerID = connectionID
	payload.BoardTitle = board.Title
	y, m, d := board.CreateDate.Date()
	payload.CreateDate = fmt.Sprintf("%v-%d-%v", y, m, d)
	stmt = "INSERT INTO `sidekiq-dev`.BoardsFollowed (profileID, boardID, boardTitle, createDate, ownerID) " +
		"VALUES (:profileID, :boardID, :boardTitle, :createDate, :ownerID);"
	_, err = sql.Conn.NamedExec(stmt, payload)
	if err != nil {
		return nil, errors.Wrap(err, "unable to insert in database")
	}

	// if the profile is NOT a member on the board then he/she gets viewer permission
	if !util.Contains(board.Admins, profileIDStr) && !util.Contains(board.Authors, profileIDStr) &&
		!util.Contains(board.Subscribers, profileIDStr) {
		// cache
		key := fmt.Sprintf("boards:%s", profileIDStr)
		bp := permissions.GetBoardPermissionsNew(key, cache, board, profileIDStr)
		bp[payload.BoardID] = "viewer"
		cache.SetValue(key, bp.ToJSON())
	}
	return util.SetResponse(nil, 1, "Board followed successfully"), nil
}

func boardSettings(cache *cache.Cache, profileService peoplerpc.AccountServiceClient, storageService storage.Service,
	sql *database.Database, db *mongodatabase.DBConfig, id string, profileID int, payload map[string]interface{},
) (map[string]interface{}, error) {
	profileIDStr := strconv.Itoa(profileID)
	dbconn, err := db.New(consts.Board)
	if err != nil {
		return nil, err
	}
	boardCollection, boardsClient := dbconn.Collection, dbconn.Client
	defer boardsClient.Disconnect(context.TODO())
	boardID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var boardInfo *model.Board
	err = boardCollection.FindOne(context.TODO(), bson.M{"_id": boardID}).Decode(&boardInfo)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find board info")
	}
	ownerKey := fmt.Sprintf("boards:%s", profileIDStr)
	ownerBoardPermission := permissions.GetBoardPermissionsNew(ownerKey, cache, boardInfo, strconv.Itoa(profileID))
	role := ownerBoardPermission[id]
	if role == "" || (role != "owner" && role != "admin") {
		if boardInfo.Owner != profileIDStr && !util.Contains(boardInfo.Admins, profileIDStr) {
			return util.SetResponse(nil, 0, "User do not have access to update this board settings"), nil
		}
	}

	if val, ok := payload["state"]; ok {
		if boardInfo.Hidden && val.(string) != consts.Hidden { // unhide
			payload["hidden"] = false
		}
		if !boardInfo.Hidden && val.(string) == consts.Hidden { // hide
			payload["hidden"] = true
		}
	}

	filter := bson.M{"_id": boardID}
	_, err = boardCollection.UpdateOne(context.TODO(), filter, bson.M{"$set": payload})
	if err != nil {
		return nil, errors.Wrap(err, "unable to delete board at mongo")
	}

	// fetch owner profile
	// ownerInfo, err := profileService.FetchConciseProfile(profileID)
	// if err != nil {
	// 	return nil, errors.Wrap(err, "unable to find owner's info.")
	// }

	cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(profileID)}
	ownerInfo, err := profileService.GetConciseProfile(context.TODO(), cpreq)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find owner's info.")
	}

	// get updated board
	err = boardCollection.FindOne(context.TODO(), bson.M{"_id": boardID}).Decode(&boardInfo)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find board info")
	}
	boardInfo.OwnerInfo = ownerInfo

	// get location
	loc, err := fetchThingLocationOnBoard(db, id)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find board's location")
	}
	boardInfo.Location = loc

	return util.SetResponse(boardInfo, 1, "Board settings saved successfully"), nil
}

// based on gRPC
func getBoardMembers2(db *mongodatabase.DBConfig, mysqlDB *database.Database, profileService peoplerpc.AccountServiceClient, storageService storage.Service, boardID, limit, page, search, role string) (map[string]interface{}, error) {
	var err error
	dbconn, err := db.New(consts.Board)
	if err != nil {
		return nil, err
	}
	boardCollection, boardsClient := dbconn.Collection, dbconn.Client
	defer boardsClient.Disconnect(context.TODO())

	boardObjID, err := primitive.ObjectIDFromHex(boardID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to decode board ObjectID")
	}

	var board *model.Board
	filter := bson.M{"_id": boardObjID}
	err = boardCollection.FindOne(context.TODO(), filter).Decode(&board)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find the board")
	}

	contains := func(s []int32, b int32) bool {
		for _, bmr := range s {
			if bmr == b {
				return true
			}
		}
		return false
	}

	var memberIDs []int32
	idInt, err := strconv.Atoi(board.Owner)
	if err != nil {
		return nil, err
	}

	if !contains(memberIDs, int32(idInt)) {
		memberIDs = append(memberIDs, int32(idInt))
	}

	if len(board.Admins) != 0 {
		for _, member := range board.Admins {
			idInt, err := strconv.Atoi(member)
			if err != nil {
				return nil, err
			}
			if !contains(memberIDs, int32(idInt)) {
				memberIDs = append(memberIDs, int32(idInt))
			}
		}
	}

	if len(board.Subscribers) != 0 {
		for _, member := range board.Subscribers {
			idInt, err := strconv.Atoi(member)
			if err != nil {
				return nil, err
			}
			if !contains(memberIDs, int32(idInt)) {
				memberIDs = append(memberIDs, int32(idInt))
			}
		}
	}

	if len(board.Authors) != 0 {
		for _, member := range board.Authors {
			idInt, err := strconv.Atoi(member)
			if err != nil {
				return nil, err
			}
			if !contains(memberIDs, int32(idInt)) {
				memberIDs = append(memberIDs, int32(idInt))
			}
		}
	}

	if len(board.Guests) != 0 {
		for _, member := range board.Guests {
			idInt, err := strconv.Atoi(member)
			if err != nil {
				return nil, err
			}
			if !contains(memberIDs, int32(idInt)) {
				memberIDs = append(memberIDs, int32(idInt))
			}
		}
	}

	return util.SetResponse(memberIDs, 1, "memberIDs fetched successfully"), nil
}

func getBoardMembers(db *mongodatabase.DBConfig, mysqlDB *database.Database, profileService peoplerpc.AccountServiceClient, storageService storage.Service, boardID, limit, page, search, role string) (map[string]interface{}, error) {
	var err error
	dbconn, err := db.New(consts.Board)
	if err != nil {
		return nil, err
	}
	boardCollection, boardsClient := dbconn.Collection, dbconn.Client
	defer boardsClient.Disconnect(context.TODO())

	boardObjID, err := primitive.ObjectIDFromHex(boardID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to decode board ObjectID")
	}

	var board *model.Board
	filter := bson.M{"_id": boardObjID}
	err = boardCollection.FindOne(context.TODO(), filter).Decode(&board)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find the board")
	}

	members := []model.BoardMemberRole{}
	membersWithPermissions := []model.BoardMemberRole{}
	var memberIDs, blockedMemberIDs []string

	// get role
	getRole := func(userRole, roleOnBoard string) string {
		if userRole == "" {
			return roleOnBoard
		} else {
			return strings.ToLower(userRole)
		}
	}

	// contains
	contains := func(s []model.BoardMemberRole, b model.BoardMemberRole) bool {
		for _, bmr := range s {
			if bmr == b {
				return true
			}
		}
		return false
	}

	role = strings.ToLower(role)
	if len(board.Admins) != 0 {
		for _, member := range board.Admins {
			m := model.BoardMemberRole{}
			m.Role = getRole(role, consts.Admin)
			m.ProfileID = member
			if !contains(members, m) {
				members = append(members, m)
			}
		}
	}

	if len(board.Subscribers) != 0 {
		for _, member := range board.Subscribers {
			m := model.BoardMemberRole{}
			m.Role = getRole(role, consts.Subscriber)
			m.ProfileID = member
			if !contains(members, m) {
				members = append(members, m)
			}
		}
	}

	if len(board.Authors) != 0 {
		for _, member := range board.Authors {
			m := model.BoardMemberRole{}
			m.Role = getRole(role, consts.Author)
			m.ProfileID = member
			if !contains(members, m) {
				members = append(members, m)
			}
		}
	}

	if len(board.Guests) != 0 {
		for _, member := range board.Guests {
			m := model.BoardMemberRole{}
			if role == "" {
				m.Role = consts.Guest
			} else {
				m.Role = strings.ToLower(role)
			}
			m.ProfileID = member
			if !contains(members, m) {
				members = append(members, m)
			}
		}
	}

	memberIDs = append(memberIDs, board.Admins...)
	if role == consts.Viewer {
		memberIDs = append(memberIDs, board.Viewers...)
	}
	memberIDs = append(memberIDs, board.Subscribers...)
	memberIDs = append(memberIDs, board.Authors...)
	blockedMemberIDs = append(blockedMemberIDs, board.Blocked...)

	var searchFilter string
	if search != "" {
		searchFilter = `AND
    (
         CONCAT(firstName, ' ', lastName) LIKE '%` + search + `%'
         OR screenName LIKE '%` + search + `%'
    )`
	}

	errChan := make(chan error, len(members))
	goRoutines := 0

	// get basic info from mysql
	for _, member := range members {
		goRoutines += 1
		go func(member model.BoardMemberRole, errChan chan<- error) {
			stmt := `SELECT id, firstName, lastName,accountID, IFNULL(screenName, '') AS screenName, 
            IFNULL(photo, '') as photo FROM` + "`sidekiq-dev`.AccountProfile WHERE id = ? " + searchFilter
			// could have used this service
			// profileService.FetchConciseProfile()
			profileID, _ := strconv.Atoi(member.ProfileID)
			err := mysqlDB.Conn.Get(&member, stmt, profileID)
			if err != nil {
				if err == sql.ErrNoRows {
					return
				} else {
					errChan <- errors.Wrap(err, "unable to find basic info")
				}
			} else {
				if member.Photo == "" {
					// photo, err := getProfileImage(mysqlDB, storageService, member.AccountID, profileID)
					// cp, err := profileService.FetchConciseProfile(profileID)
					// if err != nil {
					// 	fmt.Println("photo not found for profile", profileID)
					// 	errChan <- errors.Wrap(err, "unable to find profile image")
					// }
					cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(profileID)}
					cp, err := profileService.GetConciseProfile(context.TODO(), cpreq)
					if err != nil {
						errChan <- errors.Wrap(err, "unable to find owner's info.")
					}

					member.Photo = cp.Photo
				}

				// fetch this from profile service
				thumbs, err := getProfileImageThumb(mysqlDB, storageService, member.AccountID, profileID)
				// thumbs, err := profileService.(mysqlDB, storageService, member.AccountID, profileID)
				if err != nil {
					fmt.Println("photo not found for profile thumb", profileID)
					member.Thumbs = model.Thumbnails{}
				} else {
					member.Thumbs.Original = member.Photo
					member.Thumbs = thumbs
				}

				membersWithPermissions = append(membersWithPermissions, member)
				errChan <- nil
			}
		}(member, errChan)
	}

	// waiting for the goroutines to finish
	for goRoutines != 0 {
		if err := <-errChan; err != nil {
			return nil, errors.Wrap(err, "error from go routine(fetching board members)")
		}
		goRoutines--
	}

	res := make(map[string]interface{})
	res["data"] = make(map[string]interface{})

	// get the owner info
	profileID, _ := strconv.Atoi(board.Owner)
	// cp, err := profileService.FetchConciseProfile(profileID)
	// if err != nil {
	// 	return nil, errors.Wrap(err, "unable to fetch concise profile")
	// }
	cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(profileID)}
	cp, err := profileService.GetConciseProfile(context.TODO(), cpreq)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch concise profile")
	}

	res["data"].(map[string]interface{})["ownerInfo"] = cp

	if len(memberIDs) == 0 {
		res["data"].(map[string]interface{})["memberIDs"] = []int{}
	} else {
		res["data"].(map[string]interface{})["memberIDs"] = memberIDs
	}
	if len(blockedMemberIDs) == 0 {
		res["data"].(map[string]interface{})["blockedMemberIDs"] = []int{}
	} else {
		res["data"].(map[string]interface{})["blockedMemberIDs"] = blockedMemberIDs
	}

	// pagination
	var pageNo int
	pageNo, _ = strconv.Atoi(page)
	limitInt, _ := strconv.Atoi(limit)
	subset := paginate(membersWithPermissions, pageNo, limitInt)

	if len(membersWithPermissions) == 0 {
		res["data"].(map[string]interface{})["info"] = []int{}
		res["data"].(map[string]interface{})["total"] = 0
		res["status"] = 1
		res["message"] = "No such member exists."
		return res, nil
	}

	res["data"].(map[string]interface{})["info"] = subset
	res["data"].(map[string]interface{})["total"] = len(membersWithPermissions)
	res["status"] = 1
	res["message"] = "All members fetched successfully"

	return res, nil
}

func fetchConnectionsMembers(db *mongodatabase.DBConfig, mysqlDB *database.Database, profileID int, boardID string) (map[string]interface{}, error) {
	var err error
	dbconn, err := db.New(consts.Board)
	if err != nil {
		return nil, err
	}
	boardCollection, boardsClient := dbconn.Collection, dbconn.Client
	defer boardsClient.Disconnect(context.TODO())

	profileIDStr := strconv.Itoa(profileID)
	boardObjID, err := primitive.ObjectIDFromHex(boardID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to decode board ObjectID")
	}

	var board *model.Board

	filter := bson.M{"_id": boardObjID, "owner": profileIDStr}
	err = boardCollection.FindOne(context.TODO(), filter).Decode(&board)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find the board")
	}

	dbconn2, err := db.New(consts.Connection)
	if err != nil {
		return nil, err
	}

	connCollection, connClient := dbconn2.Collection, dbconn2.Client
	defer connClient.Disconnect(context.TODO())

	findConnFilter := bson.M{"profileID": profileIDStr}
	cursor, err := connCollection.Find(context.TODO(), findConnFilter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find boards")
	}

	members := []model.BoardMemberRole{}
	membersFinal := []model.BoardMemberRole{}

	err = cursor.All(context.TODO(), &members)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find profile's connections.")
	}

	for _, member := range members {
		// check if the member is present in the board or not
		if util.Contains(board.Admins, member.ProfileID) {
			member.Role = "admin"
		} else if util.Contains(board.Authors, member.ProfileID) {
			member.Role = "author"
		} else if util.Contains(board.Subscribers, member.ProfileID) {
			member.Role = "subscriber"
		} else if util.Contains(board.Viewers, member.ProfileID) {
			member.Role = "viewer"
		}
		membersFinal = append(membersFinal, member)
	}
	if len(membersFinal) == 0 {
		return util.SetResponse(nil, 1, "You have no members in your connections."), nil
	} else {
		return util.SetResponse(membersFinal, 1, "Members from your connections fetched successfully"), nil
	}
}

// func inviteMembers(cache *cache.Cache, profileService profile.Service, storageService storage.Service, db *mongodatabase.DBConfig,
// 	mysql *database.Database, boardID string, profileID int,
// 	invites []model.BoardMemberRoleRequest) (map[string]interface{}, error) {
// 	var stmt string
// 	var dups, nonDups int
// 	var alreadyPresent []string
// 	var err error

// 	dbconn, err := db.New(consts.Board)
// 	if err != nil {
// 		return nil, errors.Wrap(err, "unable to connect to Board")
// 	}
// 	boardCollection, boardClient := dbconn.Collection, dbconn.Client
// 	defer boardClient.Disconnect(context.TODO())

// 	boardObjID, _ := primitive.ObjectIDFromHex(boardID)
// 	var board model.Board
// 	err = boardCollection.FindOne(context.TODO(), bson.M{"_id": boardObjID}).Decode(&board)
// 	if err != nil {
// 		return nil, errors.Wrap(err, "no such board exists")
// 	}

// 	for _, invite := range invites {
// 		bi := model.BoardInvite{
// 			BoardID:   boardID,
// 			SenderID:  strconv.Itoa(profileID),
// 			InviteeID: invite.ProfileID,
// 			Role:      invite.Role,
// 		}

// 		// get the basic info
// 		idInt, _ := strconv.Atoi(bi.InviteeID)
// 		cp, err := profileService.FetchConciseProfile(idInt)
// 		if err != nil {
// 			return nil, errors.Wrap(err, "unable to find basic info")
// 		}

// 		// check if the invitation record already exists
// 		var count int
// 		stmt = "SELECT COUNT(*) FROM `sidekiq-dev`.BoardInvites WHERE inviteeID = ? AND boardID = ?"
// 		err = mysql.Conn.Get(&count, stmt, bi.InviteeID, bi.BoardID)
// 		if err != nil {
// 			return nil, errors.Wrap(err, "unable to get record's existence")
// 		}
// 		// msg := "%s %s is already a member on the board"

// 		if count == 0 {
// 			// check if the inviteeID is already on that board or not
// 			if util.Contains(board.Admins, bi.InviteeID) ||
// 				util.Contains(board.Authors, bi.InviteeID) ||
// 				util.Contains(board.Viewers, bi.InviteeID) ||
// 				util.Contains(board.Subscribers, bi.InviteeID) {
// 				alreadyPresent = append(alreadyPresent, fmt.Sprintf("%s %s", cp.FirstName, cp.LastName))
// 			} else { // not an existing member on the board
// 				stmt = "INSERT INTO `sidekiq-dev`.BoardInvites(id, senderID, inviteeID, boardID, role) VALUES (:id, :senderID, :inviteeID, :boardID, :role) "
// 				_, err = mysql.Conn.NamedExec(stmt, bi)
// 				if err != nil {
// 					return nil, errors.Wrap(err, "unable to invite member")
// 				}
// 				nonDups += 1
// 			}
// 		} else {
// 			dups += 1
// 			continue
// 		}
// 	}

// 	if len(invites) == len(alreadyPresent) {
// 		return util.SetResponse(nil, 1, "Already members on the board"), nil
// 	}
// 	if nonDups > 0 {
// 		return util.SetResponse(nil, 1, "Invites sent successfully!"), nil
// 		// if len(alreadyPresent) == 0 {
// 		// } else {
// 		// 	return util.SetResponse(nil, 1, fmt.Sprintf("%v are already member(s) on the board. Other invites sent successfully!", alreadyPresent)), nil
// 		// }
// 	} else {
// 		return util.SetResponse(nil, 1, "Invites are already sent!"), nil
// 	}
// }

func inviteMembers(cache *cache.Cache, profileService profile.Service, storageService storage.Service, db *mongodatabase.DBConfig,
	mysql *database.Database, boardID string, profileID int,
	invites []model.BoardMemberRoleRequest,
) (map[string]interface{}, error) {
	var stmt string
	var dups, nonDups int
	var err error
	for _, invite := range invites {
		bi := model.BoardInvite{
			BoardID:   boardID,
			SenderID:  strconv.Itoa(profileID),
			InviteeID: invite.ProfileID,
			Role:      invite.Role,
		}
		// check if the invitation record already exists
		var count int
		stmt = "SELECT COUNT(*) FROM `sidekiq-dev`.BoardInvites WHERE inviteeID = ? AND boardID = ?"
		err = mysql.Conn.Get(&count, stmt, bi.InviteeID, bi.BoardID)
		if err != nil {
			return nil, errors.Wrap(err, "unable to get record's existence")
		}

		if count == 0 {
			stmt = "INSERT INTO `sidekiq-dev`.BoardInvites(id, senderID, inviteeID, boardID, role) VALUES (:id, :senderID, :inviteeID, :boardID, :role) "
			_, err = mysql.Conn.NamedExec(stmt, bi)
			if err != nil {
				return nil, errors.Wrap(err, "unable to invite member")
			}
			nonDups += 1
		} else {
			dups += 1
			continue
		}
	}

	if nonDups > 0 {
		return util.SetResponse(nil, 1, "Invites have been sent!"), nil
	} else {
		return util.SetResponse(nil, 1, "Invites are already sent!"), nil
	}
}

func handleBoardInvitation(cache *cache.Cache, db *mongodatabase.DBConfig, mysql *database.Database,
	profileID int, boardInvitation model.HandleBoardInvitation,
) (map[string]interface{}, error) {
	var err error
	var setDelete bool
	var resMsg string
	type data struct {
		SenderID int    `db:"senderID"`
		Role     string `db:"role"`
	}
	d := data{}

	profileIDStr := strconv.Itoa(profileID)

	if boardInvitation.Type == "accept" {
		var role string
		// get the role from mysql
		stmt := "SELECT role, senderID FROM `sidekiq-dev`.BoardInvites WHERE boardID = ? AND InviteeID = ?"
		err = mysql.Conn.Get(&d, stmt, boardInvitation.BoardID, profileID)
		if err != nil {
			if err == sql.ErrNoRows {
				return util.SetResponse(nil, 0, "No such invitation record found"), nil
			} else {
				return nil, errors.Wrap(err, "unable to accept the invitation")
			}
		}

		role = d.Role

		// get the board
		dbconn, err := db.New(consts.Board)
		if err != nil {
			return nil, err
		}
		boardCollection, boardsClient := dbconn.Collection, dbconn.Client
		defer boardsClient.Disconnect(context.TODO())

		var board *model.Board
		boardObjID, _ := primitive.ObjectIDFromHex(boardInvitation.BoardID)
		boardCollection.FindOne(context.TODO(), bson.M{"_id": boardObjID}).Decode(&board)

		switch role {
		case "admin":
			if !util.Contains(board.Admins, profileIDStr) {
				board.Admins = append(board.Admins, profileIDStr)
			}
		case "author":
			if !util.Contains(board.Authors, profileIDStr) {
				board.Authors = append(board.Authors, profileIDStr)
			}
		case "subscriber":
			if !util.Contains(board.Subscribers, profileIDStr) {
				board.Subscribers = append(board.Subscribers, profileIDStr)
			}
		case "viewer":
			if !util.Contains(board.Viewers, profileIDStr) {
				board.Viewers = append(board.Viewers, profileIDStr)
			}
		case "guest":
			if !util.Contains(board.Guests, profileIDStr) {
				board.Guests = append(board.Guests, profileIDStr)
			}

		}

		// update the board
		_, err = boardCollection.UpdateOne(context.TODO(), bson.M{"_id": boardObjID}, bson.M{"$set": board})
		if err != nil {
			return nil, errors.Wrap(err, "unable to update the board")
		}

		// cache the user
		cacheKey := fmt.Sprintf("boards:%s", profileIDStr)
		bp := permissions.GetBoardPermissionsNew(cacheKey, cache, board, profileIDStr)
		if role == "guest" {
			bp[boardInvitation.BoardID] = "viewer"
			cache.SetValue(cacheKey, bp.ToJSON())
		} else {
			bp[boardInvitation.BoardID] = role
			cache.SetValue(cacheKey, bp.ToJSON())
		}
		setDelete = true
		resMsg = "Invitation accepted!"
	} else { // handle reject
		setDelete = true
		resMsg = "Invitation rejected!"
	}

	// remove invitation record from mysql
	if setDelete {
		bi := model.BoardInvite{}
		bi.BoardID = boardInvitation.BoardID
		bi.InviteeID = profileIDStr
		stmt := "DELETE FROM `sidekiq-dev`.BoardInvites WHERE boardID = :boardID AND inviteeID = :inviteeID"
		_, err = mysql.Conn.NamedExec(stmt, bi)
		if err != nil {
			return nil, err
		}
	}

	return util.SetResponse(d.SenderID, 1, resMsg), nil
}

func listBoardInvites(storageService storage.Service, db *mongodatabase.DBConfig, mysql *database.Database, profileID int) (map[string]interface{}, error) {
	boardInvitations := []model.BoardInvite{}
	stmt := "SELECT * FROM `sidekiq-dev`.BoardInvites WHERE inviteeID = ? ORDER BY createDate ASC"
	err := mysql.Conn.Select(&boardInvitations, stmt, profileID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to list your invitations")
	}

	if len(boardInvitations) == 0 {
		return util.SetResponse(nil, 1, "You have no pending board invitations"), nil
	}

	dbconn, err := db.New(consts.Board)
	if err != nil {
		return nil, errors.Wrap(err, "unable to establish connection w/ Board")
	}
	boardCollection, boardsClient := dbconn.Collection, dbconn.Client
	defer boardsClient.Disconnect(context.TODO())

	opts := options.FindOne().SetProjection(
		bson.M{"title": 1},
	)

	invitations := []model.ListInvitations{}

	errChan := make(chan error)
	goRoutines := 0

	for _, invite := range boardInvitations {
		goRoutines++
		go func(invite model.BoardInvite, errChan chan<- error) {
			i := model.ListInvitations{}
			i.BoardID = invite.BoardID
			i.Role = invite.Role
			i.CreatedAt = invite.CreatedAt
			stmt := "SELECT firstName, lastName, accountID FROM `sidekiq-dev`.AccountProfile where id = ?"
			err = mysql.Conn.Get(&i, stmt, invite.SenderID)
			if err != nil {
				errChan <- errors.Wrap(err, "unable to fetch your invitation")
			}
			userID, err := strconv.Atoi(i.UserID)
			if err != nil {
				errChan <- errors.Wrap(err, "unable to convert userID str to int board invitation")
			}
			profileID, err := strconv.Atoi(invite.SenderID)
			if err != nil {
				errChan <- errors.Wrap(err, "unable to convert profileID str to int board invitation")
			}

			if i.Photo == "" {
				thumbs, err := helper.GetThumbnails(
					storageService,
					util.GetKeyForProfileImage(userID, profileID, "thumbs"),
					fmt.Sprintf("%d.png", profileID),
					[]string{"ic", "sm"},
				)
				if err != nil {
					errChan <- err
				}
				if thumbs.Icon != "" {
					i.Photo = thumbs.Icon
				} else {
					i.Photo = thumbs.Small
				}

				// fetch board title based on boardID
				boardObjID, _ := primitive.ObjectIDFromHex(invite.BoardID)
				var bt map[string]interface{}
				_ = boardCollection.FindOne(context.TODO(), bson.M{"_id": boardObjID}, opts).Decode(&bt)
				i.BoardTitle = bt["title"].(string)
				invitations = append(invitations, i)

				errChan <- nil
			}
		}(invite, errChan)
	}

	for goRoutines != 0 {
		if err := <-errChan; err != nil {
			return nil, err
		}
		goRoutines--
	}

	sort.Slice(invitations, func(i, j int) bool {
		return invitations[i].CreatedAt.After(invitations[j].CreatedAt)
	})

	return util.SetResponse(invitations, 1, "Your invitations are fetched successfully"), nil
}

func changeProfileRole(cache *cache.Cache, db *mongodatabase.DBConfig, profileID int, boardID string, cbp model.ChangeProfileRole) (map[string]interface{}, error) {
	var err error
	dbconn, err := db.New(consts.Board)
	if err != nil {
		return nil, err
	}
	boardCollection, boardsClient := dbconn.Collection, dbconn.Client
	defer boardsClient.Disconnect(context.TODO())

	profileIDStr := strconv.Itoa(profileID)
	isValid, err := permissions.CheckValidPermissions(profileIDStr, cache, boardCollection, boardID, []string{consts.Owner, consts.Admin}, false)
	if err != nil {
		return nil, err
	}
	if !isValid {
		return util.SetResponse(nil, 0, "User does not have the authority to change roles."), nil
	}

	var board *model.Board
	boardObjID, _ := primitive.ObjectIDFromHex(boardID)
	filter := bson.M{"_id": boardObjID}
	err = boardCollection.FindOne(context.TODO(), filter).Decode(&board)
	if err != nil {
		return nil, err
	}

	// remove from old permission array
	switch cbp.OldRole {
	case "admin":
		board.Admins = util.Remove(board.Admins, cbp.ProfileID)
	case "viewer":
		board.Viewers = util.Remove(board.Viewers, cbp.ProfileID)
	case "author":
		board.Authors = util.Remove(board.Authors, cbp.ProfileID)
	case "blocked":
		board.Blocked = util.Remove(board.Blocked, cbp.ProfileID)
	case "subscriber":
		board.Subscribers = util.Remove(board.Subscribers, cbp.ProfileID)
	case "guest":
		board.Guests = util.Remove(board.Guests, cbp.ProfileID)
	}

	// add to new permission array
	switch cbp.NewRole {
	case "admin":
		if !util.Contains(board.Admins, cbp.ProfileID) {
			board.Admins = append(board.Admins, cbp.ProfileID)
		}
	case "viewer":
		if !util.Contains(board.Viewers, cbp.ProfileID) {
			board.Viewers = append(board.Viewers, cbp.ProfileID)
		}
	case "author":
		if !util.Contains(board.Authors, cbp.ProfileID) {
			board.Authors = append(board.Authors, cbp.ProfileID)
		}
	case "blocked":
		if !util.Contains(board.Blocked, cbp.ProfileID) {
			board.Blocked = append(board.Blocked, cbp.ProfileID)
		}
	case "subscriber":
		if !util.Contains(board.Subscribers, cbp.ProfileID) {
			board.Subscribers = append(board.Subscribers, cbp.ProfileID)
		}
	case "guest":
		if !util.Contains(board.Guests, cbp.ProfileID) {
			board.Guests = append(board.Guests, cbp.ProfileID)
		}
	}

	// update the board
	_, err = boardCollection.UpdateOne(context.TODO(), filter, bson.M{"$set": board})
	if err != nil {
		return nil, err
	}

	// change the permission in cache
	cacheKey := fmt.Sprintf("boards:%s", cbp.ProfileID)
	bp := permissions.GetBoardPermissionsNew(cacheKey, cache, board, cbp.ProfileID)
	if cbp.NewRole == "guest" {
		bp[boardID] = "viewer"
		cache.SetValue(cacheKey, bp.ToJSON())
	} else {
		bp[boardID] = cbp.NewRole
		cache.SetValue(cacheKey, bp.ToJSON())
	}

	return util.SetResponse(nil, 1, "Role changed successfully."), nil
}

func blockMembers(cache *cache.Cache, db *mongodatabase.DBConfig, profileID int, boardID string, membersToBlock []model.BoardMemberRoleRequest) (map[string]interface{}, error) {
	var err error
	dbconn, err := db.New(consts.Board)
	if err != nil {
		return nil, err
	}
	boardCollection, boardsClient := dbconn.Collection, dbconn.Client
	defer boardsClient.Disconnect(context.TODO())

	var board *model.Board
	boardObjID, _ := primitive.ObjectIDFromHex(boardID)
	filter := bson.M{"_id": boardObjID}
	err = boardCollection.FindOne(context.TODO(), filter).Decode(&board)
	if err != nil {
		return nil, err
	}

	profileIDStr := strconv.Itoa(profileID)
	ownerKey := fmt.Sprintf("boards:%s", profileIDStr)
	ownerBoardPermission := permissions.GetBoardPermissionsNew(ownerKey, cache, board, profileIDStr)
	role := ownerBoardPermission[boardID]
	if role != "owner" && role != "admin" {
		return util.SetResponse(nil, 0, "User do not have the permission to block members"), nil
	}

	// remove from existing permission array
	for _, memberToBlock := range membersToBlock {
		switch memberToBlock.Role {
		case "admin":
			board.Admins = util.Remove(board.Admins, memberToBlock.ProfileID)
		case "author":
			board.Authors = util.Remove(board.Authors, memberToBlock.ProfileID)
		case "viewer":
			board.Viewers = util.Remove(board.Viewers, memberToBlock.ProfileID)
		case "subscriber":
			board.Subscribers = util.Remove(board.Subscribers, memberToBlock.ProfileID)
		case "guest":
			board.Guests = util.Remove(board.Guests, memberToBlock.ProfileID)
		}

		if !util.Contains(board.Blocked, memberToBlock.ProfileID) {
			board.Blocked = append(board.Blocked, memberToBlock.ProfileID)
		}

		// update cache
		cacheKey := fmt.Sprintf("boards:%s", memberToBlock.ProfileID)
		bp := permissions.GetBoardPermissionsNew(cacheKey, cache, board, memberToBlock.ProfileID)
		bp[boardID] = "blocked"
		cache.SetValue(cacheKey, bp.ToJSON())
	}

	// update board
	_, err = boardCollection.UpdateOne(context.TODO(), filter, bson.M{"$set": board})
	if err != nil {
		return nil, errors.Wrap(err, "unable to update the board")
	}

	return util.SetResponse(nil, 1, "Members blocked successfully"), nil
}

func unblockMembers(db *mongodatabase.DBConfig, cache *cache.Cache, profileID int, boardID string, membersToUnblock []model.BoardMemberRoleRequest) (map[string]interface{}, error) {
	var err error
	dbconn, err := db.New(consts.Board)
	if err != nil {
		return nil, err
	}
	boardCollection, boardsClient := dbconn.Collection, dbconn.Client
	defer boardsClient.Disconnect(context.TODO())

	profileIDStr := strconv.Itoa(profileID)
	isValid, err := permissions.CheckValidPermissions(profileIDStr, cache, boardCollection, boardID, []string{"owner", "admin"}, false)
	if err != nil {
		return nil, err
	}

	if !isValid {
		return util.SetResponse(nil, 0, "User does not have access to the board."), nil
	}

	var board *model.Board
	boardObjID, _ := primitive.ObjectIDFromHex(boardID)
	err = boardCollection.FindOne(context.TODO(), bson.M{"_id": boardObjID}).Decode(&board)
	if err != nil {
		return nil, err
	}

	for _, member := range membersToUnblock {
		switch member.Role {
		case "admin":
			if !util.Contains(board.Admins, member.ProfileID) {
				board.Admins = append(board.Admins, member.ProfileID)
			}
		case "author":
			if !util.Contains(board.Authors, member.ProfileID) {
				board.Authors = append(board.Authors, member.ProfileID)
			}
		case "viewer":
			if !util.Contains(board.Viewers, member.ProfileID) {
				board.Viewers = append(board.Viewers, member.ProfileID)
			}
		case "subscriber":
			if !util.Contains(board.Subscribers, member.ProfileID) {
				board.Subscribers = append(board.Subscribers, member.ProfileID)
			}
		case "guest":
			if !util.Contains(board.Guests, member.ProfileID) {
				board.Guests = append(board.Guests, member.ProfileID)
			}
		}
		board.Blocked = util.Remove(board.Blocked, member.ProfileID)

		// update cache
		cacheKey := fmt.Sprintf("boards:%s", member.ProfileID)
		bp := permissions.GetBoardPermissionsNew(cacheKey, cache, board, member.ProfileID)
		if member.Role == "guest" {
			bp[boardID] = "viewer"
			cache.SetValue(cacheKey, bp.ToJSON())
		} else {
			bp[boardID] = member.Role
			cache.SetValue(cacheKey, bp.ToJSON())
		}
	}

	// update the board
	_, err = boardCollection.UpdateOne(context.TODO(), bson.M{"_id": boardObjID}, bson.M{"$set": board})
	if err != nil {
		return nil, errors.Wrap(err, "unable to update the board")
	}

	return util.SetResponse(nil, 1, "members unblocked successfully."), nil
}

func getBlockedMembers(db *mongodatabase.DBConfig, mysql *database.Database, cache *cache.Cache,
	profileService profile.Service, storageService storage.Service, profileID int, page, limit, boardID, search string,
) (map[string]interface{}, error) {
	var err error
	dbconn, err := db.New(consts.Board)
	if err != nil {
		return nil, err
	}
	boardCollection, boardsClient := dbconn.Collection, dbconn.Client
	defer boardsClient.Disconnect(context.TODO())

	profileIDStr := strconv.Itoa(profileID)
	isValid, err := permissions.CheckValidPermissions(profileIDStr, cache, boardCollection, boardID, []string{consts.Owner, consts.Admin}, false)
	if err != nil {
		return nil, err
	}
	if !isValid {
		return util.SetResponse(nil, 0, "User does not have access to the board."), nil
	}

	var board model.Board
	boardObjID, _ := primitive.ObjectIDFromHex(boardID)
	err = boardCollection.FindOne(context.TODO(), bson.M{"_id": boardObjID}).Decode(&board)
	if err != nil {
		return nil, err
	}

	members := []model.BoardMemberRole{}
	membersWithPermissions := []model.BoardMemberRole{}

	for _, member := range board.Blocked {
		m := model.BoardMemberRole{}
		m.Role = "blocked"
		m.ProfileID = member
		members = append(members, m)
	}

	var searchFilter string
	if search != "" {
		searchFilter = `AND
	(
		 CONCAT(firstName, '', lastName) LIKE '%` + search + `%'
		 OR screenName LIKE '%` + search + `%'
	)`
	}

	// get basic info from mysql
	for _, member := range members {
		stmt := `SELECT firstName, lastName, IFNULL(screenName, '') AS screenName, 
			IFNULL(photo, '') as photo FROM` + "`sidekiq-dev`.AccountProfile WHERE id = ? " + searchFilter
		profileID, _ := strconv.Atoi(member.ProfileID)
		err := mysql.Conn.Get(&member, stmt, profileID)
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			} else {
				return nil, errors.Wrap(err, "unable to find basic info")
			}
		} else {
			membersWithPermissions = append(membersWithPermissions, member)
		}
	}

	// pagination
	var pageNo int
	pageNo, _ = strconv.Atoi(page)
	limitInt, _ := strconv.Atoi(limit)
	subset := paginate(membersWithPermissions, pageNo, limitInt)

	if len(subset) == 0 {
		return util.SetPaginationResponse(nil, 0, 1, "No blocked members found."), nil
	}
	return util.SetPaginationResponse(subset, len(membersWithPermissions), 1, "All blocked members fetched successfully."), nil
}

func removeMembers(cache *cache.Cache, db *mongodatabase.DBConfig, profileService peoplerpc.AccountServiceClient, storageService storage.Service,
	profileID int, boardID string, membersToRemove []model.BoardMemberRoleRequest) (map[string]interface{}, error) {
	var err error
	dbconn, err := db.New(consts.Board)
	if err != nil {
		return nil, err
	}
	boardCollection, boardsClient := dbconn.Collection, dbconn.Client
	defer boardsClient.Disconnect(context.TODO())

	var board *model.Board
	boardObjID, _ := primitive.ObjectIDFromHex(boardID)
	filter := bson.M{"_id": boardObjID}
	err = boardCollection.FindOne(context.TODO(), filter).Decode(&board)
	if err != nil {
		return nil, err
	}

	var removedMembersNames []string

	profileIDStr := strconv.Itoa(profileID)
	ownerKey := fmt.Sprintf("boards:%s", profileIDStr)
	ownerBoardPermission := permissions.GetBoardPermissionsNew(ownerKey, cache, board, profileIDStr)
	role := ownerBoardPermission[boardID]
	fmt.Println("role: ", role)
	if role != "owner" && role != "admin" {
		return util.SetResponse(nil, 0, "User do not have permission to remove members"), nil
	}

	// remove from existing permission array
	for _, memberToBlock := range membersToRemove {
		switch memberToBlock.Role {
		case "admin":
			board.Admins = util.Remove(board.Admins, memberToBlock.ProfileID)
		case "author":
			board.Authors = util.Remove(board.Authors, memberToBlock.ProfileID)
		case "viewer":
			board.Viewers = util.Remove(board.Viewers, memberToBlock.ProfileID)
		case "subscriber":
			board.Subscribers = util.Remove(board.Subscribers, memberToBlock.ProfileID)
		case "guest":
			board.Guests = util.Remove(board.Guests, memberToBlock.ProfileID)
		}

		// remove its permissions from the cache pertaining to that board(boardID)
		cacheKey := fmt.Sprintf("boards:%s", memberToBlock.ProfileID)
		bp := permissions.GetBoardPermissionsNew(cacheKey, cache, board, memberToBlock.ProfileID)
		delete(bp, boardID)
		cache.SetValue(cacheKey, bp.ToJSON())

		// get name
		idInt, err := strconv.Atoi(memberToBlock.ProfileID)
		if err != nil {
			return nil, errors.Wrap(err, "unable to convert string to int")
		}
		// cp, err := profileService.FetchConciseProfile(idInt)
		// if err != nil {
		// 	return nil, errors.Wrap(err, "unable to fetch concise profile")
		// }
		cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(idInt)}
		cp, err := profileService.GetConciseProfile(context.TODO(), cpreq)
		if err != nil {
			return nil, errors.Wrap(err, "unable to fetch concise profile")
		}

		removedMembersNames = append(removedMembersNames, fmt.Sprintf("%s %s", cp.FirstName, cp.LastName))
	}

	// update the board
	_, err = boardCollection.UpdateOne(context.TODO(), filter, bson.M{"$set": board})
	if err != nil {
		return nil, errors.Wrap(err, "unable to update the board")
	}

	return util.SetResponse(removedMembersNames, 1, "Members removed successfully"), nil
}

func fetchSubBoardsByProfile(cache *cache.Cache, db *mongodatabase.DBConfig, mysql *database.Database, boardID string, profileID, limit int, publicOnly bool) (map[string]interface{}, error) {
	dbConn, err := db.New(consts.Board)
	if err != nil {
		return nil, err
	}
	boardCollection, boardClient := dbConn.Collection, dbConn.Client
	defer boardClient.Disconnect(context.TODO())

	profileIDStr := strconv.Itoa(profileID)
	isValid, err := permissions.CheckValidPermissions(profileIDStr, cache, boardCollection, boardID, []string{"blocked"}, true)
	if err != nil {
		return nil, err
	}
	if !isValid {
		return util.SetResponse(nil, 0, "User does not have access to the board."), nil
	}

	var curr *mongo.Cursor
	var findFilter primitive.M
	allSubBoards := make(map[string][]*model.Board)

	totalGoroutines := 4
	errChan := make(chan error)
	if publicOnly {
		totalGoroutines = 1
	}
	var res interface{}
	go func(errChan chan<- error) {
		defer util.RecoverGoroutinePanic(errChan)
		findFilter = bson.M{"visible": "PUBLIC", "parentID": boardID}
		subBoards, err := fetchSubBoardsByFilter(boardCollection, mysql, curr, findFilter, limit)
		if err != nil {
			errChan <- errors.Wrap(err, "unable to fetch sub-boards")
		}
		if len(subBoards) > 0 {
			if publicOnly {
				res = subBoards
			} else {
				allSubBoards["public"] = subBoards
			}
		} else {
			if publicOnly {
				res = nil
			} else {
				allSubBoards["public"] = nil
			}
		}
		errChan <- nil
	}(errChan)
	if !publicOnly {
		go func(errChan chan<- error) {
			defer util.RecoverGoroutinePanic(errChan)
			findFilter = bson.M{"owner": profileIDStr, "parentID": boardID}
			subBoards, err := fetchSubBoardsByFilter(boardCollection, mysql, curr, findFilter, limit)
			if err != nil {
				errChan <- errors.Wrap(err, "unable to fetch sub-boards")
			}

			if len(subBoards) > 0 {
				allSubBoards["private"] = subBoards
			} else {
				allSubBoards["private"] = nil
			}
			errChan <- nil
		}(errChan)
		go func(errChan chan<- error) {
			defer util.RecoverGoroutinePanic(errChan)
			findFilter = bson.M{"visible": "MEMBERS", "parentID": boardID}
			subBoards, err := fetchSubBoardsByFilter(boardCollection, mysql, curr, findFilter, limit)
			if err != nil {
				errChan <- errors.Wrap(err, "unable to fetch sub-boards")
			}

			if len(subBoards) > 0 {
				allSubBoards["members"] = subBoards
			} else {
				allSubBoards["members"] = nil
			}
			errChan <- nil
		}(errChan)
		go func(errChan chan<- error) {
			defer util.RecoverGoroutinePanic(errChan)
			// fetch profile connections
			dbConn, err := db.New("Connection")
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
				// fetch sub-boards filter
				findFilter = bson.M{"visible": "CONTACTS", "parentID": boardID, "owner": bson.M{"$in": connectionArr}}
				subBoards, err := fetchSubBoardsByFilter(boardCollection, mysql, curr, findFilter, limit)
				if err != nil {
					errChan <- errors.Wrap(err, "unable to fetch sub-boards")
				}
				if len(subBoards) > 0 {
					allSubBoards["contacts"] = subBoards
				} else {
					allSubBoards["contacts"] = nil
				}
			} else {
				allSubBoards["contacts"] = nil
			}
			errChan <- nil
		}(errChan)
	}
	for i := 0; i < totalGoroutines; i++ {
		if err := <-errChan; err != nil {
			return nil, errors.Wrap(err, "error fromfetchSubboardsByProfile go-routine")
		}
	}
	if publicOnly {
		return util.SetResponse(res, 1, "Sub-boards fetched successfully."), nil
	}
	return util.SetResponse(allSubBoards, 1, "Sub-boards fetched successfully."), nil
}

func fetchSubBoardsByFilter(boardCollection *mongo.Collection, mysql *database.Database, curr *mongo.Cursor, findFilter primitive.M, limit int) (subBoards []*model.Board, err error) {
	findOptions := options.Find()
	findOptions.SetSort(bson.M{"createDate": -1})
	if limit != 0 {
		opts := options.Find().SetLimit(int64(limit))
		curr, err = boardCollection.Find(context.TODO(), findFilter, opts, findOptions)
	} else {
		curr, err = boardCollection.Find(context.TODO(), findFilter, findOptions)
	}
	if err != nil {
		return nil, errors.Wrap(err, "unable to find sub boards")
	}
	err = curr.All(context.TODO(), &subBoards)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch sub boards")
	}
	// map owner profile
	errChan := make(chan error)
	for index := range subBoards {
		go func(i int, errChan chan<- error) {
			defer util.RecoverGoroutinePanic(errChan)

			ownerInfo := model.ConciseProfile{}
			stmt := `SELECT id, firstName, lastName,
							IFNULL(screenName, '') AS screenName,
							IFNULL(photo, '') AS photo FROM` + "`sidekiq-dev`.AccountProfile WHERE id = ?"
			itemOwner, _ := strconv.Atoi(subBoards[i].Owner)
			err = mysql.Conn.Get(&ownerInfo, stmt, itemOwner)
			if err != nil {
				errChan <- errors.Wrap(err, "unable to map profile info")
			}
			subBoards[i].OwnerInfo = &peoplerpc.ConciseProfileReply{
				Id:         int32(ownerInfo.Id),
				FirstName:  ownerInfo.FirstName,
				LastName:   ownerInfo.LastName,
				AccountID:  int32(ownerInfo.UserID),
				Photo:      ownerInfo.Photo,
				ScreenName: ownerInfo.ScreenName,
			}

			errChan <- nil
		}(index, errChan)
	}
	totalGoroutines := len(subBoards)
	for i := 0; i < totalGoroutines; i++ {
		if err := <-errChan; err != nil {
			return nil, errors.Wrap(err, "error fromfetchSubboardsByFilter go-routine")
		}
	}
	return
}

func boardAuth(db *mongodatabase.DBConfig, boardID primitive.ObjectID, password string) (map[string]interface{}, error) {
	var board *model.Board
	var status int
	var msg string
	dbconn, err := db.New(consts.Board)
	if err != nil {
		return nil, err
	}
	collection, client := dbconn.Collection, dbconn.Client
	defer client.Disconnect(context.TODO())
	filter := bson.M{"_id": boardID}
	err = collection.FindOne(context.TODO(), filter).Decode(&board)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch board")
	}
	if board.Password == password {
		status = 1
		msg = "Authentication successful"
	} else {
		status = 0
		msg = "Authentication failed. Password mismatch"
	}
	return util.SetResponse(nil, status, msg), nil
}

func getBoardFollowers(mysql *database.Database, db *mongodatabase.DBConfig, profileService peoplerpc.AccountServiceClient,
	storageService storage.Service, boardID, search, page, limit string,
) (map[string]interface{}, error) {
	// dbconn, err := db.New(consts.Board)
	// if err != nil {
	// 	return nil, err
	// }
	// boardCollection, boardClient := dbconn.Collection, dbconn.Client
	// defer boardClient.Disconnect(context.TODO())

	// boardObjID, _ := primitive.ObjectIDFromHex(boardID)
	// count, err := boardCollection.CountDocuments(context.TODO(), bson.M{"_id": boardObjID})
	// if err != nil {
	// 	return nil, errors.Wrap(err, "unable to find count")
	// }

	var followers []model.ConciseProfile
	var err error

	var searchFilter string
	if search != "" {
		searchFilter = ` AND (CONCAT(p.firstName, '', p.lastName) LIKE '%` + search + `%' OR p.screenName LIKE '%` + search + `%')`
	}

	stmt := `
	SELECT
		p.accountID,
		p.id,
		p.firstName,
		p.lastName,
		p.screenName
	FROM ` +
		" `sidekiq-dev`.AccountProfile as p " +
		` WHERE p.id IN (
			SELECT profileID FROM` + "`sidekiq-dev`.BoardsFollowed WHERE boardID = ?)" + searchFilter

	err = mysql.Conn.Select(&followers, stmt, boardID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch board followers")
	}

	// fetch profile image
	errCh := make(chan error)
	for idx := range followers {
		go func(errCh chan<- error, idx int) {
			defer util.RecoverGoroutinePanic(errCh)
			key := util.GetKeyForProfileImage(followers[idx].UserID, followers[idx].Id, "")
			fileName := fmt.Sprintf("%d.png", followers[idx].Id)
			file, err := storageService.GetUserFile(key, fileName)
			if err != nil {
				errCh <- errors.Wrap(err, "unable to fetch profile image from board followers")
				return
			}
			followers[idx].Photo = file.Filename
			errCh <- nil
		}(errCh, idx)
	}

	for i := 0; i < len(followers); i++ {
		if err := <-errCh; err != nil {
			log.Println("Error from goroutine: ", err)
		}
	}

	// pagination
	var pageNo int
	pageNo, _ = strconv.Atoi(page)
	limitInt, _ := strconv.Atoi(limit)
	subset := paginate2(followers, pageNo, limitInt)

	if len(subset) == 0 {
		return util.SetPaginationResponse(nil, 0, 1, "No followers found."), nil
	}
	return util.SetPaginationResponse(subset, len(followers), 1, "Followers fetched successfully."), nil
}

func getSharedBoards(db *mongodatabase.DBConfig, cache *cache.Cache, profileService peoplerpc.AccountServiceClient,
	storageService storage.Service, profileID int, search, page, limit, sortBy, orderBy string,
) (map[string]interface{}, error) {
	dbconn, err := db.New(consts.Board)
	if err != nil {
		return nil, err
	}
	boardCollection, boardClient := dbconn.Collection, dbconn.Client
	defer boardClient.Disconnect(context.TODO())

	profileIDStr := strconv.Itoa(profileID)

	filter := bson.M{
		"$and": bson.A{
			bson.M{
				"$or": bson.A{
					bson.M{"subscribers": profileIDStr},
					bson.M{"admins": profileIDStr},
					bson.M{"authors": profileIDStr},
					bson.M{"guests": profileIDStr},
				},
			},
			bson.M{"isDefaultBoard": false},
		},
	}

	var filterorderBy int64

	if sortBy == "" {
		sortBy = "createDate"
	}

	if orderBy == "" || strings.ToLower(orderBy) == "desc" {
		filterorderBy = -1
	} else {
		filterorderBy = 1
	}

	findOptions := options.Find()
	collation := &options.Collation{
		Locale:   "en", // Set your desired locale.
		Strength: 2,    // Strength 2 for case-insensitive.
	}
	findOptions.SetSort(bson.M{sortBy: filterorderBy})
	findOptions.SetLimit(int64(25))
	sortingOption := findOptions.SetSort(bson.M{sortBy: filterorderBy})
	findOptions = sortingOption.SetCollation(collation)

	var filteredRet, sharedBoards, finalResp []model.Board
	cur, err := boardCollection.Find(context.TODO(), filter, findOptions)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find shared boards of profile")
	}

	err = cur.All(context.TODO(), &sharedBoards)
	if err != nil {
		return nil, errors.Wrap(err, "unable to decode shared board cursor")
	}

	errChan := make(chan error)
	goRoutines := 0
	cps := make(map[int]*peoplerpc.ConciseProfileReply)

	// get owner's info
	for idx := range sharedBoards {
		goRoutines += 1
		go func(idx int, errChan chan<- error) {
			ownerIDint, err := strconv.Atoi(sharedBoards[idx].Owner)
			if err != nil {
				errChan <- errors.Wrap(err, "unable to convert string to int")
			}

			if ownerIDint != 0 {
				if val, ok := cps[ownerIDint]; !ok {
					// val, err = profileService.FetchConciseProfile(ownerIDint)
					// if err != nil {
					// 	errChan <- errors.Wrap(err, "unable to fetch concise profile")
					// }

					cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(ownerIDint)}
					val, err := profileService.GetConciseProfile(context.TODO(), cpreq)
					if err != nil {
						errChan <- errors.Wrap(err, "unable to fetch concise profile")
					}

					cps[ownerIDint] = val
				} else {
					sharedBoards[idx].OwnerInfo = val
				}
			}

			if search != "" {
				if fuzzy.Match(search, sharedBoards[idx].Title) || fuzzy.MatchFold(search, sharedBoards[idx].Title) {
					filteredRet = append(filteredRet, sharedBoards[idx])
				}
			} else {
				filteredRet = sharedBoards
			}

			if util.Contains(sharedBoards[idx].Likes, fmt.Sprint(profileID)) {
				sharedBoards[idx].IsLiked = true
			} else {
				sharedBoards[idx].IsLiked = false
			}
			sharedBoards[idx].TotalLikes = len(sharedBoards[idx].Likes)
			sharedBoards[idx].TotalComments = len(sharedBoards[idx].Comments)

			errChan <- nil
		}(idx, errChan)
	}

	// waiting for goroutines to finish
	for goRoutines != 0 {
		if err := <-errChan; err != nil {
			return nil, errors.Wrap(err, "error from go routine(board listing)")
		}
		goRoutines--
	}

	if strings.ToLower(sortBy) == "owner" {
		sort.Slice(sharedBoards, func(i, j int) bool {
			name1 := fmt.Sprintf("%s %s", sharedBoards[i].OwnerInfo.FirstName, sharedBoards[i].OwnerInfo.LastName)
			name2 := fmt.Sprintf("%s %s", sharedBoards[j].OwnerInfo.FirstName, sharedBoards[j].OwnerInfo.LastName)
			if strings.ToLower(orderBy) == "asc" {
				return strings.ToLower(name1) < strings.ToLower(name2)
			}
			return strings.ToLower(name1) > strings.ToLower(name2)
		})
	}

	// pagination
	var pageNo int
	pageNo, _ = strconv.Atoi(page)
	limitInt, _ := strconv.Atoi(limit)
	var data []interface{}
	for _, d := range filteredRet {
		data = append(data, d)
	}

	subset := util.PaginateFromArray(data, pageNo, limitInt)

	for _, d := range subset {
		goRoutines += 1
		go func(d interface{}, errChan chan<- error) {
			// get board role
			board := d.(model.Board)
			board.IsBoardFollower = util.Contains(board.Followers, profileIDStr)
			profileKey := fmt.Sprintf("boards:%s", profileIDStr)
			perms := permissions.GetBoardPermissionsNew(profileKey, cache, &board, profileIDStr)
			board.Role = perms[board.Id.Hex()]
			finalResp = append(finalResp, board)
			errChan <- nil
		}(d, errChan)
	}

	for goRoutines != 0 {
		if err := <-errChan; err != nil {
			return nil, errors.Wrap(err, "error from go routine(board listing)")
		}
		goRoutines--
	}

	return util.SetPaginationResponse(finalResp, len(sharedBoards), 1, "Shared boards fetched successfully"), nil
}

func isProfileBoardMember(db *mongodatabase.DBConfig, profileService profile.Service, boardID, profileID string) (bool, error) {
	dbconn, err := db.New(consts.Board)
	if err != nil {
		return false, err
	}
	boardCollection, boardClient := dbconn.Collection, dbconn.Client
	boardObjID, _ := primitive.ObjectIDFromHex(boardID)
	defer boardClient.Disconnect(context.TODO())

	filter := bson.M{
		"$and": bson.A{
			bson.M{"$or": bson.A{
				bson.M{"owner": profileID},
				bson.M{"admins": profileID},
				bson.M{"authors": profileID},
				bson.M{"subscribers": profileID},
			}},
			bson.M{"_id": boardObjID},
		},
	}
	count, err := boardCollection.CountDocuments(context.TODO(), filter)
	if err != nil {
		return false, err
	}
	if int(count) == 0 {
		return false, nil
	}
	return true, nil
}

func getBoardFileDetails(db *mongodatabase.DBConfig, profileService profile.Service, boardID, profileID string, fileOwnerDetals chan<- []map[string]interface{}, errChan chan<- error) {
	defer util.RecoverGoroutinePanic(errChan)
	// Connecting to File collection
	dbconn, err := db.New(consts.File)
	if err != nil {
		log.Println("error connecting file collection:", err)
		errChan <- errors.Wrap(err, "error in file owners ")
	}

	fileCollection, fileClient := dbconn.Collection, dbconn.Client
	defer fileClient.Disconnect(context.TODO())
	boardObjID, _ := primitive.ObjectIDFromHex(boardID)
	fileDetails := []map[string]interface{}{}

	// This query will get all things of caller's and visible:MEMBER things of other's
	filter := bson.M{
		// "$and": bson.A{
		// 	bson.M{"$or": bson.A{
		// 		bson.M{"owner": profileID},
		// 		bson.M{
		// 			"$and": bson.A{
		// 				bson.M{"owner": bson.M{"$ne": profileID}},
		// 				bson.M{"visible": "PUBLIC"},
		// 			},
		// 		},
		// 	}},
		// 	bson.M{"boardID": boardObjID},
		// },
		"boardID": boardObjID,
	}
	curr, err := fileCollection.Find(context.TODO(), filter)
	if err != nil {
		log.Println("error getting file owners:", err)
		errChan <- errors.Wrap(err, "error in file owners ")
	}
	curr.All(context.TODO(), &fileDetails)
	errChan <- nil
	fileOwnerDetals <- fileDetails
}

func getChildBoardDetails(db *mongodatabase.DBConfig, profileService profile.Service, boardID, profileID string, childBoardOwnerDetails chan<- []map[string]interface{}, errChan chan<- error) {
	defer util.RecoverGoroutinePanic(errChan)
	// Connecting to Board collection
	dbconn, err := db.New(consts.Board)
	if err != nil {
		log.Println("error connecting board collection:", err)
		errChan <- errors.Wrap(err, "error in child board owners ")
	}
	boardCollection, boardClient := dbconn.Collection, dbconn.Client
	defer boardClient.Disconnect(context.TODO())
	boardObjID, _ := primitive.ObjectIDFromHex(boardID)
	// ownerDetails := []map[string]interface{}{}

	data, err := getChildBoards(boardCollection, boardObjID)
	if err != nil {
		log.Println("error getting child board owners:", err)
		errChan <- errors.Wrap(err, "error in child board owners ")
	}

	errChan <- nil
	childBoardOwnerDetails <- data
}

func getBoardNotesDetails(db *mongodatabase.DBConfig, profileService profile.Service, boardID, profileID string, notesOwnerDetails chan<- []map[string]interface{}, errChan chan<- error) {
	defer util.RecoverGoroutinePanic(errChan)
	// Getting owners from Note
	dbconn, err := db.New(consts.Note)
	if err != nil {
		log.Println("error connecting note collection:", err)
		errChan <- errors.Wrap(err, "error in note owners ")
	}
	noteCollection, noteClient := dbconn.Collection, dbconn.Client
	defer noteClient.Disconnect(context.TODO())
	boardObjID, _ := primitive.ObjectIDFromHex(boardID)
	noteDetails := []map[string]interface{}{}

	// This query will get all things of caller's and visible:MEMBER notes of other's
	filter := bson.M{
		// "$and": bson.A{
		// 	bson.M{"$or": bson.A{
		// 		bson.M{"owner": profileID},
		// 		bson.M{
		// 			"$and": bson.A{
		// 				bson.M{"owner": bson.M{"$ne": profileID}},
		// 				bson.M{"visible": "PUBLIC"},
		// 			},
		// 		},
		// 	}},
		// 	bson.M{"boardID": boardObjID},
		// },
		"boardID": boardObjID,
	}

	curr, err := noteCollection.Find(context.TODO(), filter)
	if err != nil {
		log.Println("error getting note owner data:", err)
		errChan <- errors.Wrap(err, "error in note owners ")
	}
	curr.All(context.TODO(), &noteDetails)
	errChan <- nil
	notesOwnerDetails <- noteDetails
}

func getBoardTaskDetails(db *mongodatabase.DBConfig, profileService profile.Service, boardID, profileID string, taskOwnerDetails chan<- []map[string]interface{}, errChan chan<- error) {
	defer util.RecoverGoroutinePanic(errChan)
	// Getting owners from Note
	dbconn, err := db.New(consts.Task)
	if err != nil {
		log.Println("error connecting task collection:", err)
		errChan <- errors.Wrap(err, "error in task owners ")
	}
	taskCollection, taskClient := dbconn.Collection, dbconn.Client
	defer taskClient.Disconnect(context.TODO())
	boardObjID, _ := primitive.ObjectIDFromHex(boardID)
	taskDetails := []map[string]interface{}{}

	// This query will get all things of caller's and visible:MEMBER notes of other's
	filter := bson.M{
		// "$and": bson.A{
		// 	bson.M{"$or": bson.A{
		// 		bson.M{"owner": profileID},
		// 		bson.M{
		// 			"$and": bson.A{
		// 				bson.M{"owner": bson.M{"$ne": profileID}},
		// 				bson.M{"visible": "PUBLIC"},
		// 			},
		// 		},
		// 	}},
		// 	bson.M{"boardID": boardObjID},
		// },
		"boardID": boardObjID,
	}
	curr, err := taskCollection.Find(context.TODO(), filter)
	if err != nil {
		log.Println("error getting task owner data:", err)
		errChan <- errors.Wrap(err, "error in task owners ")
	}
	curr.All(context.TODO(), &taskDetails)
	errChan <- nil
	taskOwnerDetails <- taskDetails
}

func getBoardThingOwners(mysql *database.Database, db *mongodatabase.DBConfig, profileService profile.Service, boardID, profileID string, userID int) (map[string]interface{}, error) {
	// Checking if caller is a member of board or not
	isBoardMember, err := isProfileBoardMember(db, profileService, boardID, profileID)
	if err != nil {
		return nil, errors.Wrap(err, "error checking profile is board member")
	}
	if !isBoardMember {
		return util.SetResponse(nil, 0, "profile is not a member of board"), nil
	}

	// Getting board thing owners
	var allResults []map[string]interface{}

	filesDetails := make(chan []map[string]interface{})
	notesDetails := make(chan []map[string]interface{})
	childBoardOwnerDetails := make(chan []map[string]interface{})
	taskDetails := make(chan []map[string]interface{})

	errChan := make(chan error)

	go func() {
		util.RecoverGoroutinePanic(errChan)
		getBoardFileDetails(db, profileService, boardID, profileID, filesDetails, errChan)
	}()
	go func() {
		util.RecoverGoroutinePanic(errChan)
		getChildBoardDetails(db, profileService, boardID, profileID, childBoardOwnerDetails, errChan)
	}()
	go func() {
		util.RecoverGoroutinePanic(errChan)
		getBoardNotesDetails(db, profileService, boardID, profileID, notesDetails, errChan)
	}()
	go func() {
		util.RecoverGoroutinePanic(errChan)
		getBoardTaskDetails(db, profileService, boardID, profileID, taskDetails, errChan)
	}()

	for i := 0; i < 4; i++ {
		if err := <-errChan; err != nil {
			return nil, errors.Wrap(err, "error from go routine")
		}
	}
	var profileIDs []string

	allResults = append(allResults, <-filesDetails...)
	allResults = append(allResults, <-notesDetails...)
	allResults = append(allResults, <-taskDetails...)
	allResults = append(allResults, <-childBoardOwnerDetails...)

	// Removing duplicate entries
	keys := make(map[string]bool)

	for _, detail := range allResults {
		fmt.Println("THIS WAS OWNER", detail["owner"].(string))
		if _, value := keys[detail["owner"].(string)]; !value {
			keys[detail["owner"].(string)] = true
			profileIDs = append(profileIDs, detail["owner"].(string))
		}
	}
	conciseData, err := profileService.GetOwnerInfoUsingProfileIDs(profileIDs)
	if err != nil {
		return nil, errors.Wrap(err, "error from get owner info util")
	}

	return util.SetResponse(conciseData, 1, "owner details fetched successfully"), nil
}

func getBoardThingExt(mysql *database.Database, db *mongodatabase.DBConfig, profileService profile.Service, boardID, profileID string, userID int) (map[string]interface{}, error) {
	// Checking if the caller is a member of board or not
	isBoardMember, err := isProfileBoardMember(db, profileService, boardID, profileID)
	if err != nil {
		return nil, errors.Wrap(err, "error checking profile is board member")
	}
	if !isBoardMember {
		return util.SetResponse(nil, 0, "profile is not a member of board"), nil
	}

	filesDetails := make(chan []map[string]interface{})
	errChan := make(chan error)

	go func() {
		util.RecoverGoroutinePanic(errChan)
		getBoardFileDetails(db, profileService, boardID, profileID, filesDetails, errChan)
	}()

	if err := <-errChan; err != nil {
		return nil, errors.Wrap(err, "error getting file details for file type")
	}

	fd := <-filesDetails

	// Removing duplicate entries
	keys := make(map[string]bool)
	list := []string{}

	for _, result := range fd {
		fileExt := result["fileExt"].(string)
		if fileExt == "" {
			continue
		}
		ext := strings.Split(fileExt, ".")
		if _, value := keys[ext[1]]; !value {
			fmt.Println("i")
			keys[ext[1]] = true
			list = append(list, ext[1])
		}
	}

	return util.SetResponse(list, 1, "file details fetched successfully"), nil
}

func fetchBoardInfo(db *mongodatabase.DBConfig, boardID string, fields []string) (map[string]interface{}, error) {
	boardObjID, _ := primitive.ObjectIDFromHex(boardID)
	dbconn, err := db.New(consts.Board)
	if err != nil {
		return nil, err
	}
	boardCollection, boardsClient := dbconn.Collection, dbconn.Client
	defer boardsClient.Disconnect(context.TODO())
	var board map[string]interface{}
	err = boardCollection.FindOne(context.TODO(), bson.M{"_id": boardObjID}).Decode(&board)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find board")
	}

	res := make(map[string]interface{})
	res["_id"] = board["_id"]
	res["owner"] = board["owner"]
	res["isPassword"] = board["isPassword"]
	res["password"] = board["password"]
	res["title"] = board["title"]
	res["visible"] = board["visible"]
	if len(fields) > 0 {
		for _, f := range fields {
			if _, ok := res[f]; !ok {
				res[f] = board[f]
			}
		}
	}

	return res, nil
}

func updateBoardThingsTags(mongodb *mongodatabase.DBConfig, mysql *database.Database, profileID int, boardID, thingID string, tags []string) error {
	fmt.Println("tags: ", tags)
	dbconn, err := mongodb.New(consts.BoardThingsTags)
	if err != nil {
		return errors.Wrap(err, "unable to connect to "+consts.BoardThingsTags)
	}
	btt, bttClient := dbconn.Collection, dbconn.Client
	defer bttClient.Disconnect(context.TODO())

	boardObjID, err := primitive.ObjectIDFromHex(boardID)
	if err != nil {
		return errors.Wrap(err, "unable convert string to ObjectID")
	}

	thingObjID, err := primitive.ObjectIDFromHex(thingID)
	if err != nil {
		return errors.Wrap(err, "unable convert string to ObjectID")
	}

	filter := bson.M{"boardID": boardObjID}

	// get things tags
	var boardThingTags model.BoardThingTags
	err = btt.FindOne(context.TODO(), filter).Decode(&boardThingTags)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			fmt.Println("empty found")
		} else {
			return errors.Wrap(err, "unable to get thingsTags")
		}
	}

	if boardThingTags.Tags == nil {
		boardThingTags.Tags = make(map[string][]string)
	}
	boardThingTags.Tags[thingObjID.Hex()] = tags

	// modify the object and update in mongo
	opts := options.Update().SetUpsert(true)
	_, err = btt.UpdateOne(context.TODO(), filter, bson.M{"$set": boardThingTags}, opts)
	if err != nil {
		return errors.Wrap(err, "unable to update BoardThingsTags")
	}

	return nil
}

func getBoardThingsTags(mongo *mongodatabase.DBConfig, boardID string) ([]string, error) {
	dbconn, err := mongo.New(consts.BoardThingsTags)
	if err != nil {
		return nil, errors.Wrap(err, "unable to connect to "+consts.BoardThingsTags)
	}
	btt, bttClient := dbconn.Collection, dbconn.Client
	defer bttClient.Disconnect(context.TODO())

	boardObjID, err := primitive.ObjectIDFromHex(boardID)
	if err != nil {
		return nil, errors.Wrap(err, "unable convert string to ObjectID")
	}

	filter := bson.M{"boardID": boardObjID}

	var thingsTags map[string]interface{}
	err = btt.FindOne(context.TODO(), filter).Decode(&thingsTags)
	if err != nil {
		if strings.Contains(err.Error(), "mongo: no documents in result") {
			fmt.Println("No documents found.")
			return []string{}, nil
		}
		return nil, errors.Wrap(err, "unable to get thingsTags")
	}

	var tags []string
	if data, ok := thingsTags["tags"].(map[string]interface{}); ok {
		if data != nil {
			for _, t := range data {
				tt, ok := t.(primitive.A)
				if ok {
					for _, ttt := range tt {
						tags = append(tags, ttt.(string))
					}
				}
			}
		}
	}
	// remove dups
	tags = util.RemoveArrayDuplicate(tags)
	return tags, nil
}

func deleteFromBoardThingsTags(mongo *mongodatabase.DBConfig, boardID, thingID string) (map[string]interface{}, error) {
	dbconn, err := mongo.New(consts.BoardThingsTags)
	if err != nil {
		return nil, errors.Wrap(err, "unable to connect to "+consts.BoardThingsTags)
	}
	btt, bttClient := dbconn.Collection, dbconn.Client
	defer bttClient.Disconnect(context.TODO())

	boardObjID, err := primitive.ObjectIDFromHex(boardID)
	if err != nil {
		return nil, errors.Wrap(err, "unable convert string to ObjectID")
	}

	filter := bson.M{"boardID": boardObjID}

	var boardThingsTags model.BoardThingTags
	err = btt.FindOne(context.TODO(), filter).Decode(&boardThingsTags)
	if err != nil {
		return nil, errors.Wrap(err, "unable find board things tags")
	}

	if boardThingsTags.Tags[thingID] != nil {
		delete(boardThingsTags.Tags, thingID)
	}

	_, err = btt.UpdateOne(context.TODO(), filter, bson.M{"$set": boardThingsTags})
	if err != nil {
		return nil, errors.Wrap(err, "unable to update board things tags")
	}

	return nil, nil
}

func getBoardProfileRole(mongo *mongodatabase.DBConfig, cache *cache.Cache, boardID string, profileID string) (string, error) {
	dbconn, err := mongo.New(consts.Board)
	if err != nil {
		return "", errors.Wrap(err, "unable to connect to "+consts.BoardThingsTags)
	}
	boardColl, boardClient := dbconn.Collection, dbconn.Client
	defer boardClient.Disconnect(context.TODO())

	boardObjID, err := primitive.ObjectIDFromHex(boardID)
	if err != nil {
		return "", err
	}

	filter := bson.M{"_id": boardObjID}

	var boardObj *model.Board
	err = boardColl.FindOne(context.TODO(), filter).Decode(&boardObj)
	if err != nil {
		return "", errors.Wrap(err, "unable find board")
	}

	var ownerBoardPermission model.BoardPermission
	profileKey := fmt.Sprintf("boards:%s", profileID)
	ownerBoardPermission = permissions.GetBoardPermissionsNew(profileKey, cache, boardObj, profileID)
	role := ownerBoardPermission[boardObjID.Hex()]
	return role, nil
}

func getProfileTags(db *mongodatabase.DBConfig, profileID int) ([]string, error) {
	var profileTags []string

	// fetch all tags from mongo collections where owner is profileID
	errCh := make(chan error)
	filter := bson.M{"$and": bson.A{
		bson.M{"owner": strconv.Itoa(profileID)},
		bson.M{"state": "ACTIVE"},
	}}
	var mx sync.Mutex
	var goRoutines = 0
	var opts options.FindOptions
	opts.SetProjection(bson.M{"tags": 1})

	// fetch all Notes tags and append to tags
	goRoutines += 1
	go func(errChan chan<- error) {
		defer util.RecoverGoroutinePanic(errChan)
		var notes []map[string]interface{}
		noteConn, err := db.New(consts.Note)
		if err != nil {
			fmt.Println("unable to connect note")
			errCh <- errors.Wrap(err, "unable to connect note")
			return
		}
		noteCollection, noteClient := noteConn.Collection, noteConn.Client
		defer noteClient.Disconnect(context.TODO())

		curr, err := noteCollection.Find(context.TODO(), filter, &opts)
		if err != nil {
			fmt.Println("unable to fetch note tags")
			errCh <- errors.Wrap(err, "unable to fetch note tags")
			return
		}
		err = curr.All(context.TODO(), &notes)
		if err != nil {
			fmt.Println("error while note")
			errCh <- errors.Wrap(err, "failure in variable mapping")
			return
		}
		for i := range notes {
			mx.Lock()
			val, isOk := notes[i]["tags"]
			if isOk && val != nil {
				profileTags = append(profileTags, util.BsonAtoStrArr(notes[i]["tags"].(primitive.A))...)
			}
			mx.Unlock()
		}
		errCh <- nil
	}(errCh)

	// fetch all Tasks tags and append to tags
	goRoutines += 1
	go func(errChan chan<- error) {
		defer util.RecoverGoroutinePanic(errChan)
		var tasks []map[string]interface{}
		taskConn, err := db.New(consts.Task)
		if err != nil {
			fmt.Println("unable to connect task")
			errCh <- errors.Wrap(err, "unable to connect task")
			return
		}
		taskCollection, taskClient := taskConn.Collection, taskConn.Client
		defer taskClient.Disconnect(context.TODO())
		curr, err := taskCollection.Find(context.TODO(), filter)
		if err != nil {
			fmt.Println("unable to fetch task tags")
			errCh <- errors.Wrap(err, "unable to fetch task tags")
			return
		}
		err = curr.All(context.TODO(), &tasks)
		if err != nil {
			fmt.Println("error while task")
			errCh <- errors.Wrap(err, "failure in variable mapping")
			return
		}
		for i := range tasks {
			mx.Lock()
			fmt.Println(reflect.TypeOf(tasks[i]["tags"]))
			val, isOk := tasks[i]["tags"]
			if isOk && val != nil {
				profileTags = append(profileTags, util.BsonAtoStrArr(tasks[i]["tags"].(primitive.A))...)
				// profileTags = append(profileTags, val.([]string)...)
			}
			mx.Unlock()
		}
		errCh <- nil
	}(errCh)

	// fetch all Files tags and append to tags
	goRoutines += 1
	go func(errChan chan<- error) {
		defer util.RecoverGoroutinePanic(errChan)
		var files []map[string]interface{}
		fileConn, err := db.New(consts.File)
		if err != nil {
			fmt.Println("unable to connect file")
			errCh <- errors.Wrap(err, "unable to connect file")
			return
		}
		fileCollection, fileClient := fileConn.Collection, fileConn.Client
		defer fileClient.Disconnect(context.TODO())
		curr, err := fileCollection.Find(context.TODO(), filter)
		if err != nil {
			fmt.Println("error while file")
			fmt.Println("unable to fetch files tags")
			errCh <- errors.Wrap(err, "unable to fetch files tags")
			return
		}
		err = curr.All(context.TODO(), &files)
		if err != nil {
			errCh <- errors.Wrap(err, "failure in variable mapping")
			return
		}
		for i := range files {
			mx.Lock()
			fmt.Println(reflect.TypeOf(files[i]["tags"]))
			val, isOk := files[i]["tags"]
			if isOk && val != nil {
				profileTags = append(profileTags, util.BsonAtoStrArr(files[i]["tags"].(primitive.A))...)
				// profileTags = append(profileTags, val.([]string)...)
			}
			mx.Unlock()
		}
		errCh <- nil
	}(errCh)

	// fetch all Collection tags and append to tags
	goRoutines += 1
	go func(errChan chan<- error) {
		defer util.RecoverGoroutinePanic(errChan)
		var col []map[string]interface{}
		colConn, err := db.New(consts.Collection)
		if err != nil {
			fmt.Println("unable to connect collection")
			errCh <- errors.Wrap(err, "unable to connect collection")
			return
		}
		colCollection, colClient := colConn.Collection, colConn.Client
		defer colClient.Disconnect(context.TODO())
		curr, err := colCollection.Find(context.TODO(), filter)
		if err != nil {
			fmt.Println("unable to fetch collection tags")
			errCh <- errors.Wrap(err, "unable to fetch collection tags")
			return
		}
		err = curr.All(context.TODO(), &col)
		if err != nil {
			fmt.Println("error while Collection")
			errCh <- errors.Wrap(err, "failure in variable mapping")
			return
		}
		for i := range col {
			mx.Lock()
			val, isOk := col[i]["tags"]
			if isOk && val != nil {
				profileTags = append(profileTags, util.BsonAtoStrArr(col[i]["tags"].(primitive.A))...)
				// profileTags = append(profileTags, val.([]string)...)
			}
			mx.Unlock()
		}
		errCh <- nil
	}(errCh)

	// fetch all Board tags and append to tags
	goRoutines += 1
	go func(errChan chan<- error) {
		defer util.RecoverGoroutinePanic(errChan)
		var boards []*model.Board
		boardConn, err := db.New(consts.Board)
		if err != nil {
			fmt.Println("unable to connect board")
			errCh <- errors.Wrap(err, "unable to connect board")
			return
		}
		boardCollection, boardClient := boardConn.Collection, boardConn.Client
		defer boardClient.Disconnect(context.TODO())
		curr, err := boardCollection.Find(context.TODO(), filter)
		if err != nil {
			fmt.Println("unable to fetch note")
			errCh <- errors.Wrap(err, "unable to fetch board tags")
			return
		}
		err = curr.All(context.TODO(), &boards)
		if err != nil {
			fmt.Println("error while Board")
			errCh <- errors.Wrap(err, "failure in variable mapping")
			return
		}
		for i := range boards {
			mx.Lock()
			profileTags = append(profileTags, boards[i].Tags...)
			mx.Unlock()
		}
		errCh <- nil
	}(errCh)

	for i := 0; i < goRoutines; i++ {
		if err := <-errCh; err != nil {
			fmt.Printf("error occurred from go routine%v", err)
			return nil, err
		}
	}
	return profileTags, nil
}
