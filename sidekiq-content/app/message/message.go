package message

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/cache"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/database"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/mongodatabase"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/permissions"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-models/model"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-models/model/notification"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func getCacheKey(groupID string) string {
	return fmt.Sprintf("group:%s", groupID)
}

func getPushSubscriptionsByGroupMembers(db *database.Database, groupMembers []string) ([]*notification.PushSubscription, error) {
	subs := []*notification.PushSubscription{}
	if groupMembers == nil || len(groupMembers) < 1 {
		return subs, nil
	}

	args := make([]interface{}, len(groupMembers))
	for i, id := range groupMembers {
		intId, err := strconv.Atoi(id)
		if err != nil {
			return nil, err
		}
		args[i] = intId
	}

	stmt := "SELECT id, profileID, type, endpoint, p256dh, auth, expirationTime, createDate FROM `sidekiq-dev`.PushSubscriptions WHERE profileID IN (?" + strings.Repeat(",?", len(args)-1) + ");"

	err := db.Conn.Select(&subs, stmt, args...)
	if err != nil {
		return nil, err
	}
	return subs, nil
}

func getApplePushSubscriptionsByGroupMembers(db *database.Database, groupMembers []string) ([]*notification.ApplePushSubscription, error) {
	subs := []*notification.ApplePushSubscription{}
	if groupMembers == nil || len(groupMembers) < 1 {
		return subs, nil
	}

	args := make([]interface{}, len(groupMembers))
	for i, id := range groupMembers {
		intId, err := strconv.Atoi(id)
		if err != nil {
			return nil, err
		}
		args[i] = intId
	}

	stmt := "SELECT id, profileID, type, deviceToken, expirationTime, createDate FROM `sidekiq-dev`.PushSubscriptionsApple WHERE profileID IN (?" + strings.Repeat(",?", len(args)-1) + ");"

	err := db.Conn.Select(&subs, stmt, args...)
	if err != nil {
		return nil, err
	}
	return subs, nil
}

func getGroupFromDB(db *mongodatabase.DBConfig, groupID string) (*model.ChatGroup, error) {
	var group model.ChatGroup
	groupId, err := primitive.ObjectIDFromHex(groupID)
	if err != nil {
		return nil, err
	}
	dbconn, err := db.New("ChatGroups")
	if err != nil {
		return nil, err
	}
	groupCollection, groupClient := dbconn.Collection, dbconn.Client
	defer groupClient.Disconnect(context.TODO())
	filter := bson.D{{"_id", groupId}}
	fmt.Println(filter)
	curr := groupCollection.FindOne(context.TODO(), filter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find group")
	}
	err = curr.Decode(&group)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch group")
	}
	return &group, nil
}

func getMessagesByGroup(db *mongodatabase.DBConfig, groupID string, userID int) ([]*model.Chat, error) {
	var messages []*model.Chat

	dbconn, err := db.New("ChatGroups")
	if err != nil {
		return nil, err
	}
	groupCollection, groupClient := dbconn.Collection, dbconn.Client
	defer groupClient.Disconnect(context.TODO())
	user := strconv.Itoa(userID)
	group, err := primitive.ObjectIDFromHex(groupID)
	if err != nil {
		return nil, err
	}
	filter := bson.D{{"$and", []interface{}{
		bson.D{{"_id", group}}, bson.D{{"$or", []interface{}{
			bson.D{{"admin", user}},
			bson.D{{"authors", user}}}}},
	}}}
	fmt.Println(filter)
	cursor := groupCollection.FindOne(context.TODO(), filter)
	if cursor.Err() != nil {
		return nil, errors.Wrap(cursor.Err(), "unable to find group")
	}

	dbconn2, err := db.New("Chats")
	if err != nil {
		return nil, err
	}
	messageCollection, messageClient := dbconn2.Collection, dbconn2.Client
	defer messageClient.Disconnect(context.TODO())
	findOptions := options.Find()
	// Sort by `price` field descending
	findOptions.SetSort(bson.D{{"createDate", -1}})

	filter2 := bson.D{{"groupId", group}}

	curr, err := messageCollection.Find(context.TODO(), filter2, findOptions)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find messages")
	}
	err = curr.All(context.TODO(), &messages)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch messages")
	}
	return messages, nil
}

