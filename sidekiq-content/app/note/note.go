package note

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/board"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/profile"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/storage"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/cache"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/consts"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/database"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/helper"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/mongodatabase"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/permissions"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/util"
	"github.com/ProImaging/sidekiq-backend/sidekiq-models/model"
	peoplerpc "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"
	"github.com/pkg/errors"

	// peoplerpc "github.com/sidekiq-people/proto/people"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func addNotes(db *mongodatabase.DBConfig, postID primitive.ObjectID, notes []interface{}) error {
	dbconn, err := db.New(consts.Note)
	if err != nil {
		return err
	}
	noteColl, noteClient := dbconn.Collection, dbconn.Client
	defer noteClient.Disconnect(context.TODO())

	errChan := make(chan error)
	goRoutines := 0

	fmt.Println("total notes: ", len(notes))

	for i := 0; i < len(notes); i++ {
		goRoutines++
		go func(i int, errChan chan<- error) {
			note := notes[i].(map[string]interface{})
			note["postID"] = postID
			note["type"] = "NOTE"
			if val, ok := note["body"]; ok {
				note["body_raw"] = util.RemoveHtmlTag(val.(string))
			}
			note["editBy"] = ""
			note["editDate"] = nil
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

	_, err = noteColl.InsertMany(context.TODO(), notes)
	if err != nil {
		return errors.Wrap(err, "unable to insert notes in post")
	}

	// what should be the response?
	return nil
}

func getNotesByBoard(db *mongodatabase.DBConfig, mysql *database.Database, cache *cache.Cache,
	boardService board.Service, profileService peoplerpc.AccountServiceClient, storageService storage.Service,
	boardID string, profileID int, owner string, tagArr []string, uploadDate string, limit int, l, page string,
) (map[string]interface{}, error) {
	var notes []*model.Note

	var err error
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
		return nil, err
	}

	var board *model.Board
	filter := bson.M{"_id": boardObjID}

	err = boardCollection.FindOne(context.TODO(), filter).Decode(&board)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find board")
	}

	dbconn2, err := db.New(consts.Note)
	if err != nil {
		return nil, err
	}
	noteCollection, noteClient := dbconn2.Collection, dbconn2.Client
	defer noteClient.Disconnect(context.TODO())

	var curr *mongo.Cursor
	var opts *options.FindOptions
	var isPaginated bool = false

	findNotesFilter := bson.M{"$and": bson.A{
		bson.M{"boardID": boardObjID},
		bson.M{"state": bson.M{"$ne": consts.Hidden}},
	}}
	if owner != "" {
		findNotesFilter["owner"] = owner
	}
	if len(tagArr) != 0 {
		findNotesFilter["tags"] = bson.M{"$all": tagArr}
	}
	if uploadDate != "" {
		copyDate := uploadDate
		custom := "2006-01-02T15:04:05Z"

		start := copyDate + "T00:00:00Z"
		dayStart, _ := time.Parse(custom, start)

		uploadDate = uploadDate + "T11:59:59Z"
		dayEnd, _ := time.Parse(custom, uploadDate)

		findNotesFilter["createDate"] = bson.M{"$gte": dayStart, "$lte": dayEnd}
	}

	findOptions := options.Find()
	findOptions.SetSort(bson.M{"createDate": -1})

	total, err := noteCollection.CountDocuments(context.TODO(), findNotesFilter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find total count")
	}

	if limit != 0 {
		opts = options.Find().SetLimit(int64(limit))
		curr, err = noteCollection.Find(context.TODO(), findNotesFilter, opts, findOptions)
	} else if limit == 0 && l != "" && page != "" {
		pgInt, _ := strconv.Atoi(page)
		limitInt, _ := strconv.Atoi(l)
		offset := (pgInt - 1) * limitInt
		findOptions.SetSkip(int64(offset))
		findOptions.SetLimit(int64(limitInt))
		curr, err = noteCollection.Find(context.TODO(), findNotesFilter, findOptions)
		isPaginated = true
	}
	defer curr.Close(context.TODO())
	if err != nil {
		return nil, errors.Wrap(err, "unable to find notes")
	}

	err = curr.All(context.TODO(), &notes)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find notes")
	}
	if len(notes) == 0 {
		return util.SetPaginationResponse([]*model.Note{}, 0, 1, "Board contains no notes. Please add one."), nil
	}

	for _, note := range notes {
		noteOwner, _ := strconv.Atoi(note.Owner)
		// ownerInfo, err := profileService.FetchConciseProfile(noteOwner)

		cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(noteOwner)}
		ownerInfo, err := profileService.GetConciseProfile(context.TODO(), cpreq)
		if err != nil {
			return nil, errors.Wrap(err, "unable to find basic info")
		}

		ownerInfo.Id = int32(noteOwner)
		note.OwnerInfo = ownerInfo

		// reaction count
		if util.Contains(note.Likes, profileIDStr) {
			note.IsLiked = true
		} else {
			note.IsLiked = false
		}
		note.TotalComments = len(note.Comments)
		note.TotalLikes = len(note.Likes)

		// get location
		loc, err := boardService.GetThingLocationOnBoard(note.BoardID.Hex())
		if err != nil {
			continue
		}
		note.Location = loc
	}

	if isPaginated {
		return util.SetPaginationResponse(notes, int(total), 1, "Notes fetched successfully"), nil
	}
	return util.SetResponse(notes, 1, "Notes fetched successfully"), nil
}

