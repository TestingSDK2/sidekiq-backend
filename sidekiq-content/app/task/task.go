package task

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	// peoplerpc "github.com/sidekiq-people/proto/people"
	peoplerpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/board"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/member"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-models/model"
	"github.com/pkg/errors"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/storage"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/cache"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/consts"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/database"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/mongodatabase"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/permissions"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/util"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func addTasks(db *mongodatabase.DBConfig, profileService peoplerpc.AccountServiceClient, storageService storage.Service, postID primitive.ObjectID, tasks []interface{}) error {
	dbconn, err := db.New(consts.Task)
	if err != nil {
		return err
	}
	taskColl, taskClient := dbconn.Collection, dbconn.Client
	defer taskClient.Disconnect(context.TODO())

	errChan := make(chan error)
	goRoutines := 0

	for i := 0; i < len(tasks); i++ {
		goRoutines++
		go func(i int, errChan chan<- error) {
			// defer wg.Done()
			task := tasks[i].(map[string]interface{})
			task["postID"] = postID
			task["type"] = "TASK"
			task["comments"] = nil
			task["likes"] = nil
			task["state"] = consts.Active
			task["createDate"] = time.Now()
			task["editBy"] = ""
			task["editDate"] = nil
			// add assigneeID's info
			assignedMemberInfo, err := member.GetAssignedMemberInfo(task, profileService)
			if err != nil {
				errChan <- errors.Wrap(err, "unable to get assignedMemberInfo")
			}
			task["assignedMemberInfo"] = assignedMemberInfo

			reporterInfo, err := member.GetReporterInfo(task, profileService)
			if err != nil {
				errChan <- errors.Wrap(err, "unable to get reporterInfo")
			}
			task["reporterInfo"] = reporterInfo
		}(i, errChan)
	}

	// waiting for goroutines to finish
	fmt.Println(strings.Repeat("-", 100), "waiting for go routines to be finished")
	for goRoutines != 0 {
		if err := <-errChan; err != nil {
			return errors.Wrap(err, "error from go routine")
		}
		goRoutines--
	}
	fmt.Println(strings.Repeat("*", 100), "all go routines complete successfully")
	_, err = taskColl.InsertMany(context.TODO(), tasks)
	if err != nil {
		return errors.Wrap(err, "unable to insert tasks in post")
	}

	// what should be the response?
	return nil
}

