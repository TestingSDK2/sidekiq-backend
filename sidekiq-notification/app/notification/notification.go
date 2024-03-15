package notification

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ProImaging/sidekiq-backend/sidekiq-models/model"
	"github.com/ProImaging/sidekiq-backend/sidekiq-notification/consts"
	"github.com/ProImaging/sidekiq-backend/sidekiq-notification/database"
	"github.com/ProImaging/sidekiq-backend/sidekiq-notification/mongodatabase"
	"github.com/ProImaging/sidekiq-backend/sidekiq-notification/util"
	accountProtobuf "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"
	"github.com/sirupsen/logrus"

	realtimeProtobuf "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-realtime/v1"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func createNotification(db *mongodatabase.DBConfig, mysql *database.Database, notification *model.Notification) error {
	dbConn, err := db.New(consts.Notification)
	if err != nil {
		return err
	}

	notificationCollection, notificationClient := dbConn.Collection, dbConn.Client
	defer notificationClient.Disconnect(context.TODO())

	_, err = notificationCollection.InsertOne(context.TODO(), notification)
	if err != nil {
		return err
	}

	profileID, err := strconv.Atoi(notification.RecipientProfileID)
	if err != nil {
		return err
	}

	stmt := "UPDATE `sidekiq-dev`.AccountProfile SET unread_notification_count = unread_notification_count + 1 WHERE id = ?"
	_, err = mysql.Conn.Exec(stmt, profileID)
	if err != nil {
		return errors.Wrap(err, "unable to updated unread_notification_count")
	}

	return nil
}

// Function to update the isRead field of a notification based on its ID.
func markNotificationAsRead(db *mongodatabase.DBConfig, mysql *database.Database, notificationID string, profileID string) error {
	dbConn, err := db.New(consts.Notification)
	if err != nil {
		return err
	}

	notificationCollection, notificationClient := dbConn.Collection, dbConn.Client
	defer notificationClient.Disconnect(context.TODO())

	objectID, err := primitive.ObjectIDFromHex(notificationID)
	if err != nil {
		fmt.Println("Error:", err)
		return err
	}

	// Define a filter to find the notification by ID.
	filter := bson.M{"_id": objectID, "recipientProfileId": profileID}

	// Define an update to set the isRead field to true.
	update := bson.M{
		"$set": bson.M{"isRead": true},
	}

	result, err := notificationCollection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		return err
	}

	if result.ModifiedCount > 0 {
		stmt := "UPDATE `sidekiq-dev`.AccountProfile SET unread_notification_count = unread_notification_count - 1 WHERE id = ?"
		_, err = mysql.Conn.Exec(stmt, profileID)
		if err != nil {
			return errors.Wrap(err, "unable to updated unread_notification_count")
		}
	}

	return nil
}

func markAllNotificationAsRead(db *mongodatabase.DBConfig, mysql *database.Database, reqprofileID string) error {
	dbConn, err := db.New(consts.Notification)
	if err != nil {
		return err
	}

	notificationCollection, notificationClient := dbConn.Collection, dbConn.Client
	defer notificationClient.Disconnect(context.TODO())

	filter := bson.M{"recipientProfileId": reqprofileID}
	update := bson.M{
		"$set": bson.M{"isRead": true},
	}
	_, err = notificationCollection.UpdateMany(context.TODO(), filter, update)
	if err != nil {
		return err
	}

	profileID, err := strconv.Atoi(reqprofileID)
	if err != nil {
		return err
	}

	stmt := "UPDATE `sidekiq-dev`.AccountProfile SET unread_notification_count = 0 WHERE id = ?"
	_, err = mysql.Conn.Exec(stmt, profileID)
	if err != nil {
		return errors.Wrap(err, "unable to updated unread_notification_count")
	}
	return nil
}

func getNotificationList(db *mongodatabase.DBConfig, mysql *database.Database, recipientProfileID string) ([]model.Notification, error) {
	dbConn, err := db.New(consts.Notification)
	if err != nil {
		return []model.Notification{}, err
	}

	var notifications []model.Notification

	notificationCollection, notificationClient := dbConn.Collection, dbConn.Client
	defer notificationClient.Disconnect(context.TODO())

	findOptions := options.Find()
	findOptions.SetSort(bson.M{"createdDate": -1})
	cur, err := notificationCollection.Find(context.TODO(), bson.M{"recipientProfileId": recipientProfileID}, findOptions)
	if err != nil {
		return []model.Notification{}, err
	}

	err = cur.All(context.TODO(), &notifications)
	if err != nil {
		return []model.Notification{}, err
	}

	return notifications, nil
}