func addNote(db *mongodatabase.DBConfig, postID primitive.ObjectID, note interface{}) (map[string]interface{}, error) {
	dbconn, err := db.New(consts.Note)
	if err != nil {
		return nil, err
	}
	noteColl, noteClient := dbconn.Collection, dbconn.Client
	defer noteClient.Disconnect(context.TODO())

	notemap := note.(map[string]interface{})
	notemap["postID"] = postID
	notemap["type"] = "NOTE"
	notemap["comments"] = nil
	notemap["likes"] = nil
	notemap["state"] = consts.Active
	notemap["createDate"] = time.Now()
	notemap["editBy"] = ""
	notemap["editDate"] = nil

	if val, ok := notemap["body"]; ok {
		notemap["body_raw"] = util.RemoveHtmlTag(val.(string))
	}

	result, err := noteColl.InsertOne(context.TODO(), notemap)
	if err != nil {
		return nil, errors.Wrap(err, "unable to insert note in post")
	}

	notemap["_id"] = result.InsertedID
	notemap["totalLikes"] = 0
	notemap["isLiked"] = false
	notemap["totalComments"] = 0

	return util.SetResponse(notemap, 1, "Note added successfully to post."), nil
}

func addNoteInCollection(cache *cache.Cache, profileService peoplerpc.AccountServiceClient, storageService storage.Service,
	db *mongodatabase.DBConfig, note model.Note, collectionID string, profileID int,
) (map[string]interface{}, error) {
	// check if the user has valid permission to add a note
	// profileIDStr := strconv.Itoa(profileID)
	// isValid := permissions.CheckValidPermissions(profileIDStr, cache, boardID, []string{consts.Owner, consts.Admin, consts.Author}, false)
	// if !isValid {
	// 	return util.SetResponse(nil, 0, "User does not have access to the board."), nil
	// }

	dbconn, err := db.New(consts.Collection)
	if err != nil {
		return nil, err
	}

	collection, collectionClient := dbconn.Collection, dbconn.Client
	defer collectionClient.Disconnect(context.TODO())

	collectionObjID, err := primitive.ObjectIDFromHex(collectionID)
	if err != nil {
		return nil, err
	}

	// find board filter
	filter := bson.M{"_id": collectionObjID} // not adding isActive bit as if board is deleted, there won't be permission in redis, so would return error from role permissions

	var result model.Collection
	err = collection.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find board")
	}

	dbconn2, err := db.New(consts.Note)
	if err != nil {
		return nil, err
	}
	noteCollection, noteClient := dbconn2.Collection, dbconn2.Client
	defer noteClient.Disconnect(context.TODO())

	// fixed
	note.Id = primitive.NewObjectID()
	note.CollectionID = collectionObjID
	note.CreateDate = time.Now()
	note.ModifiedDate = time.Now()
	note.Type = cases.Upper(language.English).String(consts.Note)
	note.Owner = strconv.Itoa(profileID)
	note.Body_raw = util.RemoveHtmlTag(note.Body)

	// add owner info
	// cp, err := profileService.FetchConciseProfile(profileID)

	cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(profileID)}
	cp, err := profileService.GetConciseProfile(context.TODO(), cpreq)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find basic info")
	}

	note.OwnerInfo = cp
	note.OwnerInfo.Id = int32(profileID)

	_, err = noteCollection.InsertOne(context.TODO(), note)
	if err != nil {
		return nil, errors.Wrap(err, "unable to insert note at mongo")
	}

	return util.SetResponse(note, 1, "Note inserted successfully"), nil
}

