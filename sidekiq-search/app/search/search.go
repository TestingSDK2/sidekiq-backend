package search

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	model "github.com/TestingSDK2/sidekiq-backend/sidekiq-models"
	"github.com/pkg/errors"

	contentrpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-content/v1"
	peoplerpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-search/consts"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-search/database"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-search/mongodatabase"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-search/util"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func paginate(arr []map[string]interface{}, pageNo, limit int) (ret []map[string]interface{}) {
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

// PASS THE GRPC CLIENT OF PROFILE SERVICE

func globalFTS(db *mongodatabase.DBConfig, mysql *database.Database, boardService contentrpc.BoardServiceClient, profileService peoplerpc.AccountServiceClient,
	filter *model.GlobalSearchFilter, profileID int, query, page, limit, sortBy, orderBy string) (map[string]interface{}, error) {
	// searchResult
	dbconn, err := db.New(consts.SearchResult)
	if err != nil {
		return nil, err
	}
	sr, srClient := dbconn.Collection, dbconn.Client
	defer srClient.Disconnect(context.TODO())

	var wg sync.WaitGroup
	var wg2 sync.WaitGroup
	errChan := make(chan error)
	var orderByInt int
	if orderBy == "desc" || strings.ToUpper(orderBy) == "DESC" || orderBy == "" {
		orderByInt = -1
	} else if orderBy == "asc" || strings.ToUpper(orderBy) == "ASC" {
		orderByInt = 1
	}

	fmt.Println("filter before processing: ", filter)

	if filter != nil {
		err = filter.Process(db, mysql, &wg, profileID)
		if err != nil {
			return nil, err
		}
	}

	pgInt, _ := strconv.Atoi(page)
	limitInt, _ := strconv.Atoi(limit)
	offset := (pgInt - 1) * limitInt

	opts := options.Aggregate().SetMaxTime(105 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	var results []map[string]interface{}
	ret := make(chan []map[string]interface{})

	fmt.Printf("%s\n%s\n%v\n", strings.Repeat("*", 50), "filter after processing: ", filter)

	wg2.Add(1)
	go GlobalSearchFTS(query, strconv.Itoa(profileID), filter, sr, ctx, opts, ret, &wg2, offset, limitInt, sortBy, orderByInt)
	if filter.Connections {
		dbconn2, err := db.New(consts.Connection)
		if err != nil {
			return nil, err
		}
		conn, connClient := dbconn2.Collection, dbconn2.Client
		defer connClient.Disconnect(context.TODO())

		wg2.Add(1)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			connRet := SearchConnection(query, filter.Profiles, conn, ctx, opts)
			results = append(results, connRet...)
		}(&wg2)
	}

	results = <-ret
	wg2.Wait()

	cps := make(map[int]*peoplerpc.ConciseProfileReply)
	mutex := sync.Mutex{}
	ch := make(chan bool, 4)

	// sorting if the connections are searched
	if filter.Connections && sortBy == "createDate" {
		sort.Slice(results, func(i, j int) bool {
			cd1 := util.ParseDate(results[i]["createDate"])
			cd2 := util.ParseDate(results[j]["createDate"])
			if strings.ToLower(orderBy) == "asc" {
				return cd1.Before(cd2)
			}
			return cd1.After(cd2)
		})
	}

	subset := paginate(results, pgInt, limitInt)

	// ranked, paginated
	for _, result := range subset {
		ch <- true
		go func(result map[string]interface{}, errChan chan<- error) {
			defer util.RecoverGoroutinePanic(errChan)
			ownerID, _ := result["owner"].(string)
			ownerIDInt, _ := strconv.Atoi(ownerID)

			// fetch basic info
			var cp *peoplerpc.ConciseProfileReply
			if val, ok := cps[ownerIDInt]; !ok {
				cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(ownerIDInt)}
				cp, err := profileService.GetConciseProfile(context.TODO(), cpreq)
				if err != nil {
					errChan <- err
				}

				cps[ownerIDInt] = cp
			} else {
				cp = val
			}
			cp.Id = int32(ownerIDInt)
			mutex.Lock()
			result["ownerInfo"] = *cp
			mutex.Unlock()

			// reactions
			// if result["likes"] != nil {
			// 	var likes []string
			// 	for _, value := range result["likes"].(primitive.A) {
			// 		likes = append(likes, value.(string))
			// 	}
			// 	if util.Contains(likes, strconv.Itoa(profileID)) {
			// 		result["isLiked"] = true
			// 	} else {
			// 		result["isLiked"] = false
			// 	}
			// }

			// if result["comments"] != nil {
			// 	result["totalComments"] = len(result["comments"].(primitive.A))
			// }
			// if result["likes"] != nil {
			// 	result["totalLikes"] = len(result["likes"].(primitive.A))
			// }

			// get thing location
			// if boardId, ok := result["boardID"].(primitive.ObjectID); ok {
			// 	loc, err := boardService.GetThingLocationOnBoard(boardId.Hex())
			// 	if err != nil {
			// 		if err.Error() == "unable to find board: mongo: no documents in result" {
			// 			errChan <- nil
			// 		} else {
			// 			errChan <- errors.Wrap(err, "unable to fetch board location")
			// 		}
			// 	}
			// 	result["location"] = loc
			// } else {
			// 	errChan <- errors.Wrap(err, "boardID not found")
			// }
			<-ch
			errChan <- nil
		}(result, errChan)
	}

	// waiting for goroutines to finish
	for i := 0; i < len(subset); i++ {
		if err := <-errChan; err != nil {
			return nil, errors.Wrap(err, "error from go routine from ftsOnDashboard()")
		}
	}

	if len(results) == 0 {
		return util.SetPaginationResponse(subset, 0, 1, "No search results found"), nil
	}
	return util.SetPaginationResponse(subset, len(results), 1, "Search results fetched successfully"), nil
}