func getTasksOfBoard(db *mongodatabase.DBConfig, mysql *database.Database, cache *cache.Cache,
	boardService board.Service, profileService peoplerpc.AccountServiceClient, storageService storage.Service,
	boardID string, profileID int, owner string, tagArr []string, uploadDate string, limit int, page, l string,
) (map[string]interface{}, error) {
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

	boardObjID, err := primitive.ObjectIDFromHex(boardID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to establish connection")
	}

	// find board filter
	var board *model.Board
	filter := bson.M{"_id": boardObjID}
	err = boardCollection.FindOne(context.TODO(), filter).Decode(&board)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find board")
	}

	dbconn2, err := db.New(consts.Task)
	if err != nil {
		return nil, err
	}
	taskCollection, taskClient := dbconn2.Collection, dbconn2.Client
	defer taskClient.Disconnect(context.TODO())

	findTasksFilter := bson.M{"$and": bson.A{
		bson.M{"boardID": boardObjID},
		bson.M{"state": bson.M{"$ne": consts.Hidden}},
	}}
	if owner != "" {
		findTasksFilter["owner"] = owner
	}
	if len(tagArr) != 0 {
		findTasksFilter["tags"] = bson.M{"$all": tagArr}
	}
	if uploadDate != "" {
		copyDate := uploadDate
		custom := "2006-01-02T15:04:05Z"

		start := copyDate + "T00:00:00Z"
		dayStart, _ := time.Parse(custom, start)

		uploadDate = uploadDate + "T11:59:59Z"
		dayEnd, _ := time.Parse(custom, uploadDate)

		findTasksFilter["createDate"] = bson.M{"$gte": dayStart, "$lte": dayEnd}
	}

	var curr *mongo.Cursor
	var opts *options.FindOptions
	var isPaginated bool = false

	findOptions := options.Find()
	findOptions.SetSort(bson.M{"createDate": -1})

	total, err := taskCollection.CountDocuments(context.TODO(), findTasksFilter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find total count")
	}

	if limit != 0 {
		opts = options.Find().SetLimit(int64(limit))
		curr, err = taskCollection.Find(context.TODO(), findTasksFilter, opts, findOptions)
	} else if limit == 0 && l != "" && page != "" {
		// pagination
		pgInt, _ := strconv.Atoi(page)
		limitInt, _ := strconv.Atoi(l)
		offset := (pgInt - 1) * limitInt
		findOptions.SetSkip(int64(offset))
		findOptions.SetLimit(int64(limitInt))
		curr, err = taskCollection.Find(context.TODO(), findTasksFilter, findOptions)
		isPaginated = true
	}
	defer curr.Close(context.TODO())
	if err != nil {
		return nil, errors.Wrap(err, "unable to find tasks")
	}

	var tasks []*model.Task
	err = curr.All(context.TODO(), &tasks)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch tasks")
	}

	if len(tasks) == 0 {
		return util.SetPaginationResponse([]*model.Task{}, 0, 1, "Board contains no tasks. Please add one."), nil
	}

	for idx := range tasks {
		// fetching owner info of the tasks
		// taskOwner, _ := strconv.Atoi(tasks[idx].Owner)
		// ownerInfo, err := profileService.FetchConciseProfile(taskOwner, storageService)
		// if err != nil {
		// 	return nil, errors.Wrap(err, "unable to find ownner's info")
		// }
		// ownerInfo.Id = taskOwner
		// tasks[idx].OwnerInfo = ownerInfo

		// fetch assigned member info
		if tasks[idx].AssignedToID != "" {
			idInt, _ := strconv.Atoi(tasks[idx].AssignedToID)
			// info, err := profileService.FetchConciseProfile(idInt)
			// if err == nil {
			// 	info.Id = idInt
			// 	tasks[idx].AssignedMemberInfo = info
			// }

			cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(idInt)}
			info, err := profileService.GetConciseProfile(context.TODO(), cpreq)
			if err != nil {
				return nil, err
			}

			info.Id = int32(idInt)
			tasks[idx].AssignedMemberInfo = info
		} else {
			tasks[idx].AssignedMemberInfo = nil
		}

		// reaction count
		if util.Contains(tasks[idx].Likes, profileIDStr) {
			tasks[idx].IsLiked = true
		} else {
			tasks[idx].IsLiked = false
		}
		tasks[idx].TotalComments = len(tasks[idx].Comments)
		tasks[idx].TotalLikes = len(tasks[idx].Likes)

		// get location
		// loc, err := boardService.GetThingLocationOnBoard(tasks[idx].BoardID.Hex())
		// if err != nil {
		// 	continue
		// }
		// tasks[idx].Location = loc
	}

	if isPaginated {
		return util.SetPaginationResponse(tasks, int(total), 1, "Tasks fetched successfully."), nil
	}
	return util.SetResponse(tasks, 1, "Tasks fetched successfully."), nil
}

func addTask(db *mongodatabase.DBConfig, profileService peoplerpc.AccountServiceClient, storageService storage.Service, postID primitive.ObjectID, task map[string]interface{}) (map[string]interface{}, error) {
	dbconn, err := db.New(consts.Task)
	if err != nil {
		return nil, err
	}
	taskColl, taskClient := dbconn.Collection, dbconn.Client
	defer taskClient.Disconnect(context.TODO())

	task["postID"] = postID
	task["type"] = "TASK"
	task["comments"] = nil
	task["likes"] = nil
	task["state"] = consts.Active
	task["createDate"] = time.Now()
	task["editBy"] = ""
	task["editDate"] = nil

	result, err := taskColl.InsertOne(context.TODO(), task)
	if err != nil {
		return nil, err
	}

	task["totalLikes"] = 0
	task["isLiked"] = false
	task["totalComments"] = 0

	assignedMemberInfo, err := member.GetAssignedMemberInfo(task, profileService)
	if err != nil {
		return nil, err
	}
	task["assignedMemberInfo"] = assignedMemberInfo

	reporterInfo, err := member.GetReporterInfo(task, profileService)
	if err != nil {
		return nil, err
	}
	task["reporterInfo"] = reporterInfo

	task["_id"] = result.InsertedID
	return util.SetResponse(task, 1, "Task added successfully to post."), nil
}