func updateNote(cache *cache.Cache, db *mongodatabase.DBConfig, payload map[string]interface{}, boardID, postID, noteID string, profileID int) (map[string]interface{}, error) {
	noteObjID, err := primitive.ObjectIDFromHex(noteID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert to ObjID")
	}

	dbconn2, err := db.New(consts.Note)
	if err != nil {
		return nil, err
	}
	noteCollection, noteClient := dbconn2.Collection, dbconn2.Client
	defer noteClient.Disconnect(context.TODO())

	updateNoteFilter := bson.M{"_id": noteObjID}

	if payload["body"] != nil {
		payload["body_raw"] = util.RemoveHtmlTag(payload["body"].(string))
	}

	payload["modifiedDate"] = time.Now()

	_, err = noteCollection.UpdateOne(context.TODO(), updateNoteFilter, bson.M{"$set": payload})
	if err != nil {
		return nil, errors.Wrap(err, "unable to update note at mongo")
	}

	// get the updated note
	var updatedNote map[string]interface{}
	err = noteCollection.FindOne(context.TODO(), updateNoteFilter).Decode(&updatedNote)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find updated note")
	}

	if updatedNote["state"] == consts.Hidden {
		return util.SetResponse(nil, 1, "The requested note is deleted"), nil
	}

	if updatedNote["likes"] != nil {
		updatedNote["totalLikes"] = len(updatedNote["likes"].(primitive.A))

		var likes []string
		for _, value := range updatedNote["likes"].(primitive.A) {
			likes = append(likes, value.(string))
		}

		if util.Contains(likes, fmt.Sprint(profileID)) {
			updatedNote["isLiked"] = true
		} else {
			updatedNote["isLiked"] = false
		}
	} else {
		updatedNote["totalLikes"] = 0
		updatedNote["isLiked"] = false
	}

	if updatedNote["comments"] != nil {
		updatedNote["totalComments"] = len(updatedNote["comments"].(primitive.A))
	} else {
		updatedNote["totalComments"] = 0
	}

	return util.SetResponse(updatedNote, 1, "Note updated Successfully"), nil
}

func deleteNote(cache *cache.Cache, db *mongodatabase.DBConfig, boardID, noteID string, profileID int) (map[string]interface{}, error) {
	dbconn, err := db.New(consts.Note)
	if err != nil {
		return nil, err
	}
	noteColl, noteClient := dbconn.Collection, dbconn.Client
	defer noteClient.Disconnect(context.TODO())

	dbconn2, err := db.New(consts.Trash)
	if err != nil {
		return nil, err
	}

	trashColl, trashClient := dbconn2.Collection, dbconn2.Client
	defer trashClient.Disconnect(context.TODO())

	noteObjID, err := primitive.ObjectIDFromHex(noteID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to objectID")
	}

	filter := bson.M{"_id": noteObjID}

	// get the note object
	var noteTOdelete map[string]interface{}
	err = noteColl.FindOne(context.TODO(), filter).Decode(&noteTOdelete)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch note")
	}

	_, err = trashColl.InsertOne(context.TODO(), noteTOdelete)
	if err != nil {
		return nil, errors.Wrap(err, "unable to insert into Trash collection")
	}

	_, err = noteColl.DeleteOne(context.TODO(), filter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to delete note")
	}

	return util.SetResponse(noteTOdelete, 1, "Note deleted successfully."), nil
}

