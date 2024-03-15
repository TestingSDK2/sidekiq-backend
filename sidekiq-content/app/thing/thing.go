package thing

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/member"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/post"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/storage"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/consts"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/database"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/helper"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/mongodatabase"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/util"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-models/model"
	"github.com/pkg/errors"

	// peoplerpc "github.com/sidekiq-people/proto/people"
	peoplerpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func getThingFromType(db *mongodatabase.DBConfig, thingId, thingType string) (map[string]interface{}, error) {
	var colType string

	switch strings.ToUpper(thingType) {
	case "BOARD":
		colType = consts.Board
	case "NOTE":
		colType = consts.Note
	case "FILE":
		colType = consts.File
	case "TASK":
		colType = consts.Task
	case "POST":
		colType = consts.Post
	case "COLLECTION":
		colType = consts.Collection
	default:
		return nil, errors.New("Unable to match thing type.")
	}

	dbconn, err := db.New(colType)
	if err != nil {
		return nil, err
	}

	thingColl, thingClient := dbconn.Collection, dbconn.Client
	defer thingClient.Disconnect(context.TODO())

	var thing map[string]interface{}
	thingObj, err := primitive.ObjectIDFromHex(thingId)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to objectID")
	}

	err = thingColl.FindOne(context.TODO(), bson.M{"_id": thingObj}).Decode(&thing)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find thing")
	}

	return thing, nil
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

func paginate(arr []model.ReactionList, pageNo, limit, total int) (ret []model.ReactionList) {
	var startIdx, endIdx int
	startIdx = limit * (pageNo - 1)
	endIdx = limit * pageNo

	if startIdx > total {
		return []model.ReactionList{}
	}

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

func likeThing(db *mongodatabase.DBConfig, thingID, thingType string, profileID int) (map[string]interface{}, error) {
	var colType string

	switch strings.ToUpper(thingType) {
	case "NOTE":
		colType = consts.Note
	case "FILE":
		colType = consts.File
	case "TASK":
		colType = consts.Task
	case "POST":
		colType = consts.Post
	case "COLLECTION":
		colType = consts.Collection
	case "BOARD":
		colType = consts.Board
	default:
		return nil, errors.New("Unable to match thing type.")
	}

	dbconn, err := db.New(colType)
	if err != nil {
		return nil, err
	}

	thingCollection, thingClient := dbconn.Collection, dbconn.Client
	defer thingClient.Disconnect(context.TODO())
	thingObjID, err := primitive.ObjectIDFromHex(thingID)
	if err != nil {
		return nil, err
	}
	filter := bson.M{
		"_id": thingObjID,
	}
	opts := options.FindOne().SetProjection(
		bson.M{
			"reactions": 1,
			"visible":   1,
			"likes":     1,
		})

	reactionDetails := make(map[string]interface{})
	isReactions := true

	err = thingCollection.FindOne(context.TODO(), filter, opts).Decode(&reactionDetails)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find thing")
	}

	_, ok := reactionDetails["reactions"]
	if ok {
		isReactions = reactionDetails["reactions"].(bool)
	}

	if !isReactions {
		return util.SetResponse(nil, 0, "Reactions are currently turned off for this thing."), nil
	}

	var thing model.ThingReactions
	err = thingCollection.FindOne(context.TODO(), bson.M{"_id": thingObjID}, opts).Decode(&thing)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find thing")
	}

	if util.Contains(thing.Likes, strconv.Itoa(profileID)) {
		return util.SetResponse(nil, 0, "you have already liked this "+strings.ToLower(thingType)), nil
	}

	thing.Likes = append(thing.Likes, strconv.Itoa(profileID))

	update := bson.M{"$set": bson.M{"likes": thing.Likes}}
	_, err = thingCollection.UpdateOne(context.TODO(), bson.M{"_id": thingObjID}, update)
	if err != nil {
		return nil, errors.Wrap(err, "unable to update like at mongo")
	}

	return util.SetResponse(nil, 1, "Like added successfully."), nil
}

func dislikeThing(db *mongodatabase.DBConfig, thingID, thingType string, profileID int) (map[string]interface{}, error) {
	var colType string

	switch strings.ToUpper(thingType) {
	case "NOTE":
		colType = consts.Note
	case "FILE":
		colType = consts.File
	case "TASK":
		colType = consts.Task
	case "POST":
		colType = consts.Post
	case "COLLECTION":
		colType = consts.Collection
	case "BOARD":
		colType = consts.Board
	default:
		return nil, errors.New("Unable to match thing type.")
	}

	dbconn, err := db.New(colType)
	if err != nil {
		return nil, err
	}

	thingCollection, thingClient := dbconn.Collection, dbconn.Client
	defer thingClient.Disconnect(context.TODO())

	thingObjID, err := primitive.ObjectIDFromHex(thingID)
	if err != nil {
		return nil, err
	}

	filter := bson.M{"_id": thingObjID}
	update := bson.M{"$pull": bson.M{"likes": strconv.Itoa(profileID)}}

	result, err := thingCollection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		return nil, errors.Wrap(err, "unable to update thing at mongo")
	}

	if result.ModifiedCount == 0 {
		return util.SetResponse(nil, 0, "Profile not found in likes."), nil
	}

	return util.SetResponse(nil, 1, "Like removed successfully."), nil
}

func addThingComment2(db *mongodatabase.DBConfig, thingID, thingType string, profileID int, comment string) (map[string]interface{}, error) {
	var colType string

	switch strings.ToUpper(thingType) {
	case consts.BoardType:
		colType = consts.Board
	case consts.PostType:
		colType = consts.Post
	case consts.NoteType:
		colType = consts.Note
	case consts.FileType:
		colType = consts.File
	case consts.TaskType:
		colType = consts.Task
	case consts.CollectionType:
		colType = consts.Collection
	default:
		return nil, errors.New("Unable to match thing type.")
	}

	dbconn, err := db.New(colType)
	if err != nil {
		return nil, err
	}
	thingCollection, thingClient := dbconn.Collection, dbconn.Client
	defer thingClient.Disconnect(context.TODO())

	thingObjID, err := primitive.ObjectIDFromHex(thingID)
	if err != nil {
		return nil, err
	}
	filter := bson.M{
		"_id": thingObjID,
	}
	opts := options.FindOne().SetProjection(
		bson.M{
			"reactions": 1,
			"visible":   1,
			"comments":  1,
		})

	var thing model.ThingReactions
	err = thingCollection.FindOne(context.TODO(), filter, opts).Decode(&thing)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find thing")
	}

	var newComment model.Comment
	newComment.ID = primitive.NewObjectID()
	newComment.ProfileID = profileID
	newComment.Message = comment
	newComment.CreateDate = time.Now()
	newComment.LastModifiedDate = time.Now()
	thing.Comments = append(thing.Comments, newComment)

	update := bson.M{"$set": bson.M{"comments": thing.Comments}}

	_, err = thingCollection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		return nil, errors.Wrap(err, "unable to update comment at mongo")
	}
	t := newComment.CreateDate
	newComment.AddedTime = t.Format("01-02-2006 15:04:05")
	t = newComment.LastModifiedDate
	newComment.EditTime = t.Format("01-02-2006 15:04:05")

	return util.SetResponse(newComment, 1, "Comment added successfully."), nil
}