func addTaskInCollection(db *mongodatabase.DBConfig, profileService peoplerpc.AccountServiceClient, storageService storage.Service,
	cache *cache.Cache, collectionID string, task model.Task, profileID int,
) (map[string]interface{}, error) {
	// profileIDStr := strconv.Itoa(profileID)
	// isValid := permissions.CheckValidPermissions(profileIDStr, cache, boardID, []string{consts.Owner, consts.Author, consts.Admin}, false)
	// if !isValid {
	// 	return util.SetResponse(nil, 0, "User does not have access to the board."), nil
	// }

	dbconn, err := db.New(consts.Collection)
	if err != nil {
		return nil, err
	}
	Collection, collectionClient := dbconn.Collection, dbconn.Client
	defer collectionClient.Disconnect(context.TODO())

	collectionObjID, err := primitive.ObjectIDFromHex(collectionID)
	if err != nil {
		return nil, err
	}

	// find collection filter
	var collection model.Collection
	err = Collection.FindOne(context.TODO(), bson.M{"_id": collectionObjID}).Decode(&collection)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find collection")
	}

	dbconn2, err := db.New(consts.Task)
	if err != nil {
		return nil, err
	}
	taskCollection, taskClient := dbconn2.Collection, dbconn2.Client
	defer taskClient.Disconnect(context.TODO())

	// fixed
	task.Id = primitive.NewObjectID()
	// task.CreateDate = time.Now()
	// task.ModifiedDate = time.Now()
	task.Type = cases.Upper(language.English).String(consts.Task)
	// task.Owner = strconv.Itoa(profileID)
	// task.CollectionID = collectionObjID

	// add owner info
	// cp, err := profileService.FetchConciseProfile(profileID, storageService)
	// if err != nil {
	// 	return nil, errors.Wrap(err, "unable to find owner's info.")
	// }
	// task.OwnerInfo = cp
	// task.OwnerInfo.Id = profileID

	// add assigned member info
	if task.AssignedToID != "" {
		assignedIdInt, _ := strconv.Atoi(task.AssignedToID)
		// memberInfo, err := profileService.FetchConciseProfile(assignedIdInt)
		// if err != nil {
		// 	return nil, errors.Wrap(err, "unable to find assignee's info.")
		// }

		cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(assignedIdInt)}
		memberInfo, err := profileService.GetConciseProfile(context.TODO(), cpreq)
		if err != nil {
			return nil, errors.Wrap(err, "unable to find assignee's info.")
		}

		task.AssignedMemberInfo = memberInfo
		task.AssignedMemberInfo.Id = int32(assignedIdInt)
	} else {
		task.AssignedMemberInfo = nil
	}

	_, err = taskCollection.InsertOne(context.TODO(), task)
	if err != nil {
		return nil, errors.Wrap(err, "unable to added task in Mongo.")
	}
	return util.SetResponse(task, 1, "Task added successfully."), nil
}

func updateTask(db *mongodatabase.DBConfig, cache *cache.Cache, payload map[string]interface{}, boardID, postID, taskID string, profileID int) (map[string]interface{}, error) {
	postObjID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return nil, err
	}

	taskObjID, err := primitive.ObjectIDFromHex(taskID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to decode object ID.")
	}

	dbconn2, err := db.New(consts.Task)
	if err != nil {
		return nil, err
	}

	taskCollection, taskClient := dbconn2.Collection, dbconn2.Client
	defer taskClient.Disconnect(context.TODO())

	taskFilter := bson.M{"_id": taskObjID}

	// update task filter
	payload["_id"] = taskObjID
	payload["postID"] = postObjID
	payload["modifiedDate"] = time.Now()
	_, err = taskCollection.UpdateOne(context.TODO(), taskFilter, bson.M{"$set": payload})
	if err != nil {
		return nil, errors.Wrap(err, "unable to update task at mongo")
	}

	// get the updated task
	var updatedTask map[string]interface{}
	err = taskCollection.FindOne(context.TODO(), taskFilter).Decode(&updatedTask)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find updated task")
	}

	if updatedTask["likes"] != nil {
		updatedTask["totalLikes"] = len(updatedTask["likes"].(primitive.A))

		var likes []string
		for _, value := range updatedTask["likes"].(primitive.A) {
			likes = append(likes, value.(string))
		}

		if util.Contains(likes, fmt.Sprint(profileID)) {
			updatedTask["isLiked"] = true
		} else {
			updatedTask["isLiked"] = false
		}
	} else {
		updatedTask["totalLikes"] = 0
		updatedTask["isLiked"] = false
	}

	if updatedTask["comments"] != nil {
		updatedTask["totalComments"] = len(updatedTask["comments"].(primitive.A))
	} else {
		updatedTask["totalComments"] = 0
	}

	return util.SetResponse(updatedTask, 1, "Task updated successfully."), nil
}

