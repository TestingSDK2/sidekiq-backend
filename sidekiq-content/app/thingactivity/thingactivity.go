package thingactivity

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/consts"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/mongodatabase"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/thingsqs"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/util"
	model "github.com/TestingSDK2/sidekiq-backend/sidekiq-models"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func pushThingActivityToSQS(thingSQS *thingsqs.SQSConn, msg map[string]interface{}) error {
	jsonStr, err := json.Marshal(msg)
	if err != nil {
		return errors.Wrap(err, "unable to marshal msg")
	}

	queueURL, err := thingSQS.SQS.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: aws.String("demoQueue.fifo"),
	})
	if err != nil {
		return errors.Wrap(err, "unable to get queue url")
	}

	sqsMsg := &sqs.SendMessageInput{
		MessageGroupId: aws.String(uuid.NewString()),
		MessageBody:    aws.String(string(jsonStr)),
		QueueUrl:       queueURL.QueueUrl,
	}

	_, err = thingSQS.SQS.SendMessage(sqsMsg)
	if err != nil {
		return errors.Wrap(err, "unable to send message to queue")
	}

	return nil
}

func listAllThingActivities(mongo *mongodatabase.DBConfig, thingID, limit, page string) (map[string]interface{}, error) {
	dbconn, err := mongo.New(consts.Activity)
	if err != nil {
		return nil, errors.Wrap(err, "unable to connect to "+consts.BoardThingsTags)
	}
	activityColl, bttClient := dbconn.Collection, dbconn.Client
	defer bttClient.Disconnect(context.TODO())

	thingObjID, err := primitive.ObjectIDFromHex(thingID)
	if err != nil {
		return nil, errors.Wrap(err, "unable convert string to ObjectID")
	}

	// check if the thingID is board or not
	dbconn2, err := mongo.New(consts.Board)
	if err != nil {
		return nil, errors.Wrap(err, "unable to connect to "+consts.BoardThingsTags)
	}
	boardColl, boardClient := dbconn2.Collection, dbconn2.Client
	defer boardClient.Disconnect(context.TODO())

	var isBoard bool

	count, err := boardColl.CountDocuments(context.TODO(), bson.M{"_id": thingObjID})
	if err != nil {
		return nil, errors.Wrap(err, "unable to find board count")
	}
	if count == 0 { // if not board, means its a thing?
		isBoard = false
	} else {
		isBoard = true
	}

	var filter primitive.M
	if isBoard {
		filter = bson.M{"boardID": thingObjID}
	} else {
		filter = bson.M{"thingID": thingObjID}
	}

	fmt.Println("isBoard: ", isBoard)
	fmt.Println("filter: ", filter)

	// get total count of activities
	count, err = activityColl.CountDocuments(context.TODO(), filter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find count of total activities")
	}

	findOptions := options.Find()
	findOptions.SetSort(bson.M{"dateModified": -1})

	pgInt, _ := strconv.Atoi(page)
	limitInt, _ := strconv.Atoi(limit)
	offset := (pgInt - 1) * limitInt
	findOptions.SetSkip(int64(offset))
	findOptions.SetLimit(int64(limitInt))

	var activities []model.ThingActivity
	cur, err := activityColl.Find(context.TODO(), filter, findOptions)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find thing activities")
	}

	err = cur.All(context.TODO(), &activities)
	if err != nil {
		return nil, errors.Wrap(err, "unable to unpack thing activities")
	}

	if len(activities) == 0 {
		return util.SetPaginationResponse([]string{}, 0, 1, "No activties"), nil
	}
	return util.SetPaginationResponse(activities, int(count), 1, "Activities fetched successfully"), nil
}