func addThingComment(db *mongodatabase.DBConfig, thingID, thingType string, profileID int, comment string) (map[string]interface{}, bool, error) {

	var colType string

	switch strings.ToUpper(thingType) {
	case consts.BoardType:
		colType = consts.Board
	case consts.PostType:
		colType = consts.Post
	case consts.NoteType:
		colType = consts.Note
	case consts.FileType:
		colType = consts.File
	case consts.TaskType:
		colType = consts.Task
	case consts.CollectionType:
		colType = consts.Collection
	default:
		return nil, false, errors.New("Unable to match thing type.")
	}

	dbconn, err := db.New(colType)
	if err != nil {
		return nil, false, err
	}
	thingCollection, thingClient := dbconn.Collection, dbconn.Client
	defer thingClient.Disconnect(context.TODO())
	thingObjID, err := primitive.ObjectIDFromHex(thingID)
	if err != nil {
		return nil, false, err
	}
	filter := bson.M{
		"_id": thingObjID,
	}
	// var opts *options.FindOneOptions
	opts := options.FindOne().SetProjection(
		bson.M{
			"reactions": 1,
			"visible":   1,
			"comments":  1,
		})

	reactionDetails := make(map[string]interface{})
	isReactions := true

	err = thingCollection.FindOne(context.TODO(), filter, opts).Decode(&reactionDetails)
	if err != nil {
		return nil, false, errors.Wrap(err, "unable to find thing")
	}

	_, ok := reactionDetails["reactions"]
	if ok {
		isReactions = reactionDetails["reactions"].(bool)
	}

	if !isReactions {
		return util.SetResponse(nil, 0, "Reactions are currently turned off for this thing."), false, nil
	}

	var thing model.ThingReactions
	err = thingCollection.FindOne(context.TODO(), filter, opts).Decode(&thing)
	if err != nil {
		return nil, false, errors.Wrap(err, "unable to find thing")
	}

	var newComment model.Comment
	newComment.ID = primitive.NewObjectID()
	newComment.ProfileID = profileID
	newComment.Message = comment
	newComment.CreateDate = time.Now()
	newComment.LastModifiedDate = time.Now()
	thing.Comments = append(thing.Comments, newComment)
	update := bson.M{"$set": bson.M{"comments": thing.Comments}}
	_, err = thingCollection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		return nil, false, errors.Wrap(err, "unable to update comment at mongo")
	}
	t := newComment.CreateDate
	newComment.AddedTime = t.Format("01-02-2006 15:04:05")
	t = newComment.LastModifiedDate
	newComment.EditTime = t.Format("01-02-2006 15:04:05")
	return util.SetResponse(newComment, 1, "Comment added successfully."), true, nil
}

func editComment(db *mongodatabase.DBConfig, thingID, thingType, commentID string, payload map[string]string, profileID int) (map[string]interface{}, error) {
	var colType string

	switch strings.ToUpper(thingType) {
	case consts.BoardType:
		colType = consts.Board
	case consts.PostType:
		colType = consts.Post
	case consts.NoteType:
		colType = consts.Note
	case consts.FileType:
		colType = consts.File
	case consts.TaskType:
		colType = consts.Task
	case consts.CollectionType:
		colType = consts.Collection
	default:
		return nil, errors.New("Unable to match thing type.")
	}

	dbconn, err := db.New(colType)
	if err != nil {
		return nil, err
	}

	thingCollection, thingClient := dbconn.Collection, dbconn.Client
	defer thingClient.Disconnect(context.TODO())

	thingObjID, err := primitive.ObjectIDFromHex(thingID)
	if err != nil {
		return nil, err
	}

	commentObjID, err := primitive.ObjectIDFromHex(commentID)
	if err != nil {
		return nil, err
	}

	filter := bson.M{"_id": thingObjID, "comments._id": commentObjID}
	update := bson.M{"$set": bson.M{"comments.$.message": payload["comment"]}}

	result, err := thingCollection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find thing")
	}

	if result.ModifiedCount == 0 {
		return util.SetResponse(nil, 0, "Comment not found."), nil
	}

	return util.SetResponse(nil, 1, "Comment message updated successfully."), nil
}

func deleteComment(db *mongodatabase.DBConfig, thingID, thingType, commentID string, profileID int) (map[string]interface{}, error) {
	var colType string

	switch strings.ToUpper(thingType) {
	case consts.BoardType:
		colType = consts.Board
	case consts.PostType:
		colType = consts.Post
	case consts.NoteType:
		colType = consts.Note
	case consts.FileType:
		colType = consts.File
	case consts.TaskType:
		colType = consts.Task
	case consts.CollectionType:
		colType = consts.Collection
	default:
		return nil, errors.New("Unable to match thing type.")
	}

	dbconn, err := db.New(colType)
	if err != nil {
		return nil, err
	}

	thingCollection, thingClient := dbconn.Collection, dbconn.Client
	defer thingClient.Disconnect(context.TODO())
	thingObjID, err := primitive.ObjectIDFromHex(thingID)
	if err != nil {
		return nil, err
	}

	commentObjID, err := primitive.ObjectIDFromHex(commentID)
	if err != nil {
		return nil, err
	}

	filter := bson.M{"_id": thingObjID}
	update := bson.M{"$pull": bson.M{"comments": bson.M{"_id": commentObjID}}}

	result, err := thingCollection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		return nil, errors.Wrap(err, "unable to delete comment at mongo")
	}

	if result.ModifiedCount == 0 {
		return util.SetResponse(nil, 0, "Comment not found."), nil
	}

	return util.SetResponse(nil, 1, "Comment removed successfully."), nil
}