func deleteTask(db *mongodatabase.DBConfig, cache *cache.Cache, boardID, taskID string, profileID int) (map[string]interface{}, error) {
	dbconn, err := db.New(consts.Task)
	if err != nil {
		return nil, err
	}
	taskColl, taskClient := dbconn.Collection, dbconn.Client
	defer taskClient.Disconnect(context.TODO())

	dbconn2, err := db.New(consts.Trash)
	if err != nil {
		return nil, err
	}

	trashColl, trashClient := dbconn2.Collection, dbconn2.Client
	defer trashClient.Disconnect(context.TODO())

	taskObjID, err := primitive.ObjectIDFromHex(taskID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to objectID")
	}

	filter := bson.M{"_id": taskObjID}

	// task to delete
	var tasktoDelete map[string]interface{}
	err = taskColl.FindOne(context.TODO(), filter).Decode(&tasktoDelete)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find task")
	}

	_, err = trashColl.InsertOne(context.TODO(), tasktoDelete)
	if err != nil {
		return nil, errors.Wrap(err, "unable to insert into Trash collection")
	}

	_, err = taskColl.DeleteOne(context.TODO(), filter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to delete task")
	}

	return util.SetResponse(tasktoDelete, 1, "Task deleted successfully."), nil
}

func getTaskByID(db *mongodatabase.DBConfig, profileService peoplerpc.AccountServiceClient, storageService storage.Service,
	taskID string, profileID int,
) (map[string]interface{}, error) {
	dbconn, err := db.New(consts.Task)
	if err != nil {
		return nil, err
	}

	taskCollection, taskClient := dbconn.Collection, dbconn.Client
	defer taskClient.Disconnect(context.TODO())

	taskObjID, err := primitive.ObjectIDFromHex(taskID)
	if err != nil {
		return nil, err
	}

	// find task filter
	findTaskFilter := bson.M{"_id": taskObjID}

	var task map[string]interface{}
	err = taskCollection.FindOne(context.TODO(), findTaskFilter).Decode(&task)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find task from the board.")
	}

	assignedMemberInfo, err := member.GetAssignedMemberInfo(task, profileService)
	if err != nil {
		return nil, err
	}
	task["assignedMemberInfo"] = assignedMemberInfo

	reporterInfo, err := member.GetReporterInfo(task, profileService)
	if err != nil {
		return nil, err
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

	return util.SetResponse(task, 1, "Task fetched successfully."), nil
}