func getNoteByID(db *mongodatabase.DBConfig, profileService profile.Service, storageService storage.Service,
	noteID string, profileID int,
) (map[string]interface{}, error) {
	dbconn, err := db.New(consts.Note)
	if err != nil {
		return nil, err
	}

	noteCollection, noteClient := dbconn.Collection, dbconn.Client
	defer noteClient.Disconnect(context.TODO())

	noteObjID, err := primitive.ObjectIDFromHex(noteID)
	if err != nil {
		return nil, err
	}

	// find note filter
	findNoteFilter := bson.M{"_id": noteObjID}

	var note map[string]interface{}
	err = noteCollection.FindOne(context.TODO(), findNoteFilter).Decode(&note)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find note")
	}

	if note["state"] == consts.Hidden {
		return util.SetResponse(nil, 1, "The requested note is deleted"), nil
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

	if note["comments"] != nil {
		note["totalComments"] = len(note["comments"].(primitive.A))
	} else {
		note["totalComments"] = 0
	}

	return util.SetResponse(note, 1, "Note fetched successfully"), nil
}

func fetchNotesByProfile(cache *cache.Cache, db *mongodatabase.DBConfig, mysql *database.Database,
	boardID string, profileID, limit int, publicOnly bool,
) (map[string]interface{}, error) {
	profileIDStr := strconv.Itoa(profileID)
	dbConn, err := db.New(consts.Note)
	if err != nil {
		return nil, err
	}
	noteCollection, noteClient := dbConn.Collection, dbConn.Client
	defer noteClient.Disconnect(context.TODO())
	var curr *mongo.Cursor
	var findFilter primitive.M
	allNotes := make(map[string][]*model.Note)
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
		notes, err := fetchNotesByFilter(noteCollection, mysql, curr, findFilter, limit)
		if err != nil {
			errChan <- errors.Wrap(err, "unable to fetch public notes")
		}
		if len(notes) > 0 {
			if publicOnly {
				res = notes
			} else {
				allNotes["public"] = notes
			}
		} else {
			if publicOnly {
				res = nil
			} else {
				allNotes["public"] = nil
			}
		}
		errChan <- nil
	}(errChan)
	if !publicOnly {
		go func(errChan chan<- error) {
			defer util.RecoverGoroutinePanic(errChan)
			findFilter = bson.M{"owner": profileIDStr, "boardID": boardObjID}
			notes, err := fetchNotesByFilter(noteCollection, mysql, curr, findFilter, limit)
			fmt.Println("Err", err)
			if err != nil {
				errChan <- errors.Wrap(err, " unable to fetch private notes")
			}
			if len(notes) > 0 {
				allNotes["private"] = notes
			} else {
				allNotes["private"] = nil
			}
			errChan <- nil
		}(errChan)
		go func(errChan chan<- error) {
			defer util.RecoverGoroutinePanic(errChan)
			findFilter = bson.M{"visible": "MEMBERS", "boardID": boardObjID}
			notes, err := fetchNotesByFilter(noteCollection, mysql, curr, findFilter, limit)
			if err != nil {
				errChan <- errors.Wrap(err, "unable to fetch members notes")
			}
			if len(notes) > 0 {
				allNotes["members"] = notes
			} else {
				allNotes["members"] = nil
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
				// fetch notes filter
				findFilter = bson.M{"visible": "CONTACTS", "boardID": boardObjID, "owner": bson.M{"$in": connectionArr}}
				notes, err := fetchNotesByFilter(noteCollection, mysql, curr, findFilter, limit)
				if err != nil {
					errChan <- errors.Wrap(err, "unable to fetch contact notes")
				}
				if len(notes) > 0 {
					allNotes["contacts"] = notes
				} else {
					allNotes["contacts"] = nil
				}
			} else {
				allNotes["contacts"] = nil
			}
			errChan <- nil
		}(errChan)
	}
	for i := 0; i < totalGoroutines; i++ {
		if err := <-errChan; err != nil {
			return nil, errors.Wrap(err, "error fromfetchNotesByProfile go-routine")
		}
	}
	if publicOnly {
		return util.SetResponse(res, 1, "notes fetched successfully."), nil
	}
	return util.SetResponse(allNotes, 1, "notes fetched successfully."), nil
}