func fetchBookmarks(postService post.Service, storageService storage.Service, profileService peoplerpc.AccountServiceClient, db *database.Database, mongoDB *mongodatabase.DBConfig, userID, profileID, limit, page int, sortBy, orderBy, filterByThing string) (map[string]interface{}, error) {
	var stmt string
	bmConn, err := mongoDB.New(consts.Bookmark)
	if err != nil {
		return nil, errors.Wrap(err, "unable to establish connection with Bookmark")
	}

	bmColl, bmClient := bmConn.Collection, bmConn.Client
	defer bmClient.Disconnect(context.TODO())

	data := []*model.Bookmark{}
	var c int64
	// count bookmarks in mongo
	var filter bson.M
	if filterByThing != "" && strings.ToLower(filterByThing) != "all" {
		filter = bson.M{"profileID": profileID, "thingType": strings.ToUpper(filterByThing)}
	} else {
		filter = bson.M{"profileID": profileID}
	}

	c, err = bmColl.CountDocuments(context.TODO(), filter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find bookmark's count")
	}

	if c == 0 {
		return util.SetPaginationResponse(data, int(c), 1, "All bookmarks fetched successfully"), nil
	}

	// select bookmarks from mongo based off setlimit and setskip of a profile
	var curr *mongo.Cursor
	var filterorderBy int64

	findOptions := options.Find()

	if orderBy == "" || strings.ToUpper(orderBy) == "DESC" {
		filterorderBy = -1
	} else {
		filterorderBy = 1
	}

	if sortBy != "" {
		findOptions.SetSort(bson.M{sortBy: filterorderBy})
	} else {
		findOptions.SetSort(bson.M{"createDate": -1})
	}

	if limit > 0 && page > 0 {
		offset := (page - 1) * limit
		findOptions.SetSkip(int64(offset))
		findOptions.SetLimit(int64(limit))
		curr, err = bmColl.Find(context.TODO(), filter, findOptions)
		if err != nil {
			return nil, errors.Wrap(err, "unable to find profile's bookmarks")
		}
	}
	defer curr.Close(context.TODO())

	err = curr.All(context.TODO(), &data)
	if err != nil {
		return nil, errors.Wrap(err, "unable to unpack profile's bookmarks")
	}

	var mx sync.Mutex
	errCh := make(chan error)
	// fetch profiles
	go func(data []*model.Bookmark, errChan chan<- error) {
		defer util.RecoverGoroutinePanic(errChan)
		for _, v := range data {
			profile := &model.NewConciseProfile{}
			stmt = `SELECT firstName, lastName,
			IFNULL(screenName, '') AS screenName,
			IFNULL(photo, '') AS photo FROM` + "`sidekiq-dev`.AccountProfile WHERE id = ?"
			mx.Lock()
			err = db.Conn.Get(profile, stmt, v.OwnerID)
			mx.Unlock()
			if err != nil {
				errCh <- errors.Wrap(err, "unable to fetch profiles")
				return
			}
			if profile.Photo == "" {
				photo, err := getProfileImage(db, storageService, 0, v.OwnerID)
				if err != nil {
					fmt.Println("photo not found for profile", v.OwnerID)
				}
				profile.Photo = photo
			}
			v.NewConciseProfile = *profile
		}
		errCh <- nil
		// return
	}(data, errCh)
	// fetch tags
	go func(data []*model.Bookmark, errChan chan<- error) {
		defer util.RecoverGoroutinePanic(errChan)
		noteConn, err := mongoDB.New(consts.Note)
		if err != nil {
			fmt.Println("unable to connect note")
			errCh <- errors.Wrap(err, "unable to connect note")
			return
		}
		taskConn, err := mongoDB.New(consts.Task)
		if err != nil {
			fmt.Println("unable to connect task")
			errCh <- errors.Wrap(err, "unable to connect task")
			return
		}
		fileConn, err := mongoDB.New(consts.File)
		if err != nil {
			fmt.Println("unable to connect file")
			errCh <- errors.Wrap(err, "unable to connect file")
			return
		}
		boardConn, err := mongoDB.New("Board")
		if err != nil {
			fmt.Println("unable to connect board")
			errCh <- errors.Wrap(err, "unable to connect board")
			return
		}
		colConn, err := mongoDB.New(consts.Collection)
		if err != nil {
			fmt.Println("unable to connect board")
			errCh <- errors.Wrap(err, "unable to connect board")
			return
		}

		postConn, err := mongoDB.New(consts.Post)
		if err != nil {
			fmt.Println("unable to connect post")
			errCh <- errors.Wrap(err, "unable to connect post")
			return
		}

		noteCollection, noteClient := noteConn.Collection, noteConn.Client
		defer noteClient.Disconnect(context.TODO())
		fileCollection, fileClient := fileConn.Collection, fileConn.Client
		defer fileClient.Disconnect(context.TODO())
		taskCollection, taskClient := taskConn.Collection, taskConn.Client
		defer taskClient.Disconnect(context.TODO())
		boardCollection, boardClient := boardConn.Collection, boardConn.Client
		defer boardClient.Disconnect(context.TODO())
		colCollection, colClient := colConn.Collection, colConn.Client
		defer colClient.Disconnect(context.TODO())

		postCollection, postClient := postConn.Collection, postConn.Client
		defer postClient.Disconnect(context.TODO())

		var objID primitive.ObjectID
		var filter primitive.D

		var task map[string]interface{}
		var note map[string]interface{}
		var file model.UploadedFile
		var board model.Board
		var col model.Collection
		var post model.Post

		for _, v := range data {

			if v.Flagged {
				continue
			}

			objID, err = primitive.ObjectIDFromHex(v.ThingID)
			if err != nil {
				fmt.Println("unable to convert to objID")
				errCh <- errors.Wrap(err, "unable to convert to objID")
				continue
			}
			filter = bson.D{{Key: "_id", Value: objID}}
			if strings.ToLower(v.ThingType) == "note" {
				err = noteCollection.FindOne(context.TODO(), filter).Decode(&note)
				if err != nil {
					flagBookmarkForDelete(mongoDB, profileID, v.ThingID, time.Now())
					fmt.Println("unable to fetch note")
					errCh <- errors.Wrap(err, "unable to fetch note")
					continue
				}

				err = postCollection.FindOne(context.TODO(), bson.M{"_id": note["postID"]}).Decode(&post)
				if err != nil {
					flagBookmarkForDelete(mongoDB, profileID, note["postID"].(primitive.ObjectID).Hex(), time.Now())
					fmt.Println("unable to fetch post")
					errCh <- errors.Wrap(err, "unable to fetch post")
					continue
				}

				mx.Lock()
				v.PostID = note["postID"].(primitive.ObjectID).Hex()
				v.BoardID = post.BoardID.Hex()
				v.ThingTitle = note["title"].(string)
				v.ThingUploadDate = note["createDate"].(primitive.DateTime).Time().String()

				if note["likes"] != nil {
					note["totalLikes"] = len(note["likes"].(primitive.A))

					var likes []string
					for _, value := range note["likes"].(primitive.A) {
						likes = append(likes, value.(string))
					}

					if util.Contains(likes, fmt.Sprint(profileID)) {
						note["isLiked"] = true
					} else {
						note["isLiked"] = false
					}
				} else {
					note["totalLikes"] = 0
					note["isLiked"] = false
				}

				if note["comments"] != nil {
					note["totalComments"] = len(note["comments"].(primitive.A))
				} else {
					note["totalComments"] = 0
				}

				v.Things = note
				mx.Unlock()
			} else if strings.ToLower(v.ThingType) == "task" {
				err = taskCollection.FindOne(context.TODO(), filter).Decode(&task)
				if err != nil {
					flagBookmarkForDelete(mongoDB, profileID, v.ThingID, time.Now())
					fmt.Println("unable to fetch task")
					errCh <- errors.Wrap(err, "unable to fetch task")
					continue
				}

				err = postCollection.FindOne(context.TODO(), bson.M{"_id": task["postID"]}).Decode(&post)
				if err != nil {
					flagBookmarkForDelete(mongoDB, profileID, task["postID"].(primitive.ObjectID).Hex(), time.Now())
					fmt.Println("unable to fetch post")
					errCh <- errors.Wrap(err, "unable to fetch post")
					continue
				}

				mx.Lock()
				v.PostID = task["postID"].(primitive.ObjectID).Hex()
				v.BoardID = post.BoardID.Hex()
				v.ThingTitle = task["title"].(string)

				assignedMemberInfo, err := member.GetAssignedMemberInfo(task, profileService)
				if err != nil {
					errCh <- errors.Wrap(err, "unable to fetch GetAssignedMemberInfo")
					continue
				}

				task["assignedMemberInfo"] = assignedMemberInfo

				reporterInfo, err := member.GetReporterInfo(task, profileService)
				if err != nil {
					errCh <- errors.Wrap(err, "unable to fetch reporterInfo")
					continue
				}
				task["reporterInfo"] = reporterInfo

				if task["likes"] != nil {
					task["totalLikes"] = len(task["likes"].(primitive.A))

					var likes []string
					for _, value := range task["likes"].(primitive.A) {
						likes = append(likes, value.(string))
					}

					if util.Contains(likes, fmt.Sprint(profileID)) {
						task["isLiked"] = true
					} else {
						task["isLiked"] = false
					}
				} else {
					task["totalLikes"] = 0
					task["isLiked"] = false
				}

				if task["comments"] != nil {
					task["totalComments"] = len(task["comments"].(primitive.A))
				} else {
					task["totalComments"] = 0
				}
				v.ThingUploadDate = task["createDate"].(primitive.DateTime).Time().String()
				v.Things = task

				mx.Unlock()
			} else if strings.ToLower(v.ThingType) == "file" {
				err = fileCollection.FindOne(context.TODO(), filter).Decode(&file)
				if err != nil {
					flagBookmarkForDelete(mongoDB, profileID, v.ThingID, time.Now())
					fmt.Println("unable to fetch file")
					errCh <- errors.Wrap(err, "unable to fetch file")
					continue
				}

				err = postCollection.FindOne(context.TODO(), bson.M{"_id": file.PostID}).Decode(&post)
				if err != nil {
					flagBookmarkForDelete(mongoDB, profileID, file.PostID.Hex(), time.Now())
					fmt.Println("unable to fetch post")
					errCh <- errors.Wrap(err, "unable to fetch post")
					continue
				}

				mx.Lock()

				v.Tags = file.Tags
				v.ThingTitle = file.Title
				v.PostID = file.PostID.Hex()
				v.BoardID = post.BoardID.Hex()
				v.ThingUploadDate = file.CreateDate.String()

				file.TotalLikes = len(file.Likes)
				if util.Contains(file.Likes, fmt.Sprint(profileID)) {
					file.IsLiked = true
				} else {
					file.IsLiked = false
				}
				file.TotalComments = len(file.Comments)

				err = boardCollection.FindOne(context.TODO(), bson.M{"_id": file.BoardID}).Decode(&board)
				if err != nil {
					continue
				}

				fileOwner, _ := strconv.Atoi(file.Owner)
				// ownerInfo, err := profileService.FetchConciseProfile(fileOwner)

				cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(fileOwner)}
				ownerInfo, err := profileService.GetConciseProfile(context.TODO(), cpreq)
				if err != nil {
					continue
				}

				ownerInfo.Id = int32(fileOwner)
				file.OwnerInfo = ownerInfo

				boardOwner, _ := strconv.Atoi(board.Owner)
				// boardownerInfo, err := profileService.FetchConciseProfile(boardOwner)
				// if err != nil {
				// 	continue
				// }

				cpreq = &peoplerpc.ConciseProfileRequest{ProfileId: int32(boardOwner)}
				boardownerInfo, err := profileService.GetConciseProfile(context.TODO(), cpreq)
				if err != nil {
					continue
				}

				key := ""
				if file.CollectionID == primitive.NilObjectID {
					key = util.GetKeyForBoardPostMedia(int(boardownerInfo.AccountID), int(boardownerInfo.Id), file.BoardID.Hex(), file.PostID.Hex(), "")
				} else {
					key = util.GetKeyForPostCollectionMedia(int(boardownerInfo.AccountID), int(boardownerInfo.Id), file.BoardID.Hex(), file.PostID.Hex(), file.CollectionID.Hex(), "")
				}

				fileName := fmt.Sprintf("%s%s", file.Id.Hex(), file.FileExt)
				f, err := storageService.GetUserFile(key, fileName)
				if err != nil {
					continue
				}
				file.URL = f.Filename

				thumbKey := ""
				if file.CollectionID == primitive.NilObjectID {
					thumbKey = util.GetKeyForBoardPostMedia(int(boardownerInfo.AccountID), int(boardownerInfo.Id), file.BoardID.Hex(), file.PostID.Hex(), "thumbs")
				} else {
					thumbKey = util.GetKeyForPostCollectionMedia(int(boardownerInfo.AccountID), int(boardownerInfo.Id), file.BoardID.Hex(), file.PostID.Hex(), file.CollectionID.Hex(), "thumbs")
				}

				thumbfileName := file.Id.Hex() + ".png"
				file.Thumbs, err = helper.GetThumbnails(storageService, thumbKey, thumbfileName, []string{})
				if err != nil {
					file.Thumbs = model.Thumbnails{}
				}

				v.Things = file
				mx.Unlock()
			} else if strings.ToLower(v.ThingType) == "board" {
				err = boardCollection.FindOne(context.TODO(), filter).Decode(&board)
				if err != nil {
					flagBookmarkForDelete(mongoDB, profileID, v.ThingID, time.Now())
					fmt.Println("unable to fetch board")
					errCh <- errors.Wrap(err, "unable to fetch board")
					continue
				}
				mx.Lock()
				v.Tags = board.Tags
				v.ThingTitle = board.Title
				v.PostID = ""
				v.BoardID = board.Id.Hex()
				v.ThingUploadDate = board.CreateDate.String()

				board.TotalLikes = len(board.Likes)
				if util.Contains(board.Likes, fmt.Sprint(profileID)) {
					board.IsLiked = true
				} else {
					board.IsLiked = false
				}
				board.TotalComments = len(board.Comments)

				v.Things = board
				mx.Unlock()
			} else if strings.ToLower(v.ThingType) == "collection" {
				err = colCollection.FindOne(context.TODO(), filter).Decode(&col)
				if err != nil {
					flagBookmarkForDelete(mongoDB, profileID, v.ThingID, time.Now())
					fmt.Println("unable to fetch collection")
					errCh <- errors.Wrap(err, "unable to fetch collection")
					continue
				}

				err = postCollection.FindOne(context.TODO(), bson.M{"_id": col.PostID}).Decode(&post)
				if err != nil {
					flagBookmarkForDelete(mongoDB, profileID, col.PostID.Hex(), time.Now())
					errCh <- errors.Wrap(err, "unable to find post")
					continue
				}
				mx.Lock()
				v.Tags = col.Tags
				v.ThingTitle = col.Title
				v.PostID = post.Id.Hex()
				v.BoardID = post.BoardID.Hex()
				v.ThingUploadDate = col.CreateDate.String()

				col.TotalLikes = len(col.Likes)
				if util.Contains(col.Likes, fmt.Sprint(profileID)) {
					col.IsLiked = true
				} else {
					col.IsLiked = false
				}
				col.TotalComments = len(col.Comments)

				colMap := col.ToMap()

				var files []*model.UploadedFile
				collectionID := objID
				curr, err := fileCollection.Find(context.TODO(), bson.M{"collectionID": collectionID})
				if err != nil {
					continue
				}

				err = curr.All(context.TODO(), &files)
				if err != nil {
					continue
				}

				for fileidx := range files {

					err = boardCollection.FindOne(context.TODO(), bson.M{"_id": files[fileidx].BoardID}).Decode(&board)
					if err != nil {
						continue
					}

					fileOwner, _ := strconv.Atoi(files[fileidx].Owner)
					// ownerInfo, err := profileService.FetchConciseProfile(fileOwner)

					cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(fileOwner)}
					ownerInfo, err := profileService.GetConciseProfile(context.TODO(), cpreq)
					if err != nil {
						continue
					}

					ownerInfo.Id = int32(fileOwner)
					files[fileidx].OwnerInfo = ownerInfo

					boardOwner, _ := strconv.Atoi(board.Owner)
					// boardownerInfo, err := profileService.FetchConciseProfile(boardOwner)

					cpreq = &peoplerpc.ConciseProfileRequest{ProfileId: int32(boardOwner)}
					boardownerInfo, err := profileService.GetConciseProfile(context.TODO(), cpreq)
					if err != nil {
						continue
					}

					key := util.GetKeyForPostCollectionMedia(int(boardownerInfo.AccountID), int(boardownerInfo.Id), files[fileidx].BoardID.Hex(), files[fileidx].PostID.Hex(), collectionID.Hex(), "")
					fileName := fmt.Sprintf("%s%s", files[fileidx].Id.Hex(), files[fileidx].FileExt)
					f, err := storageService.GetUserFile(key, fileName)
					if err != nil {
						continue
					}
					files[fileidx].URL = f.Filename

					thumbKey := util.GetKeyForPostCollectionMedia(int(boardownerInfo.AccountID), int(boardownerInfo.Id), files[fileidx].BoardID.Hex(), files[fileidx].PostID.Hex(), collectionID.Hex(), "thumbs")
					thumbfileName := files[fileidx].Id.Hex() + ".png"
					thumbs, err := helper.GetThumbnails(storageService, thumbKey, thumbfileName, []string{})
					if err != nil {
						thumbs = model.Thumbnails{}
					}

					files[fileidx].Thumbs = thumbs
					// reaction count
					if util.Contains(files[fileidx].Likes, fmt.Sprint(profileID)) {
						files[fileidx].IsLiked = true
					} else {
						files[fileidx].IsLiked = false
					}
					files[fileidx].TotalComments = len(files[fileidx].Comments)
					files[fileidx].TotalLikes = len(files[fileidx].Likes)
				}

				colMap["files"] = files

				v.Things = colMap
				mx.Unlock()
			} else if strings.ToLower(v.ThingType) == "post" {
				objectID, err := primitive.ObjectIDFromHex(v.ThingID)
				if err != nil {
					continue
				}
				err = postCollection.FindOne(context.TODO(), bson.M{"_id": objectID}).Decode(&post)
				if err != nil {
					flagBookmarkForDelete(mongoDB, profileID, v.ThingID, time.Now())
					fmt.Println("unable to fetch post")
					errCh <- errors.Wrap(err, "unable to fetch post")
					continue
				}

				mx.Lock()
				v.Tags = post.Tags
				v.BoardID = post.BoardID.Hex()
				v.PostID = post.Id.Hex()
				v.ThingUploadDate = post.CreateDate.String()

				post.TotalLikes = len(post.Likes)
				if util.Contains(post.Likes, fmt.Sprint(profileID)) {
					post.IsLiked = true
				} else {
					post.IsLiked = false
				}
				post.TotalComments = len(post.Comments)

				postmap := post.ToMap()

				ret, err := postService.GetFirstPostThing(v.ThingID, v.BoardID, profileID)
				if err != nil {
					continue
				}
				postmap["things"] = ret

				v.Things = postmap
				mx.Unlock()

			} else {
				fmt.Println("thing type undefined")
				continue
			}
		}
		errCh <- nil
	}(data, errCh)
	for i := 0; i < 2; i++ {
		if err := <-errCh; err != nil {
			fmt.Printf("Error occurred: %v", err)
		}
	}
	return util.SetPaginationResponse(data, int(c), 1, "All bookmarks fetched successfully"), nil
}