func fetchTasksByProfile(cache *cache.Cache, db *mongodatabase.DBConfig, mysql *database.Database,
	boardID string, profileID, limit int, publicOnly bool,
) (map[string]interface{}, error) {
	profileIDStr := strconv.Itoa(profileID)
	dbConn, err := db.New(consts.Task)
	if err != nil {
		return nil, err
	}
	taskCollection, taskClient := dbConn.Collection, dbConn.Client
	defer taskClient.Disconnect(context.TODO())
	var curr *mongo.Cursor
	var findFilter primitive.M
	allTasks := make(map[string][]*model.Task)
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
		tasks, err := fetchTasksByFilter(taskCollection, mysql, curr, findFilter, limit)
		if err != nil {
			errChan <- errors.Wrap(err, "unable to fetch public tasks")
		}
		if len(tasks) > 0 {
			if publicOnly {
				res = tasks
			} else {
				allTasks["public"] = tasks
			}
		} else {
			if publicOnly {
				res = nil
			} else {
				allTasks["public"] = nil
			}
		}
		errChan <- nil
	}(errChan)
	if !publicOnly {
		go func(errChan chan<- error) {
			defer util.RecoverGoroutinePanic(errChan)
			findFilter = bson.M{"owner": profileIDStr, "boardID": boardObjID}
			tasks, err := fetchTasksByFilter(taskCollection, mysql, curr, findFilter, limit)
			if err != nil {
				errChan <- errors.Wrap(err, "unable to fetch  private tasks")
			}

			if len(tasks) > 0 {
				allTasks["private"] = tasks
			} else {
				allTasks["private"] = nil
			}
			errChan <- nil
		}(errChan)
		go func(errChan chan<- error) {
			defer util.RecoverGoroutinePanic(errChan)
			findFilter = bson.M{"visible": "MEMBERS", "boardID": boardObjID}
			tasks, err := fetchTasksByFilter(taskCollection, mysql, curr, findFilter, limit)
			if err != nil {
				errChan <- errors.Wrap(err, "unable to fetch members tasks")
			}
			if len(tasks) > 0 {
				allTasks["members"] = tasks
			} else {
				allTasks["members"] = nil
			}
			errChan <- nil
		}(errChan)

		go func(errChan chan<- error) {
			defer util.RecoverGoroutinePanic(errChan)
			// fetch profile connections
			dbConn, err := db.New(consts.Connection)
			if err != nil {
				errChan <- errors.Wrap(err, "unable to connect to mongo collection")
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
				// fetch tasks filter
				findFilter = bson.M{"visible": "CONTACTS", "boardID": boardObjID, "owner": bson.M{"$in": connectionArr}}
				tasks, err := fetchTasksByFilter(taskCollection, mysql, curr, findFilter, limit)
				if err != nil {
					errChan <- errors.Wrap(err, "unable to fetch contact tasks")
				}
				if len(tasks) > 0 {
					allTasks["contacts"] = tasks
				} else {
					allTasks["contacts"] = nil
				}
			} else {
				allTasks["contacts"] = nil
			}
			errChan <- nil
		}(errChan)
	}
	for i := 0; i < totalGoroutines; i++ {
		if err := <-errChan; err != nil {
			return nil, errors.Wrap(err, "error fromfetchTasksByProfile go-routine")
		}
	}
	if publicOnly {
		return util.SetResponse(res, 1, "Tasks fetched successfully."), nil
	}
	return util.SetResponse(allTasks, 1, "Tasks fetched successfully."), nil
}

func fetchTasksByFilter(taskCollection *mongo.Collection, mysql *database.Database, curr *mongo.Cursor, findFilter primitive.M, limit int) (tasks []*model.Task, err error) {
	findOptions := options.Find()
	findOptions.SetSort(bson.M{"createDate": -1})
	if limit != 0 {
		opts := options.Find().SetLimit(int64(limit))
		curr, err = taskCollection.Find(context.TODO(), findFilter, opts, findOptions)
	} else {
		curr, err = taskCollection.Find(context.TODO(), findFilter, findOptions)
	}
	if err != nil {
		return nil, errors.Wrap(err, "unable to find tasks")
	}
	err = curr.All(context.TODO(), &tasks)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch tasks")
	}
	// map owner profile
	errChan := make(chan error)
	for index := range tasks {
		go func(i int, errChan chan<- error) {
			defer util.RecoverGoroutinePanic(errChan)

			// ownerInfo := model.ConciseProfile{}
			// stmt := `SELECT id, firstName, lastName,
			// 				IFNULL(screenName, '') AS screenName,
			// 				IFNULL(photo, '') AS photo FROM` + "`sidekiq-dev`.AccountProfile WHERE id = ?"
			// itemOwner, _ := strconv.Atoi(tasks[i].Owner)
			// err = mysql.Conn.Get(&ownerInfo, stmt, itemOwner)
			// if err != nil {
			// 	errChan <- errors.Wrap(err, "unable to map profile info")
			// }
			// tasks[i].OwnerInfo = &ownerInfo
			errChan <- nil
		}(index, errChan)
	}
	totalGoroutines := len(tasks)
	for i := 0; i < totalGoroutines; i++ {
		if err := <-errChan; err != nil {
			return nil, errors.Wrap(err, "error fromfetchTasksByFilter go-routine")
		}
	}
	return
}