func addMessage(cache *cache.Cache, db *mongodatabase.DBConfig, payload model.Chat, profileID int) (*model.Chat, error) {
	// check if the user has valid permissions on the board
	user := strconv.Itoa(profileID)
	userKey := fmt.Sprintf("boards:%s", user)
	boardPermission := permissions.GetBoardPermissions(userKey, cache)
	role := boardPermission[payload.BoardID.Hex()]
	if role == "" {
		return nil, errors.New("User do not have access to the board")
	}

	dbconn, err := db.New("ChatGroup")
	if err != nil {
		return nil, err
	}

	groupCollection, groupClient := dbconn.Collection, dbconn.Client
	defer groupClient.Disconnect(context.TODO())

	groupObjID, err := primitive.ObjectIDFromHex(payload.GroupID.Hex())
	if err != nil {
		return nil, err
	}

	// find group filter
	filter := bson.D{{"$and", []interface{}{
		bson.D{{"_id", groupObjID}},
		bson.D{{"members", user}},
		bson.D{{"isActive", true}},
	}}}

	var result model.ChatGroup
	err = groupCollection.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find group")
	}

	dbconn2, err := db.New("Chat")
	if err != nil {
		return nil, err
	}

	messageCollection, messageClient := dbconn2.Collection, dbconn2.Client
	defer messageClient.Disconnect(context.TODO())

	payload.Id = primitive.NewObjectID()
	payload.Sender = user
	payload.CreateDate = time.Now()
	payload.LastModifiedDate = time.Now()

	lastInsertedID, err := messageCollection.InsertOne(context.TODO(), payload)
	if err != nil {
		return nil, errors.Wrap(err, "unable to insert message at mongo")
	}
	fmt.Println(lastInsertedID)

	// better the response
	res := make(map[string]interface{})
	resData := make(map[string]interface{})
	resData["id"] = payload.Id.Hex()
	resData["error"] = false
	resData["msg"] = "Chat Inserted Successfully"
	res["data"] = resData
	return &payload, nil
}

func updateMessage(cache *cache.Cache, db *mongodatabase.DBConfig, payload model.Chat, profileID int) (map[string]interface{}, error) {
	// check if user has valid permissions on the board
	user := strconv.Itoa(profileID)
	userKey := fmt.Sprintf("boards:%s", user)
	boardPermission := permissions.GetBoardPermissions(userKey, cache)
	role := boardPermission[payload.BoardID.Hex()]
	if role == "" {
		return nil, errors.New("User do not have access to the board")
	}

	dbconn, err := db.New("ChatGroup")
	if err != nil {
		return nil, err
	}

	groupCollection, groupClient := dbconn.Collection, dbconn.Client
	defer groupClient.Disconnect(context.TODO())

	group, err := primitive.ObjectIDFromHex(payload.GroupID.Hex())
	if err != nil {
		return nil, err
	}

	// find group filter
	filter := bson.D{{"$and", []interface{}{
		bson.D{{"_id", group}},
		bson.D{{"members", user}},
		bson.D{{"isActive", true}},
	}}}

	var result model.ChatGroup
	err = groupCollection.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find group")
	}

	dbconn2, err := db.New("Chat")
	if err != nil {
		return nil, err
	}

	messageCollection, messageClient := dbconn2.Collection, dbconn2.Client
	defer messageClient.Disconnect(context.TODO())

	messageID, err := primitive.ObjectIDFromHex(payload.Id.Hex())
	if err != nil {
		return nil, err
	}

	// find chat filter
	filter = bson.D{{"$and", []interface{}{
		bson.D{{"_id", messageID}},
		bson.D{{"sender", user}},
		bson.D{{"isActive", true}},
	}}}

	var resultChat model.Chat

	err = messageCollection.FindOne(context.TODO(), filter).Decode(&resultChat)
	if err != nil {
		return nil, errors.Wrap(err, "User don't have permission to update the message")
	}
	if payload.Sender != user {
		return nil, errors.New("User don't have permission to update the message")
	}

	payload.LastModifiedDate = time.Now()

	update := bson.D{{"$set", payload}}
	_, err = messageCollection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to update your message")
	}

	// better the response
	res := make(map[string]interface{})
	resData := make(map[string]interface{})
	resData["value"] = payload
	resData["error"] = false
	resData["msg"] = "Chat updated Successfully"
	res["data"] = resData
	return res, nil
}

func deleteMessage(cache *cache.Cache, db *mongodatabase.DBConfig, Id string, profileID int) (map[string]interface{}, error) {
	dbconn, err := db.New("Chat")
	if err != nil {
		return nil, err
	}

	messageCollection, messageClient := dbconn.Collection, dbconn.Client
	defer messageClient.Disconnect(context.TODO())

	user := strconv.Itoa(profileID)
	messageID, err := primitive.ObjectIDFromHex(Id)
	if err != nil {
		return nil, err
	}

	// find chat filter
	filter := bson.D{{"$and", []interface{}{
		bson.D{{"_id", messageID}},
		bson.D{{"sender", user}},
		bson.D{{"isActive", true}},
	}}}

	var result model.Chat

	err = messageCollection.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		return nil, errors.Wrap(err, "User don't have permission to delete the chat")
	}

	// check if user has valid permissions on the board
	userKey := fmt.Sprintf("boards:%s", user)
	boardPermission := permissions.GetBoardPermissions(userKey, cache)
	role := boardPermission[result.BoardID.Hex()]
	if role == "" {
		return nil, errors.New("User do not have access to the board")
	}

	// find chat filter
	filter = bson.D{{"$and", []interface{}{
		bson.D{{"_id", messageID}},
		bson.D{{"sender", user}},
		bson.D{{"isActive", true}},
	}}}

	dt := time.Now()
	update := bson.D{{"$set", bson.D{{"isActive", false}, {"deleteDate", dt}}}} // soft delete

	_, err = messageCollection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to delete message")
	}

	res := make(map[string]interface{})
	resData := make(map[string]interface{})
	resData["error"] = false
	resData["msg"] = "Chat deleted Successfully"
	res["data"] = resData
	return res, nil
}