func addBookmark(db *database.Database, mongo *mongodatabase.DBConfig, payload model.Bookmark) (map[string]interface{}, error) {
	dbconn, err := mongo.New(consts.Bookmark)
	if err != nil {
		return nil, err
	}
	bookmarkColl, bookmarkClient := dbconn.Collection, dbconn.Client
	defer bookmarkClient.Disconnect(context.TODO())

	bookmark := true

	// check if the thing can be bookmarked or not
	thing, err := getThingFromType(mongo, payload.ThingID, payload.ThingType)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find thing from type")
	}

	if strings.ToUpper(payload.ThingType) == "POST" || strings.ToUpper(payload.ThingType) == "BOARD" {
		if v, ok := thing["bookmark"]; !ok {
			bookmark = false
		} else if v.(bool) {
			bookmark = true
		}
	}

	fmt.Println("to bookmark: ", bookmark)

	// check if bookmark already exists
	filter := bson.M{"profileID": payload.ProfileID, "thingID": payload.ThingID}
	count, err := bookmarkColl.CountDocuments(context.TODO(), filter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find count for bookmark")
	}
	if (count) > 0 {
		return util.SetResponse(nil, 0, "Bookmark already present"), nil
	}

	if bookmark {
		y, m, d := time.Now().Date()
		payload.ThingType = strings.ToUpper(payload.ThingType)
		payload.CreateDate = time.Now()
		payload.LastViewedDate = fmt.Sprintf("%v-%d-%v", y, int(m), d)
		payload.ThingLocation, err = fetchThingLocationOnBoard(mongo, payload.BoardID)
		payload.ID = primitive.NewObjectID()
		if err != nil {
			return nil, errors.Wrap(err, "unable to find thing location on board")
		}

		_, err = bookmarkColl.InsertOne(context.TODO(), payload)
		if err != nil {
			return nil, errors.Wrap(err, "unable to insert bookmark")
		}
	} else {
		return util.SetResponse(nil, 0, "The thing cannot be set as a bookmark."), nil

	}

	return util.SetResponse(payload.ID.Hex(), 1, "Bookmark added successfully"), nil
}