func fetchTasksByPost(db *mongodatabase.DBConfig, boardID, postID string, storageService storage.Service, profileService peoplerpc.AccountServiceClient) ([]map[string]interface{}, error) {
	dbconn, err := db.New(consts.Task)
	if err != nil {
		return nil, err
	}
	taskColl, noteClient := dbconn.Collection, dbconn.Client
	defer noteClient.Disconnect(context.TODO())

	postObjID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to objectID")
	}

	cur, err := taskColl.Find(context.TODO(), bson.M{"postID": postObjID})
	if err != nil {
		return nil, errors.Wrap(err, "notes of post not found")
	}

	var tasks []map[string]interface{}
	err = cur.All(context.TODO(), &tasks)
	if err != nil {
		return nil, errors.Wrap(err, "unable to unpack tasks")
	}

	for index := range tasks {
		assignedMemberInfo, err := member.GetAssignedMemberInfo(tasks[index], profileService)
		if err != nil {
			return nil, err
		}
		tasks[index]["assignedMemberInfo"] = assignedMemberInfo

		reporterInfo, err := member.GetReporterInfo(tasks[index], profileService)
		if err != nil {
			return nil, err
		}
		tasks[index]["reporterInfo"] = reporterInfo
	}

	return tasks, nil
}

func deleteTasksOnPost(db *mongodatabase.DBConfig, postID string) error {
	dbconn, err := db.New(consts.Task)
	if err != nil {
		return err
	}
	noteColl, noteClient := dbconn.Collection, dbconn.Client
	defer noteClient.Disconnect(context.TODO())

	postObjID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return errors.Wrap(err, "unable to convert string to objectID")
	}

	filter := bson.M{"postID": postObjID}

	_, err = noteColl.DeleteMany(context.TODO(), filter)
	if err != nil {
		return errors.Wrap(err, "unable to delete tasks on post")
	}

	return nil
}

