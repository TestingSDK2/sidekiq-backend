package recent

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	peoplerpc "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"

	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/board"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/member"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/post"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/storage"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/cache"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/consts"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/database"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/helper"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/mongodatabase"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/permissions"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/util"
	"github.com/ProImaging/sidekiq-backend/sidekiq-models/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func addToDashBoardRecent(db *mongodatabase.DBConfig, mysql *database.Database, thing model.Recent) error {
	thing.LastViewedDate = time.Now()
	thing.ExpectedExpiredDate = thing.LastViewedDate.Add(time.Hour * 168)

	dbconn, err := db.New(consts.Recent)
	if err != nil {
		return err
	}

	coll, client := dbconn.Collection, dbconn.Client
	defer client.Disconnect(context.TODO())

	// if it is already present then, upsert
	filter := bson.M{"thingID": thing.ThingID}
	update := bson.M{"$set": thing}
	opts := options.Update().SetUpsert(true)

	_, err = coll.UpdateOne(context.TODO(), filter, update, opts)
	if err != nil {
		return err
	}

	return nil
}

func fetchDashBoardRecentThings(db *mongodatabase.DBConfig, cache *cache.Cache, profileService peoplerpc.AccountServiceClient,
	storageService storage.Service, boardService board.Service, postService post.Service, search string, profileID int, sortBy, orderBy string, limitInt, pgInt int, isPagination bool) (map[string]interface{}, error) {

	var wg sync.WaitGroup
	var dbconn, dbconn1, dbconn2, dbconn3, dbconn4, dbconn5, dbconn6 *mongodatabase.MongoDBConn
	dbchainErr := make(chan error, 7)
	dbconnChain := make(chan *mongodatabase.MongoDBConn)
	dbconn1Chain := make(chan *mongodatabase.MongoDBConn)
	dbconn2Chain := make(chan *mongodatabase.MongoDBConn)
	dbconn3Chain := make(chan *mongodatabase.MongoDBConn)
	dbconn4Chain := make(chan *mongodatabase.MongoDBConn)
	dbconn5Chain := make(chan *mongodatabase.MongoDBConn)
	dbconn6Chain := make(chan *mongodatabase.MongoDBConn)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		defer util.RecoverGoroutinePanic(nil)
		dbconn, err := db.New(consts.Recent)
		if err != nil {
			dbchainErr <- errors.Wrap(err, "unable to connect to Recent")
			return
		}
		dbchainErr <- nil
		dbconnChain <- dbconn
	}(&wg)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		defer util.RecoverGoroutinePanic(nil)
		dbconn1, err := db.New(consts.Board)
		if err != nil {
			dbchainErr <- errors.Wrap(err, "unable to connect to Board")
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
		dbconn3, err := db.New(consts.Post)
		if err != nil {
			dbchainErr <- errors.Wrap(err, "unable to connect to Post")
			return
		}

		dbchainErr <- nil
		dbconn3Chain <- dbconn3
	}(&wg)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		defer util.RecoverGoroutinePanic(nil)
		dbconn4, err := db.New(consts.Task)
		if err != nil {
			dbchainErr <- errors.Wrap(err, "unable to connect to Task")
			return
		}

		dbchainErr <- nil
		dbconn4Chain <- dbconn4
	}(&wg)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		defer util.RecoverGoroutinePanic(nil)
		dbconn5, err := db.New(consts.Bookmark)
		if err != nil {
			dbchainErr <- errors.Wrap(err, "unable to connect to Bookmark")
			return
		}

		dbchainErr <- nil
		dbconn5Chain <- dbconn5
	}(&wg)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		defer util.RecoverGoroutinePanic(nil)
		dbconn6, err := db.New(consts.File)
		if err != nil {
			dbchainErr <- errors.Wrap(err, "unable to connect to File")
			return
		}

		dbchainErr <- nil
		dbconn6Chain <- dbconn6
	}(&wg)

	for i := 1; i <= 7; i++ {
		errdb := <-dbchainErr
		if errdb != nil {
			return nil, errors.Wrap(errdb, "unable to connect with DB")
		}
	}

	dbconn = <-dbconnChain
	dbconn1 = <-dbconn1Chain
	dbconn2 = <-dbconn2Chain
	dbconn3 = <-dbconn3Chain
	dbconn4 = <-dbconn4Chain
	dbconn5 = <-dbconn5Chain
	dbconn6 = <-dbconn6Chain

	wg.Wait()

	coll, Recentclient := dbconn.Collection, dbconn.Client
	defer Recentclient.Disconnect(context.TODO())

	boardColl, boardClient := dbconn1.Collection, dbconn1.Client
	defer boardClient.Disconnect(context.TODO())

	noteColl, noteClient := dbconn2.Collection, dbconn2.Client
	defer noteClient.Disconnect(context.TODO())

	postColl, postClient := dbconn3.Collection, dbconn3.Client
	defer postClient.Disconnect(context.TODO())

	taskColl, taskClient := dbconn4.Collection, dbconn4.Client
	defer taskClient.Disconnect(context.TODO())

	bmColl, bookmarkClient := dbconn5.Collection, dbconn5.Client
	defer bookmarkClient.Disconnect(context.TODO())

	fileCollection, fileClient := dbconn6.Collection, dbconn6.Client
	defer fileClient.Disconnect(context.TODO())

	profileIDStr := strconv.Itoa(profileID)
	findOptions := options.Find()
	var filterorderBy int64

	bpms := make(map[string]model.BoardPermission)

	if sortBy == "" {
		sortBy = "lastViewedDate"
	} else if strings.ToLower(sortBy) == "title" || strings.ToLower(sortBy) == "displaytitle" {
		sortBy = "displayTitle"
	}

	if orderBy == "" || strings.ToLower(orderBy) == "desc" {
		filterorderBy = -1
	} else {
		filterorderBy = 1
	}

	findFilter := bson.M{"profileID": profileIDStr}
	findOptions.SetSort(bson.M{sortBy: filterorderBy})
	sortingOption := findOptions.SetSort(bson.M{sortBy: filterorderBy})
	collation := &options.Collation{
		Locale:   "en", // Set your desired locale.
		Strength: 2,    // Strength 2 for case-insensitive.
	}
	findOptions = sortingOption.SetCollation(collation)

	if search != "" {
		findFilter["displayTitle"] = bson.M{"$regex": primitive.Regex{Pattern: search, Options: "i"}}
	}

	if isPagination && pgInt > 0 {
		offset := (pgInt - 1) * limitInt
		findOptions.SetSkip(int64(offset))
		findOptions.SetLimit(int64(limitInt))
	} else {
		findOptions.SetLimit(5)
	}

	total, err := coll.CountDocuments(context.TODO(), findFilter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find recently viewed things count")
	}

	if isPagination {
		if int(total) == 0 {
			return util.SetPaginationResponse([]model.Recent{}, 0, 1, "Recently viewed things would be shown here."), nil
		}
	} else {
		if int(total) == 0 {
			return util.SetResponse([]model.Recent{}, 0, "Recently viewed things would be shown here."), nil
		}
	}

	var recentThingsResults []*model.Recent
	cursor, err := coll.Find(context.TODO(), findFilter, findOptions)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find recently viewed things")
	}
	defer cursor.Close(context.TODO())

	err = cursor.All(context.TODO(), &recentThingsResults)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find recently viewed things")
	}

	resultfix := make(map[int]*map[string]interface{})

	var filteredResults []*map[string]interface{}
	errChan := make(chan error, len(recentThingsResults))
	for index, item := range recentThingsResults {
		switch strings.ToUpper(item.ThingType) {
		case "TASK":
			wg.Add(1)
			go func(wg *sync.WaitGroup, itemNew *model.Recent, errChan chan<- error, index int) {
				defer wg.Done()
				defer util.RecoverGoroutinePanic(nil)
				var task map[string]interface{}

				err := taskColl.FindOne(context.TODO(), bson.M{"_id": itemNew.ThingID}).Decode(&task)
				if err != nil {

					if errors.Is(err, mongo.ErrNoDocuments) {
						coll.DeleteMany(context.TODO(), bson.M{"thingID": itemNew.ThingID})
						errChan <- nil
						return
					}

					logrus.Error(err, " Error from finding task")
					errChan <- err
					return
				}

				isbookmarked, bid, err := checkProfileBookmark(bmColl, itemNew.ThingID.Hex(), profileID)
				if err != nil {
					task["isBookmarked"] = false
					task["bookmarkID"] = ""
				} else {
					task["isBookmarked"] = isbookmarked
					task["bookmarkID"] = bid
				}

				if postObjID, ok := task["postID"].(primitive.ObjectID); ok {
					post, err := getPostDetailsByID(postColl, postObjID, profileService, storageService)
					if err != nil {
						if errors.Is(err, mongo.ErrNoDocuments) {
							coll.DeleteMany(context.TODO(), bson.M{"thingID": postObjID})
							coll.DeleteMany(context.TODO(), bson.M{"thingID": itemNew.ThingID})
							errChan <- nil
							return
						}

						logrus.Error(err, " Error from finding post")
						errChan <- err
						return
					}

					task["ownerInfo"] = post.OwnerInfo
					task["boardID"] = post.BoardID

				} else if postIDstr, ok := task["postID"].(string); ok {

					postObjID, err := primitive.ObjectIDFromHex(postIDstr)
					if err != nil {
						logrus.Error(err, " Error from coverting ID")
						errChan <- err
						return
					}

					post, err := getPostDetailsByID(postColl, postObjID, profileService, storageService)
					if err != nil {

						if errors.Is(err, mongo.ErrNoDocuments) {
							coll.DeleteMany(context.TODO(), bson.M{"thingID": postObjID})
							coll.DeleteMany(context.TODO(), bson.M{"thingID": itemNew.ThingID})
							errChan <- nil
							return
						}

						logrus.Error(err, " Error from finding post")
						errChan <- err
						return
					}

					task["ownerInfo"] = post.OwnerInfo
					task["boardID"] = post.BoardID
				}

				assignedMemberInfo, err := member.GetAssignedMemberInfo(task, profileService)
				if err != nil {
					logrus.Error(err, " Error from GetAssignedMemberInfo")
					errChan <- err
					return
				}

				task["assignedMemberInfo"] = assignedMemberInfo

				reporterInfo, err := member.GetReporterInfo(task, profileService)
				if err != nil {
					logrus.Error(err, " Error from reporterInfo")
					errChan <- err
					return
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
				task["type"] = "TASK"
				if task["comments"] != nil {
					task["totalComments"] = len(task["comments"].(primitive.A))
				} else {
					task["totalComments"] = 0
				}

				filteredResults = append(filteredResults, &task)
				resultfix[index] = &task
				errChan <- nil

			}(&wg, item, errChan, index)

		case "FILE":
			wg.Add(1)
			defer util.RecoverGoroutinePanic(nil)
			go func(wg *sync.WaitGroup, itemNew *model.Recent, errChan chan<- error, index int) {
				defer wg.Done()
				defer util.RecoverGoroutinePanic(nil)

				filemap, err := getFileByID(boardColl, fileCollection, boardService, profileService, storageService, itemNew.ThingID.Hex(), profileID)
				if err != nil {

					if errors.Is(err, mongo.ErrNoDocuments) {
						coll.DeleteMany(context.TODO(), bson.M{"thingID": itemNew.ThingID})
						errChan <- nil
						return
					}

					errChan <- err
					return
				}

				filemap["type"] = "FILE"

				isbookmarked, bid, err := checkProfileBookmark(bmColl, itemNew.ThingID.Hex(), profileID)
				if err != nil {
					filemap["isBookmarked"] = false
					filemap["bookmarkID"] = ""
				} else {
					filemap["isBookmarked"] = isbookmarked
					filemap["bookmarkID"] = bid
				}

				if postObjID, ok := filemap["postID"].(primitive.ObjectID); ok {
					post, err := getPostDetailsByID(postColl, postObjID, profileService, storageService)
					if err != nil {

						if errors.Is(err, mongo.ErrNoDocuments) {
							coll.DeleteMany(context.TODO(), bson.M{"thingID": postObjID})
							coll.DeleteMany(context.TODO(), bson.M{"thingID": itemNew.ThingID})
							errChan <- nil
							return
						}

						logrus.Error(err, " Error from fetching post.")
						errChan <- err
						return
					}

					filemap["ownerInfo"] = post.OwnerInfo
					filemap["boardID"] = post.BoardID

				} else if postIDstr, ok := filemap["postID"].(string); ok {

					postObjID, err := primitive.ObjectIDFromHex(postIDstr)
					if err != nil {
						logrus.Error(err, " Error from coverting id.")
						errChan <- err
						return
					}

					post, err := getPostDetailsByID(postColl, postObjID, profileService, storageService)
					if err != nil {

						if errors.Is(err, mongo.ErrNoDocuments) {
							coll.DeleteMany(context.TODO(), bson.M{"thingID": postObjID})
							coll.DeleteMany(context.TODO(), bson.M{"thingID": itemNew.ThingID})
							errChan <- nil
							return
						}

						logrus.Error(err, " Error from fetching post.")
						errChan <- err
						return
					}

					filemap["ownerInfo"] = post.OwnerInfo
					filemap["boardID"] = post.BoardID

				}

				filteredResults = append(filteredResults, &filemap)
				resultfix[index] = &filemap
				errChan <- nil
			}(&wg, item, errChan, index)
		case "BOARD":
			wg.Add(1)
			go func(wg *sync.WaitGroup, itemNew *model.Recent, errChan chan<- error, index int) {
				defer wg.Done()
				defer util.RecoverGoroutinePanic(nil)

				var board model.Board

				err := boardColl.FindOne(context.TODO(), bson.M{"_id": itemNew.ThingID, "isDefaultBoard": false}).Decode(&board)
				if err != nil {

					if errors.Is(err, mongo.ErrNoDocuments) {
						coll.DeleteMany(context.TODO(), bson.M{"thingID": itemNew.ThingID})
						coll.DeleteMany(context.TODO(), bson.M{"thingID": itemNew.ThingID})
						errChan <- nil
						return
					}

					logrus.Error(err, " Error from fetching board.")
					errChan <- err
					return
				}

				board.IsBoardFollower = util.Contains(board.Followers, strconv.Itoa(profileID))
				// role
				var bpm model.BoardPermission
				profileKey := fmt.Sprintf("boards:%s", profileIDStr)
				if v, ok := bpms[board.Id.Hex()+profileIDStr]; !ok {
					bpm = permissions.GetBoardPermissionsNew(profileKey, cache, &board, profileIDStr)
					bpms[board.Id.Hex()+profileIDStr] = bpm
				} else {
					bpm = v
				}
				board.Type = "BOARD"
				board.Role = bpm[board.Id.Hex()]
				if board.Owner != "" {
					// get basic info
					ownerIdInt, _ := strconv.Atoi(board.Owner)
					cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(ownerIdInt)}
					cp, err := profileService.GetConciseProfile(context.TODO(), cpreq)
					if err != nil {
						errChan <- err
						return
					}

					board.OwnerInfo = cp
				}

				if util.Contains(board.Likes, fmt.Sprint(profileID)) {
					board.IsLiked = true
				} else {
					board.IsLiked = false
				}

				board.TotalLikes = len(board.Likes)
				board.TotalComments = len(board.Comments)

				isbookmarked, bid, err := checkProfileBookmark(bmColl, itemNew.ThingID.Hex(), profileID)
				if err != nil {
					board.IsBookmarked = false
					board.BookmarkID = ""
				} else {
					board.IsBookmarked = isbookmarked
					board.BookmarkID = bid
				}

				var members []string
				members = append(members, board.Admins...)
				members = append(members, board.Guests...)
				members = append(members, board.Subscribers...)
				members = append(members, board.Viewers...)

				if util.Contains(members, fmt.Sprint(profileID)) {
					board.IsBoardShared = true
				} else {
					board.IsBoardShared = false
				}

				boardmap := board.ToMap()

				filteredResults = append(filteredResults, &boardmap)
				resultfix[index] = &boardmap
				errChan <- nil

			}(&wg, item, errChan, index)

		case "NOTE":
			wg.Add(1)
			go func(wg *sync.WaitGroup, itemNew *model.Recent, errChan chan<- error, index int) {
				defer wg.Done()
				defer util.RecoverGoroutinePanic(nil)

				var note map[string]interface{}

				err := noteColl.FindOne(context.TODO(), bson.M{"_id": itemNew.ThingID}).Decode(&note)
				if err != nil {

					if errors.Is(err, mongo.ErrNoDocuments) {
						coll.DeleteMany(context.TODO(), bson.M{"thingID": itemNew.ThingID})
						errChan <- nil
						return
					}

					logrus.Error(err, " Error from fetching note.")
					errChan <- err
					return
				}

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
				note["type"] = "NOTE"
				if note["comments"] != nil {
					note["totalComments"] = len(note["comments"].(primitive.A))
				} else {
					note["totalComments"] = 0
				}

				isbookmarked, bid, err := checkProfileBookmark(bmColl, itemNew.ThingID.Hex(), profileID)
				if err != nil {
					note["isBookmarked"] = false
					note["bookmarkID"] = ""
				} else {
					note["isBookmarked"] = isbookmarked
					note["bookmarkID"] = bid
				}

				post, err := getPostDetailsByID(postColl, note["postID"].(primitive.ObjectID), profileService, storageService)
				if err != nil {
					if errors.Is(err, mongo.ErrNoDocuments) {
						coll.DeleteMany(context.TODO(), bson.M{"thingID": note["postID"].(primitive.ObjectID)})
						coll.DeleteMany(context.TODO(), bson.M{"thingID": itemNew.ThingID})
						errChan <- nil
						return
					}

					logrus.Error(err, " Error from fetching post.")
					errChan <- err
					return
				}

				note["ownerInfo"] = post.OwnerInfo
				note["boardID"] = post.BoardID

				filteredResults = append(filteredResults, &note)
				resultfix[index] = &note
				errChan <- nil
			}(&wg, item, errChan, index)
		case "POST":
			wg.Add(1)
			go func(wg *sync.WaitGroup, itemNew *model.Recent, errChan chan<- error, index int) {
				defer wg.Done()
				defer util.RecoverGoroutinePanic(nil)

				var post model.Post

				err := postColl.FindOne(context.TODO(), bson.M{"_id": itemNew.ThingID}).Decode(&post)
				if err != nil {

					if errors.Is(err, mongo.ErrNoDocuments) {
						coll.DeleteMany(context.TODO(), bson.M{"thingID": itemNew.ThingID})
						errChan <- nil
						return
					}

					logrus.Error(err, " Error from fetching post.")
					errChan <- err
					return
				}
				post.Type = "POST"
				if post.Owner != "" {
					// get basic info
					ownerIdInt, _ := strconv.Atoi(post.Owner)
					// cp, err := profileService.FetchConciseProfile(ownerIdInt)

					cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(ownerIdInt)}
					cp, err := profileService.GetConciseProfile(context.TODO(), cpreq)
					if err != nil {
						errChan <- err
						return
					}

					post.OwnerInfo = cp
				}

				ret, _ := postService.GetFirstPostThing(post.Id.Hex(), post.BoardID.Hex(), profileID)

				isbookmarked, bid, err := checkProfileBookmark(bmColl, itemNew.ThingID.Hex(), profileID)
				if err != nil {
					post.IsBookmarked = false
					post.BookmarkID = ""
				} else {
					post.IsBookmarked = isbookmarked
					post.BookmarkID = bid
				}

				postMap := post.ToMap()
				postMap["things"] = ret

				filteredResults = append(filteredResults, &postMap)
				resultfix[index] = &postMap
				errChan <- nil
			}(&wg, item, errChan, index)
		}
	}

	fmt.Println("GO ROUTINE WAITING TO FINISH")
	wg.Wait()
	fmt.Println("GO ROUTINE COMPLETED")

	totalGo := len(recentThingsResults)
	for totalGo != 0 {
		cherr := <-errChan
		if cherr != nil {
			return nil, errors.Wrap(cherr, "error from go routine: ")
		}
		totalGo--
	}

	finalresult := make([]*map[string]interface{}, len(resultfix))
	for i := 0; i < len(resultfix); i++ {
		finalresult[i] = resultfix[i]
	}

	if strings.ToLower(sortBy) == "owner" {
		sort.Slice(finalresult, func(i, j int) bool {
			name1 := getOwnerName((*finalresult[i])["ownerInfo"])
			name2 := getOwnerName((*finalresult[j])["ownerInfo"])

			if strings.ToLower(orderBy) == "asc" {
				return strings.ToLower(name1) < strings.ToLower(name2)
			}
			return strings.ToLower(name1) > strings.ToLower(name2)
		})
	} else if strings.ToLower(sortBy) == "title" || strings.ToLower(sortBy) == "displaytitle" {
		sort.Slice(finalresult, func(i, j int) bool {
			if strings.ToLower(orderBy) == "asc" {
				return strings.ToLower((*finalresult[i])["title"].(string)) < strings.ToLower((*finalresult[j])["title"].(string))
			}
			return strings.ToLower((*finalresult[i])["title"].(string)) > strings.ToLower((*finalresult[j])["title"].(string))
		})
	} else if strings.ToLower(sortBy) == "createdate" {
		sort.Slice(finalresult, func(i, j int) bool {
			createdDatei := getCreatedDate((*finalresult[i])["createDate"])
			createdDatej := getCreatedDate((*finalresult[j])["createDate"])
			if strings.ToLower(orderBy) == "asc" {
				return createdDatei.Before(createdDatej)
			}
			return createdDatei.After(createdDatej)
		})
	}

	if !isPagination {
		return util.SetResponse(finalresult, 1, "Recent items fetched successfully"), nil
	}

	findFilter = bson.M{"profileID": profileIDStr}
	total, err = coll.CountDocuments(context.TODO(), findFilter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find recently viewed things count")
	}

	return util.SetPaginationResponse(finalresult, int(total), 1, "Recent items fetched successfully"), nil

}