func flagBookmarkForDelete(mongo *mongodatabase.DBConfig, profileID int, id string, deleteDate time.Time) (map[string]interface{}, error) {
	dbconn, err := mongo.New(consts.Bookmark)
	if err != nil {
		return nil, err
	}
	bookmarkColl, bookmarkClient := dbconn.Collection, dbconn.Client
	defer bookmarkClient.Disconnect(context.TODO())

	filter := bson.M{"thingID": id}

	// count
	count, err := bookmarkColl.CountDocuments(context.TODO(), filter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find bookmark's count")
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
			return nil, errors.Wrap(err, "Unable to update bookmark documents")
		}

		fmt.Printf("Updated %v bookmark documents\n", updateResult.ModifiedCount)
	}

	dbconnRecent, err := mongo.New(consts.Recent)
	if err != nil {
		return nil, err
	}
	recentColl, recentClient := dbconnRecent.Collection, dbconnRecent.Client
	defer recentClient.Disconnect(context.TODO())

	thingObj, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	recentfilter := bson.M{"thingID": thingObj}

	// count
	recentcount, err := recentColl.CountDocuments(context.TODO(), recentfilter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find recent's count")
	}

	if int(recentcount) > 0 {

		// Perform the update
		updateResult, err := recentColl.DeleteMany(context.TODO(), recentfilter)
		if err != nil {
			return nil, errors.Wrap(err, "Unable to delete recent documents")
		}

		fmt.Printf("Deleted %v recent documents\n", updateResult.DeletedCount)
	}

	return util.SetResponse(nil, 1, ""), nil
}