func fetchNotesByFilter(noteCollection *mongo.Collection, mysql *database.Database, curr *mongo.Cursor, findFilter primitive.M, limit int) (notes []*model.Note, err error) {
	findOptions := options.Find()
	findOptions.SetSort(bson.M{"createDate": -1})

	if limit != 0 {
		opts := options.Find().SetLimit(int64(limit))
		curr, err = noteCollection.Find(context.TODO(), findFilter, opts, findOptions)
	} else {
		curr, err = noteCollection.Find(context.TODO(), findFilter, findOptions)
	}

	if err != nil {
		return nil, errors.Wrap(err, "unable to find notes")
	}

	err = curr.All(context.TODO(), &notes)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch notes")
	}
	// map owner profile
	errChan := make(chan error)
	for index := range notes {
		go func(i int, errChan chan<- error) {
			defer util.RecoverGoroutinePanic(errChan)

			ownerInfo := model.ConciseProfile{}
			stmt := `SELECT id, firstName, lastName,
							IFNULL(screenName, '') AS screenName,
							IFNULL(photo, '') AS photo FROM` + "`sidekiq-dev`.AccountProfile WHERE id = ?"
			itemOwner, _ := strconv.Atoi(notes[i].Owner)
			err = mysql.Conn.Get(&ownerInfo, stmt, itemOwner)
			if err != nil {
				errChan <- errors.Wrap(err, "unable to map profile info")
			}
			notes[i].OwnerInfo = &peoplerpc.ConciseProfileReply{
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
	totalGoroutines := len(notes)
	for i := 0; i < totalGoroutines; i++ {
		if err := <-errChan; err != nil {
			return nil, errors.Wrap(err, "error fromfetchNotesByFilter go-routine")
		}
	}
	return
}

func fetchNotesByPost(db *mongodatabase.DBConfig, boardID, postID string) ([]map[string]interface{}, error) {
	dbconn, err := db.New(consts.Note)
	if err != nil {
		return nil, err
	}
	noteColl, noteClient := dbconn.Collection, dbconn.Client
	defer noteClient.Disconnect(context.TODO())

	postObjID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to objectID")
	}

	cur, err := noteColl.Find(context.TODO(), bson.M{"postID": postObjID})
	if err != nil {
		return nil, errors.Wrap(err, "notes of post not found")
	}

	var notes []map[string]interface{}
	err = cur.All(context.TODO(), &notes)
	if err != nil {
		return nil, errors.Wrap(err, "unable to unpack notes")
	}

	return notes, nil
}

func fetchCollectionByPost(db *mongodatabase.DBConfig, boardID, postID string, profileService peoplerpc.AccountServiceClient, storageService storage.Service, profileID int) ([]map[string]interface{}, error) {
	boardObjectID, err := primitive.ObjectIDFromHex(boardID)
	if err != nil {
		return nil, err
	}

	postObjID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to objectID")
	}

	dbconn, err := db.New(consts.Collection)
	if err != nil {
		return nil, err
	}
	Coll, ColClient := dbconn.Collection, dbconn.Client
	defer ColClient.Disconnect(context.TODO())

	dbconn2, err := db.New(consts.File)
	if err != nil {
		return nil, errors.Wrap(err, "unable to establish connection with File collection.")
	}

	fileCollection, fileClient := dbconn2.Collection, dbconn2.Client
	defer fileClient.Disconnect(context.TODO())

	dbconn3, err := db.New(consts.Board)
	if err != nil {
		return nil, errors.Wrap(err, "unable to establish connection with File collection.")
	}

	boardCollection, boardClient := dbconn3.Collection, dbconn3.Client
	defer boardClient.Disconnect(context.TODO())

	dbconn4, err := db.New(consts.Post)
	if err != nil {
		return nil, err
	}

	postCol, postClient := dbconn4.Collection, dbconn4.Client
	defer postClient.Disconnect(context.TODO())

	var board model.Board
	err = boardCollection.FindOne(context.TODO(), bson.M{"_id": boardObjectID}).Decode(&board)
	if err != nil {
		return nil, err
	}

	profileStr := strconv.Itoa(profileID)

	boardOwnerInt, err := strconv.Atoi(board.Owner)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to int")
	}

	// boardownerInfo, err := profileService.FetchConciseProfile(boardOwnerInt)

	cpreq := &peoplerpc.ConciseProfileRequest{ProfileId: int32(boardOwnerInt)}
	boardownerInfo, err := profileService.GetConciseProfile(context.TODO(), cpreq)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find basic info")
	}

	var post model.Post
	err = postCol.FindOne(context.TODO(), bson.M{"_id": postObjID}).Decode(&post)
	if err != nil {
		return nil, err
	}

	cur, err := Coll.Find(context.TODO(), bson.M{"postID": postObjID})
	if err != nil {
		return nil, errors.Wrap(err, "collection of post not found")
	}

	var collections []map[string]interface{}
	err = cur.All(context.TODO(), &collections)
	if err != nil {
		return nil, errors.Wrap(err, "unable to unpack collection")
	}

	for idx := range collections {
		var files []*model.UploadedFile
		collectionID := collections[idx]["_id"].(primitive.ObjectID)
		curr, err := fileCollection.Find(context.TODO(), bson.M{"collectionID": collectionID})
		if err != nil {
			continue
		}

		err = curr.All(context.TODO(), &files)
		if err != nil {
			continue
		}

		for fileidx := range files {
			key := util.GetKeyForPostCollectionMedia(int(boardownerInfo.AccountID), int(boardownerInfo.Id), boardID, postID, collectionID.Hex(), "")
			fileName := fmt.Sprintf("%s%s", files[fileidx].Id.Hex(), files[fileidx].FileExt)
			f, err := storageService.GetUserFile(key, fileName)
			if err != nil {
				return nil, errors.Wrap(err, "unable to presign image")
			}
			files[fileidx].URL = f.Filename

			thumbKey := util.GetKeyForPostCollectionMedia(int(boardownerInfo.AccountID), int(boardownerInfo.Id), boardID, postID, collectionID.Hex(), "thumbs")
			thumbfileName := files[fileidx].Id.Hex() + ".png"
			thumbs, err := helper.GetThumbnails(storageService, thumbKey, thumbfileName, []string{})
			if err != nil {
				thumbs = model.Thumbnails{}
			}

			files[fileidx].Thumbs = thumbs
			// reaction count
			if util.Contains(files[fileidx].Likes, profileStr) {
				files[fileidx].IsLiked = true
			} else {
				files[fileidx].IsLiked = false
			}
			files[fileidx].TotalComments = len(files[fileidx].Comments)
			files[fileidx].TotalLikes = len(files[fileidx].Likes)
		}

		collections[idx]["files"] = files
	}

	return collections, nil
}

func deleteNotesOnPost(db *mongodatabase.DBConfig, postID string) error {
	dbconn, err := db.New(consts.Note)
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
		return errors.Wrap(err, "unable to delete notes on post")
	}

	return nil
}