func getPostDetailsByID(postColl *mongo.Collection, postID primitive.ObjectID, profileService peoplerpc.AccountServiceClient,
	storageService storage.Service) (*model.Post, error) {

	var post model.Post

	err := postColl.FindOne(context.TODO(), bson.M{"_id": postID}).Decode(&post)
	if err != nil {
		return nil, err
	}

	post.Type = "POST"
	if post.Owner != "" {
		// get basic info
		ownerIdInt, _ := strconv.Atoi(post.Owner)
		// cp, err := profileService.FetchConciseProfile(ownerIdInt)

		cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(ownerIdInt)}
		cp, err := profileService.GetConciseProfile(context.TODO(), cpreq)
		if err != nil {
			return nil, errors.Wrap(err, "unable to find basic info")
		}

		post.OwnerInfo = cp
	}

	return &post, nil
}

func getOwnerName(ownerInfo interface{}) string {
	if ownerobj, ok := ownerInfo.(*peoplerpc.ConciseProfileReply); ok {
		return fmt.Sprintf("%s %s", ownerobj.FirstName, ownerobj.LastName)
	} else if ownerobj, ok := ownerInfo.(*model.ConciseProfile); ok {
		return fmt.Sprintf("%s %s", ownerobj.FirstName, ownerobj.LastName)
	} else if ownerobj, ok := ownerInfo.(model.ConciseProfile); ok {
		return fmt.Sprintf("%s %s", ownerobj.FirstName, ownerobj.LastName)
	} else if ownerobj, ok := ownerInfo.(map[string]interface{}); ok {
		return fmt.Sprintf("%s %s", ownerobj["firstName"].(string), ownerobj["lastName"].(string))
	}
	logrus.Printf("No match found. Missing ownerinfo %T type", ownerInfo)
	return ""
}