// PASS THE GRPC CLIENT OF PROFILE SERVICE
func autoComplete(db *mongodatabase.DBConfig, profileService peoplerpc.AccountServiceClient, profileID int, query string) (map[string]interface{}, error) {
	dbconn, err := db.New(consts.SearchResult)
	if err != nil {
		return nil, err
	}
	sr, srClient := dbconn.Collection, dbconn.Client
	defer srClient.Disconnect(context.TODO())

	opts := options.Aggregate().SetMaxTime(105 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	var wg sync.WaitGroup

	var results []map[string]interface{}
	acResults := make(chan []map[string]interface{})

	wg.Add(1)
	go func() {
		defer util.RecoverGoroutinePanic(nil)
		GetAutoCompleteResults(query, sr, ctx, opts, acResults, &wg)
	}()

	results = append(results, <-acResults...)
	wg.Wait()
	errChan := make(chan error)
	cps := make(map[int]*peoplerpc.ConciseProfileReply)
	mutex := sync.Mutex{}
	ch := make(chan bool, 3)

	// limit and score done
	for i, val := range results {
		ch <- true
		go func(result map[string]interface{}, errChan chan<- error, i int) {
			// fetch basic info
			var cp *peoplerpc.ConciseProfileReply
			ownerID, _ := result["owner"].(string)
			ownerIDInt, _ := strconv.Atoi(ownerID)

			if val, ok := cps[ownerIDInt]; !ok {
				cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(ownerIDInt)}
				cp, err := profileService.GetConciseProfile(context.TODO(), cpreq)
				if err != nil {
					errChan <- err
				}

				cps[ownerIDInt] = cp
			} else {
				cp = val
			}
			cp.Id = int32(ownerIDInt)
			mutex.Lock()
			result["ownerInfo"] = *cp
			mutex.Unlock()

			<-ch
			errChan <- nil
		}(val, errChan, i)
	}

	// waiting for goroutines to finish
	for i := 0; i < len(results); i++ {
		if err := <-errChan; err != nil {
			return nil, errors.Wrap(err, "error from go routine from autoComplete()")
		}
	}

	return util.SetResponse(results, 1, "Autocomplete results found."), nil
}

func addToSearchHistory(db *mongodatabase.DBConfig, profileID int, query string) (map[string]interface{}, error) {
	dbconn, err := db.New(consts.SearchHistory)
	if err != nil {
		return nil, err
	}
	shColl, shClient := dbconn.Collection, dbconn.Client
	defer shClient.Disconnect(context.TODO())

	filter := bson.M{"profileID": strconv.Itoa(profileID)}

	// fetch search history
	var searchHistory model.SearchHistory
	err = shColl.FindOne(context.TODO(), filter).Decode(&searchHistory)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			searchHistory.ProfileID = strconv.Itoa(profileID)
			searchHistory.History = append(searchHistory.History, query)
			_, err = shColl.InsertOne(context.TODO(), searchHistory)
			if err != nil {
				return nil, errors.Wrap(err, "unable to insert search history")
			}
		} else {
			return nil, errors.Wrap(err, "unable to find search history")
		}
	}

	if util.Contains(searchHistory.History, query) {
		searchHistory.History = util.Remove(searchHistory.History, query)
	}
	searchHistory.History = append(searchHistory.History, query)

	_, err = shColl.UpdateOne(context.TODO(), filter, bson.M{"$set": searchHistory})
	if err != nil {
		return nil, errors.Wrap(err, "unable to update the search history")
	}

	return util.SetResponse(nil, 1, "Added"), nil
}