func getNotificationDisplayCount(db *mongodatabase.DBConfig, mysql *database.Database, recipientProfileID string) (int64, error) {
	profileID, err := strconv.Atoi(recipientProfileID)
	if err != nil {
		return 0, err
	}

	stmt := "SELECT unread_notification_count FROM `sidekiq-dev`.AccountProfile WHERE id = ?;"
	var count int64

	err = mysql.Conn.Get(&count, stmt, profileID)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func notificationHandler(db *mongodatabase.DBConfig, mysql *database.Database, receiverIDs []int32, senderId int, accountServiceClient accountProtobuf.AccountServiceClient, realtimeServiceClient realtimeProtobuf.DeliveryServiceClient, thingType, thingID, actionType, message string) error {
	defer util.Recover()

	for _, receiverID := range receiverIDs {

		if int(receiverID) == senderId {
			return nil
		}

		requestSender := &accountProtobuf.ConciseProfileRequest{
			ProfileId: int32(senderId),
		}

		senderInfo, err := accountServiceClient.GetConciseProfile(context.TODO(), requestSender)
		if err != nil {
			return errors.Wrap(err, "unable to get profile details")
		}

		requestReceiver := &accountProtobuf.ConciseProfileRequest{
			ProfileId: int32(receiverID),
		}

		receiverInfo, err := accountServiceClient.GetConciseProfile(context.TODO(), requestReceiver)
		if err != nil {
			return errors.Wrap(err, "unable to get profile details")
		}

		if senderInfo != nil {
			switch actionType {
			case consts.BoardFollowed:
				message = fmt.Sprintf("Your board has been followed by %s %s", senderInfo.FirstName, senderInfo.LastName)
			case consts.DeleteConnection:
				message = fmt.Sprintf("%s %s has removed you from their connection.", senderInfo.FirstName, senderInfo.LastName)
			case consts.AcceptConnectionRequest:
				message = fmt.Sprintf("%s %s has accepted your request", senderInfo.FirstName, senderInfo.LastName)
			case consts.TaskInitiated:
				message = fmt.Sprintf("Task has been assigned by %s %s", senderInfo.FirstName, senderInfo.LastName)
			case consts.TaskUpdated:
				message = fmt.Sprintf(message, senderInfo.FirstName, senderInfo.LastName)
			case consts.AddComment:
				message = fmt.Sprintf("Comment added to your %s by %s %s", strings.ToLower(thingType), senderInfo.FirstName, senderInfo.LastName)
			}

			if message == "" {
				return nil
			}

			objnotification := model.Notification{
				ID:                 primitive.NewObjectID(),
				RecipientProfileID: fmt.Sprint(receiverID),
				SenderProfileID:    fmt.Sprint(senderId),
				ThingType:          strings.ToUpper(thingType),
				ThingID:            thingID,
				ActionType:         actionType,
				NotificationText:   message,
				IsRead:             false,
				CreatedDate:        time.Now(),
			}

			err = createNotification(db, mysql, &objnotification)
			if err != nil {
				return errors.Wrap(err, "error while creating notification object")
			}

			request := realtimeProtobuf.NotificationRequest{
				NotificationId:     objnotification.ID.Hex(),
				RecipientMemberId:  fmt.Sprint(receiverInfo.AccountID),
				RecipientProfileId: objnotification.RecipientProfileID,
				SenderProfileId:    objnotification.SenderProfileID,
				ThingType:          objnotification.ThingType,
				ThingId:            objnotification.ThingID,
				IsRead:             objnotification.IsRead,
				ActionType:         objnotification.ActionType,
				NotificationText:   objnotification.NotificationText,
			}

			resp, err := realtimeServiceClient.DeliverNotification(context.TODO(), &request)
			if err != nil {
				return errors.Wrap(err, "error while delivering notification from real time service.")
			}

			logrus.Info("Notification delivery status:", resp.Acknowledgment)
		}
	}
	return nil
}