func getCreatedDate(createDate interface{}) time.Time {
	if newDate, ok := createDate.(primitive.DateTime); ok {
		return newDate.Time()
	} else if newDate, ok := createDate.(string); ok {
		parsedTime, _ := time.Parse(time.RFC3339Nano, newDate)
		return parsedTime
	}

	logrus.Printf("No match found. Missing createDate %T type", createDate)
	return time.Time{}
}

func checkProfileBookmark(bmColl *mongo.Collection, thingID string, profileID int) (bool, string, error) {
	var bm model.Bookmark
	err := bmColl.FindOne(context.TODO(), bson.M{"thingID": thingID, "profileID": profileID}).Decode(&bm)
	if err != nil {
		return false, "", nil
	}
	return true, bm.ID.Hex(), nil
}

func getFileByID(boardCollection *mongo.Collection, fileCollection *mongo.Collection, boardService board.Service, profileService peoplerpc.AccountServiceClient, storageService storage.Service, mediaID string, profileID int) (map[string]interface{}, error) {
	var file model.UploadedFile
	var board model.Board

	mediaObjID, err := primitive.ObjectIDFromHex(mediaID)
	if err != nil {
		return nil, err
	}

	filter := bson.M{"_id": mediaObjID}

	err = fileCollection.FindOne(context.TODO(), filter).Decode(&file)
	if err != nil {
		return nil, err
	}

	boardCollection.FindOne(context.TODO(), bson.M{"_id": file.BoardID}).Decode(&board)

	fileOwner, _ := strconv.Atoi(file.Owner)
	// ownerInfo, err := profileService.FetchConciseProfile(fileOwner)

	cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(fileOwner)}
	ownerInfo, err := profileService.GetConciseProfile(context.TODO(), cpreq)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find basic info")
	}

	ownerInfo.Id = int32(fileOwner)
	file.OwnerInfo = ownerInfo

	boardOwner, _ := strconv.Atoi(board.Owner)
	// boardownerInfo, err := profileService.FetchConciseProfile(boardOwner)

	cpreq = &peoplerpc.ConciseProfileRequest{ProfileId: int32(boardOwner)}
	boardownerInfo, err := profileService.GetConciseProfile(context.TODO(), cpreq)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find basic info")
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
		return nil, err
	}
	file.URL = f.Filename

	// Fetch Thumbnails
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
	return file.ToMap(), nil
}

func deleteRecentItems(db *mongodatabase.DBConfig, mysql *database.Database, req model.RecentDeletePayload, profileID int) error {

	dbconn, err := db.New(consts.Recent)
	if err != nil {
		return err
	}

	coll, client := dbconn.Collection, dbconn.Client
	defer client.Disconnect(context.TODO())

	for _, id := range req.RecentIds {
		objId, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			continue
		}

		filter := bson.M{"thingID": objId, "profileID": fmt.Sprint(profileID)}
		_, err = coll.DeleteOne(context.TODO(), filter)
		if err != nil {
			continue
		}
	}

	return nil
}