func deleteBookmark(mongo *mongodatabase.DBConfig, profileID int, id string) (map[string]interface{}, error) {
	dbconn, err := mongo.New(consts.Bookmark)
	if err != nil {
		return nil, err
	}
	bookmarkColl, bookmarkClient := dbconn.Collection, dbconn.Client
	defer bookmarkClient.Disconnect(context.TODO())

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to ObjectID")
	}

	_, err = bookmarkColl.DeleteOne(context.TODO(), bson.M{"_id": objID})
	if err != nil {
		return nil, errors.Wrap(err, "unable to delete bookmark")
	}

	return util.SetResponse(nil, 1, "Bookmark deleted successfully"), nil
}

func deleteAllBookmarks(mongo *mongodatabase.DBConfig, profileID int) (map[string]interface{}, error) {
	dbconn, err := mongo.New(consts.Bookmark)
	if err != nil {
		return nil, err
	}
	bookmarkColl, bookmarkClient := dbconn.Collection, dbconn.Client
	defer bookmarkClient.Disconnect(context.TODO())

	_, err = bookmarkColl.DeleteMany(context.TODO(), bson.M{"profileID": profileID})
	if err != nil {
		return nil, errors.Wrap(err, "unable to delete bookmarks of the profile")
	}

	return util.SetResponse(nil, 1, "All bookmarks of the profile deleted successfully"), nil
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
		fmt.Printf("%s - %s", info.ParentID, info.Title)
		if info.ParentID == "" {
			location += info.Title
		} else {
			location += info.Title + "/"
		}

	}
	return
}