func getActionTask(db *mongodatabase.DBConfig, cache *cache.Cache, profileService peoplerpc.AccountServiceClient, storageService storage.Service,
	profileID int, sortBy, orderBy string, limitInt, pageInt int, filterBy string,
) (map[string]interface{}, error) {
	dbconn, err := db.New(consts.Task)
	if err != nil {
		return nil, err
	}

	dbconn1, err := db.New(consts.Post)
	if err != nil {
		return nil, err
	}

	taskCollection, taskClient := dbconn.Collection, dbconn.Client
	defer taskClient.Disconnect(context.TODO())

	postCollection, postClient := dbconn1.Collection, dbconn1.Client
	defer postClient.Disconnect(context.TODO())

	var posts []model.Post
	cur, err := postCollection.Find(context.TODO(), bson.M{"owner": fmt.Sprint(profileID)})
	if err != nil {
		return nil, err
	}

	err = cur.All(context.TODO(), &posts)
	if err != nil {
		return nil, err
	}

	var postIDs []primitive.ObjectID
	for _, post := range posts {
		postIDs = append(postIDs, post.Id)
	}

	var findTaskFilter primitive.M
	if filterBy == "" {
		findTaskFilter = bson.M{"$or": []bson.M{
			{"assignedToID": fmt.Sprint(profileID)},
			{"postID": bson.M{"$in": postIDs}},
		}}
	} else {
		findTaskFilter = bson.M{"$and": []bson.M{
			{"$or": []bson.M{
				{"assignedToID": fmt.Sprint(profileID)},
				{"postID": bson.M{"$in": postIDs}},
			}},
			{"taskStatus": filterBy},
		}}
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
	findOptions.SetSort(bson.M{sortBy: filterorderBy})
	sortingOption := findOptions.SetSort(bson.M{sortBy: filterorderBy})
	collation := &options.Collation{
		Locale:   "en", // Set your desired locale.
		Strength: 2,    // Strength 2 for case-insensitive.
	}
	findOptions = sortingOption.SetCollation(collation)

	if pageInt > 0 {
		offset := (pageInt - 1) * limitInt
		findOptions.SetSkip(int64(offset))
		findOptions.SetLimit(int64(limitInt))
	} else {
		findOptions.SetLimit(10)
	}

	var tasks []map[string]interface{}
	finaltask := make([]map[string]interface{}, 0)

	cursor, err := taskCollection.Find(context.TODO(), findTaskFilter, findOptions)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find task")
	}
	defer cursor.Close(context.TODO())

	err = cursor.All(context.TODO(), &tasks)
	if err != nil {
		return nil, errors.Wrap(err, "unable to cursor")
	}

	for index := range tasks {

		var taskobjId primitive.ObjectID

		if objID, ok := tasks[index]["_id"].(primitive.ObjectID); ok {
			taskobjId = objID
		} else if objIDstr, ok := tasks[index]["_id"].(string); ok {
			taskobjId, err = primitive.ObjectIDFromHex(objIDstr)
			if err != nil {
				return nil, errors.Wrap(err, " Error from coverting ID")
			}
		}

		if postObjID, ok := tasks[index]["postID"].(primitive.ObjectID); ok {
			post, err := getPostDetailsByID(postCollection, postObjID)
			if err != nil {
				if errors.Is(err, mongo.ErrNoDocuments) {
					deleteTask(db, cache, "", taskobjId.Hex(), profileID)
				}
				continue
			}
			tasks[index]["boardID"] = post.BoardID
		} else if postIDstr, ok := tasks[index]["postID"].(string); ok {

			postObjID, err := primitive.ObjectIDFromHex(postIDstr)
			if err != nil {
				return nil, errors.Wrap(err, " Error from coverting ID")
			}

			post, err := getPostDetailsByID(postCollection, postObjID)
			if err != nil {
				if errors.Is(err, mongo.ErrNoDocuments) {
					if taskobjID, ok := tasks[index]["_id"].(primitive.ObjectID); ok {
						deleteTask(db, cache, "", taskobjID.Hex(), profileID)
					} else if taskobjID, ok := tasks[index]["_id"].(string); ok {
						deleteTask(db, cache, "", taskobjID, profileID)
					}
				}
				continue
			}
			tasks[index]["boardID"] = post.BoardID
		}

		assignedMemberInfo, err := member.GetAssignedMemberInfo(tasks[index], profileService)
		if err != nil {
			return nil, err
		}
		tasks[index]["assignedMemberInfo"] = assignedMemberInfo

		reporterInfo, err := member.GetReporterInfo(tasks[index], profileService)
		if err != nil {
			return nil, err
		}
		tasks[index]["reporterInfo"] = reporterInfo

		if tasks[index]["likes"] != nil {
			tasks[index]["totalLikes"] = len(tasks[index]["likes"].(primitive.A))

			var likes []string
			for _, value := range tasks[index]["likes"].(primitive.A) {
				likes = append(likes, value.(string))
			}

			if util.Contains(likes, fmt.Sprint(profileID)) {
				tasks[index]["isLiked"] = true
			} else {
				tasks[index]["isLiked"] = false
			}
		} else {
			tasks[index]["totalLikes"] = 0
			tasks[index]["isLiked"] = false
		}

		if tasks[index]["comments"] != nil {
			tasks[index]["totalComments"] = len(tasks[index]["comments"].(primitive.A))
		} else {
			tasks[index]["totalComments"] = 0
		}

		finaltask = append(finaltask, tasks[index])
	}

	total, err := taskCollection.CountDocuments(context.TODO(), findTaskFilter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find task count")
	}

	return util.SetPaginationResponse(finaltask, int(total), 1, "Action Tasks fetched successfully."), nil
}

func getPostDetailsByID(postColl *mongo.Collection, postID primitive.ObjectID) (*model.Post, error) {
	var post model.Post
	err := postColl.FindOne(context.TODO(), bson.M{"_id": postID}).Decode(&post)
	if err != nil {
		return nil, err
	}
	return &post, nil
}