func fetchSearchHistory(db *mongodatabase.DBConfig, profileID int) (map[string]interface{}, error) {
	dbconn, err := db.New(consts.SearchHistory)
	if err != nil {
		return nil, err
	}
	shColl, shClient := dbconn.Collection, dbconn.Client
	defer shClient.Disconnect(context.TODO())

	filter := bson.M{"profileID": strconv.Itoa(profileID)}

	var results, finalRet model.SearchHistory
	err = shColl.FindOne(context.TODO(), filter).Decode(&results)
	if err != nil {
		errors.Wrap(err, "unable to find search results")
	}
	if err != nil {
		if err == mongo.ErrNoDocuments {
			results.ProfileID = strconv.Itoa(profileID)
			results.History = []string{}
			return util.SetResponse(results, 1, "You have no search history"), nil
		}
	}

	finalRet.ProfileID = results.ProfileID

	if len(results.History) > 5 {
		finalRet.History = results.History[len(results.History)-5:]
	}

	// reverse
	for i, j := 0, len(finalRet.History)-1; i < j; i, j = i+1, j-1 {
		finalRet.History[i], finalRet.History[j] = finalRet.History[j], finalRet.History[i]
	}

	return util.SetResponse(finalRet, 1, "Search history fetched"), nil
}

func updateSearchResults(db *mongodatabase.DBConfig, data map[string]interface{}, updateType string, args []string) error {
	dbconn, err := db.New(consts.SearchResult)
	if err != nil {
		return err
	}
	srColl, srClient := dbconn.Collection, dbconn.Client
	defer srClient.Disconnect(context.TODO())
	if updateType == "delete" {
		thingObjID, err := primitive.ObjectIDFromHex(args[0])
		if err != nil {
			return errors.Wrap(err, "unable to convert string to ObjectID for thingID")
		}
		_, err = srColl.DeleteOne(context.TODO(), bson.M{"_id": thingObjID})
		if err != nil {
			return errors.Wrap(err, "unable to delete record from SearchResult")
		}
		return nil
	}

	var searchResult model.SearchResult
	// Populate fields from the 'data' map
	if val, ok := data["_id"]; ok {
		// thingID, err := primitive.ObjectIDFromHex(val)
		// if err != nil {
		// 	return errors.Wrap(err, "unable to convert string to ObjectID for thingID")
		// }
		fmt.Println("val: ", val)
		if fmt.Sprintf("%T", val) == "string" {
			objID, err := primitive.ObjectIDFromHex(val.(string))
			if err != nil {
				return err
			}
			searchResult.ID = objID
		} else {
			searchResult.ID = val.(primitive.ObjectID)
		}
	}

	// saving the boardID
	if data["type"] == "BOARD" {
		searchResult.BoardID = searchResult.ID.Hex()
	} else {
		if val, ok := data["boardID"]; ok {
			// searchResult.BoardID = val.(primitive.ObjectID).Hex()
			searchResult.BoardID = val.(string)
		}
	}

	// if val, ok := data["boardID"]; ok {
	// 	searchResult.BoardID = val.(primitive.ObjectID).Hex()
	// if data["type"] == "BOARD" {
	// 	searchResult.BoardID = primitive.NilObjectID
	// } else {
	// boardID, err := primitive.ObjectIDFromHex(val)
	// if err != nil {
	// 	return errors.Wrap(err, "unable to convert string to ObjectID for boardID")
	// }
	// }
	// }

	if val, ok := data["type"]; ok {
		searchResult.Type = val.(string)
	}
	if val, ok := data["visible"]; ok {
		searchResult.Visible = val.(string)
	}
	if tags, ok := data["tags"].([]interface{}); ok {
		searchResult.Tags = make([]string, len(tags))
		for i, tag := range tags {
			if tagStr, isString := tag.(string); isString {
				searchResult.Tags[i] = tagStr
			} else {
				return errors.New("tag element is not a string")
			}
		}
	}
	if val, ok := data["description"]; ok {
		searchResult.Description = val.(string)
	}
	if val, ok := data["title"]; ok {
		searchResult.Title = val.(string)
	}
	if val, ok := data["body_raw"]; ok {
		searchResult.Body_raw = val.(string)
	}

	fmt.Print(strings.Repeat("*", 50))
	util.PrettyPrint("Search Result Object", searchResult)
	fmt.Print(strings.Repeat("*", 50))
	thingObjID := searchResult.ID
	switch updateType {
	case "insert":
		_, err = srColl.InsertOne(context.TODO(), searchResult)
		if err != nil {
			return errors.Wrap(err, "unable to insert into SearchResult")
		}
	case "update":
		_, err = srColl.UpdateOne(context.TODO(), bson.M{"_id": thingObjID}, bson.M{"$set": searchResult})
		if err != nil {
			return errors.Wrap(err, "unable to update record in SearchResult")
		}

	default:
		return errors.New("updateType is empty or incorrect")
	}
	return nil
}