func fetchReactions(db *mongodatabase.DBConfig, profileService peoplerpc.AccountServiceClient, storageService storage.Service,
	thingID, thingType, reactionType string, profileID, limit, page int,
) (map[string]interface{}, error) {
	var colType string

	switch strings.ToUpper(thingType) {
	case "NOTE":
		colType = consts.Note
	case "FILE":
		colType = consts.File
	case "TASK":
		colType = consts.Task
	case "POST":
		colType = consts.Post
	case "COLLECTION":
		colType = consts.Collection
	case "BOARD":
		colType = consts.Board
	default:
		return nil, errors.New("Unable to match thing type.")
	}

	dbconn, err := db.New(colType)
	if err != nil {
		return nil, err
	}

	thingCollection, thingClient := dbconn.Collection, dbconn.Client
	defer thingClient.Disconnect(context.TODO())
	thingObjID, err := primitive.ObjectIDFromHex(thingID)
	if err != nil {
		return nil, err
	}

	filter := bson.M{
		"_id": thingObjID,
	}

	type allComments struct {
		Comment []model.Comment `json:"comments" bson:"comments"`
	}

	var comments allComments

	type allLikes struct {
		Likes []string `json:"likes" bson:"likes"`
	}
	var likes allLikes
	var response []model.ReactionList
	var profileInfo model.ReactionList
	var opts *options.FindOneOptions
	var total int
	// pagination
	offset := (page - 1) * limit
	findOptions := options.Find()
	findOptions.SetSkip(int64(offset))
	findOptions.SetLimit(int64(limit))

	profileMap := make(map[int]*peoplerpc.ConciseProfileReply)
	if strings.ToUpper(reactionType) == "COMMENTS" {
		opts = options.FindOne().SetProjection(
			bson.M{
				"comments": 1,
				"_id":      0,
			})
		err = thingCollection.FindOne(context.TODO(), filter, opts).Decode(&comments)
		if err != nil {
			return nil, errors.Wrap(err, "unable to find reaction")
		}
		total = len(comments.Comment)

		var profileId int
		for i := range comments.Comment {
			profileId = comments.Comment[i].ProfileID

			ownerInfo, exist := profileMap[profileId]
			if !exist {
				// ownerInfo, err = profileService.FetchConciseProfile(profileId)

				cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(profileId)}
				ownerInfo, err := profileService.GetConciseProfile(context.TODO(), cpreq)
				if err != nil {
					return nil, errors.Wrap(err, "unable to find basic info")
				}

				profileMap[profileId] = ownerInfo
			}

			ownerInfo.Id = int32(profileId)
			profileInfo.ConciseProfile = ownerInfo
			profileInfo.Comment = &comments.Comment[i]
			// t := comments.Comment[i].CreateDate
			// profileInfo.Comment.AddedTime = t.Format("01-02-2006 15:04:05")
			// t = comments.Comment[i].LastModifiedDate
			// profileInfo.Comment.EditTime = t.Format("01-02-2006 15:04:05")
			response = append(response, profileInfo)
		}

	} else if strings.ToUpper(reactionType) == "LIKES" {
		opts = options.FindOne().SetProjection(
			bson.M{
				"likes": 1,
				"_id":   0,
			})
		err = thingCollection.FindOne(context.TODO(), filter, opts).Decode(&likes)
		if err != nil {
			return nil, errors.Wrap(err, "unable to find reaction")
		}
		total = len(likes.Likes)
		var profileId int
		for i := range likes.Likes {
			profileId, _ = strconv.Atoi(likes.Likes[i])
			// ownerInfo, err := profileService.FetchConciseProfile(profileId)

			cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(profileId)}
			ownerInfo, err := profileService.GetConciseProfile(context.TODO(), cpreq)
			if err != nil {
				return nil, errors.Wrap(err, "unable to find basic info")
			}

			ownerInfo.Id = cpreq.ProfileId
			profileInfo.ConciseProfile = ownerInfo
			profileInfo.Comment = nil
			response = append(response, profileInfo)
		}
	}

	subsetResponse := paginate(response, page, limit, total)
	if reactionType == "comments" {
		sort.Slice(subsetResponse, func(i, j int) bool {
			return subsetResponse[j].Comment.LastModifiedDate.Before(subsetResponse[i].Comment.LastModifiedDate)
		})
	}

	return util.SetPaginationResponse(subsetResponse, total, 1, "Reaction fetched successfully."), nil
}

func isBookmarkedByProfile(db *mongodatabase.DBConfig, thingID string, profileID int) (bool, string, error) {
	dbconn, err := db.New(consts.Bookmark)
	if err != nil {
		return false, "", err
	}
	bmColl, client := dbconn.Collection, dbconn.Client
	defer client.Disconnect(context.TODO())

	var bm model.Bookmark
	err = bmColl.FindOne(context.TODO(), bson.M{"thingID": thingID, "profileID": profileID}).Decode(&bm)
	if err != nil {
		return false, "", nil
	}

	return true, bm.ID.Hex(), nil
}
func getPostDetailsByThingID(db *mongodatabase.DBConfig, thingID, thingType string) (*model.Post, error) {
	var post model.Post

	switch strings.ToUpper(thingType) {

	case "FILE":
		var file model.UploadedFile

		dbconn, err := db.New(consts.File)
		if err != nil {
			return nil, errors.Wrap(err, "unable to establish connection with file collection.")
		}

		fileCollection, fileClient := dbconn.Collection, dbconn.Client
		defer fileClient.Disconnect(context.TODO())

		thingObjID, err := primitive.ObjectIDFromHex(thingID)
		if err != nil {
			return nil, errors.Wrap(err, "Error while converting object id")
		}

		filter := bson.M{"_id": thingObjID}

		err = fileCollection.FindOne(context.TODO(), filter).Decode(&file)
		if err != nil {
			return nil, errors.Wrap(err, "File not found")
		}

		dbconn2, err := db.New(consts.Post)
		if err != nil {
			return nil, errors.Wrap(err, "unable to establish connection with post collection.")
		}

		postCollection, postClient := dbconn2.Collection, dbconn2.Client
		defer postClient.Disconnect(context.TODO())

		filter = bson.M{"_id": file.PostID}

		err = postCollection.FindOne(context.TODO(), filter).Decode(&post)
		if err != nil {
			return nil, errors.Wrap(err, "Post not found")
		}

		return &post, nil

	case "NOTE":
		var note map[string]interface{}

		dbconn, err := db.New(consts.Note)
		if err != nil {
			return nil, errors.Wrap(err, "unable to establish connection with note collection.")
		}

		Collection, Client := dbconn.Collection, dbconn.Client
		defer Client.Disconnect(context.TODO())

		thingObjID, err := primitive.ObjectIDFromHex(thingID)
		if err != nil {
			return nil, errors.Wrap(err, "Error while converting object id")
		}

		filter := bson.M{"_id": thingObjID}

		err = Collection.FindOne(context.TODO(), filter).Decode(&note)
		if err != nil {
			return nil, errors.Wrap(err, "Note not found")
		}

		dbconn2, err := db.New(consts.Post)
		if err != nil {
			return nil, errors.Wrap(err, "unable to establish connection with post collection.")
		}

		postCollection, postClient := dbconn2.Collection, dbconn2.Client
		defer postClient.Disconnect(context.TODO())

		filter = bson.M{"_id": note["postID"]}
		err = postCollection.FindOne(context.TODO(), filter).Decode(&post)
		if err != nil {
			return nil, errors.Wrap(err, "Post not found")
		}

		return &post, nil

	case "TASK":

		var task map[string]interface{}

		dbconn, err := db.New(consts.Task)
		if err != nil {
			return nil, errors.Wrap(err, "unable to establish connection with task collection.")
		}

		Collection, Client := dbconn.Collection, dbconn.Client
		defer Client.Disconnect(context.TODO())

		thingObjID, err := primitive.ObjectIDFromHex(thingID)
		if err != nil {
			return nil, errors.Wrap(err, "Error while converting object id")
		}

		filter := bson.M{"_id": thingObjID}
		err = Collection.FindOne(context.TODO(), filter).Decode(&task)
		if err != nil {
			return nil, errors.Wrap(err, "Task not found")
		}

		dbconn2, err := db.New(consts.Post)
		if err != nil {
			return nil, errors.Wrap(err, "unable to establish connection with post collection.")
		}

		postCollection, postClient := dbconn2.Collection, dbconn2.Client
		defer postClient.Disconnect(context.TODO())

		filter = bson.M{"_id": task["postID"]}
		err = postCollection.FindOne(context.TODO(), filter).Decode(&post)
		if err != nil {
			return nil, errors.Wrap(err, "Post not found")
		}

		return &post, nil

	case "COLLECTION":

		var col model.Collection

		dbconn, err := db.New(consts.Collection)
		if err != nil {
			return nil, errors.Wrap(err, "unable to establish connection with task collection.")
		}

		Collection, Client := dbconn.Collection, dbconn.Client
		defer Client.Disconnect(context.TODO())

		thingObjID, err := primitive.ObjectIDFromHex(thingID)
		if err != nil {
			return nil, errors.Wrap(err, "Error while converting object id")
		}

		filter := bson.M{"_id": thingObjID}
		err = Collection.FindOne(context.TODO(), filter).Decode(&col)
		if err != nil {
			return nil, errors.Wrap(err, "Task not found")
		}

		dbconn2, err := db.New(consts.Post)
		if err != nil {
			return nil, errors.Wrap(err, "unable to establish connection with post collection.")
		}

		postCollection, postClient := dbconn2.Collection, dbconn2.Client
		defer postClient.Disconnect(context.TODO())

		filter = bson.M{"_id": col.PostID}
		err = postCollection.FindOne(context.TODO(), filter).Decode(&post)
		if err != nil {
			return nil, errors.Wrap(err, "Post not found")
		}

		return &post, nil

	case "POST":

		dbconn, err := db.New(consts.Post)
		if err != nil {
			return nil, errors.Wrap(err, "unable to establish connection with post collection.")
		}

		Collection, Client := dbconn.Collection, dbconn.Client
		defer Client.Disconnect(context.TODO())

		thingObjID, err := primitive.ObjectIDFromHex(thingID)
		if err != nil {
			return nil, errors.Wrap(err, "Error while converting object id")
		}

		filter := bson.M{"_id": thingObjID}

		err = Collection.FindOne(context.TODO(), filter).Decode(&post)
		if err != nil {
			return nil, errors.Wrap(err, "Post not found")
		}

		return &post, nil

	default:
		return nil, errors.New("thing type not match.")
	}
}

func addPostComment(db *mongodatabase.DBConfig, post *model.Post, profileID int, comment string) (map[string]interface{}, error) {
	dbconn, err := db.New(consts.Post)
	if err != nil {
		return nil, errors.Wrap(err, "unable to establish connection with post collection.")
	}

	postCollection, Client := dbconn.Collection, dbconn.Client
	defer Client.Disconnect(context.TODO())

	filter := bson.M{"_id": post.Id}

	var newComment model.Comment
	newComment.ID = primitive.NewObjectID()
	newComment.ProfileID = profileID
	newComment.Message = comment
	newComment.CreateDate = time.Now()
	newComment.LastModifiedDate = time.Now()
	post.Comments = append(post.Comments, newComment)
	update := bson.M{"$set": bson.M{"comments": post.Comments}}
	_, err = postCollection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		return nil, errors.Wrap(err, "unable to update post at mongo")
	}

	t := newComment.CreateDate
	newComment.AddedTime = t.Format("01-02-2006 15:04:05")
	t = newComment.LastModifiedDate
	newComment.EditTime = t.Format("01-02-2006 15:04:05")
	return util.SetResponse(newComment, 1, "Comment added successfully."), nil
}

func getThingBasedOffIDAndType(db *mongodatabase.DBConfig, thingID, thingType string) (map[string]interface{}, error) {
	var colType string

	switch strings.ToUpper(thingType) {
	case consts.BoardType:
		colType = consts.Board
	case consts.PostType:
		colType = consts.Post
	case consts.NoteType:
		colType = consts.Note
	case consts.FileType:
		colType = consts.File
	case consts.TaskType:
		colType = consts.Task
	case consts.CollectionType:
		colType = consts.Collection
	default:
		return nil, errors.New("Unable to match thing type.")
	}

	dbconn, err := db.New(colType)
	if err != nil {
		return nil, err
	}
	thingCollection, thingClient := dbconn.Collection, dbconn.Client
	defer thingClient.Disconnect(context.TODO())

	objID, err := primitive.ObjectIDFromHex(thingID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to ObjectID")
	}
	var thing map[string]interface{}
	err = thingCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&thing)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find the thing")
	}

	return thing, nil
}
