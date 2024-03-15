package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Notification struct {
	ID               primitive.ObjectID `bson:"_id" json:"_id"`
	SenderProfileID  string             `bson:"senderProfileId,omitempty" json:"senderProfileId,omitempty"`
	ThingType        string             `bson:"thingType" json:"thingType"`
	ThingID          string             `bson:"thingId" json:"thingId"`
	IsRead           bool               `bson:"isRead" json:"isRead"`
	ActionType       string             `bson:"actionType" json:"actionType"`
	NotificationText string             `bson:"notificationText" json:"notificationText"`
	CreatedDate      time.Time          `bson:"createdDate" json:"createdDate"`
	SenderDetails    *ConciseProfile    `json:"senderDetails"`
}

type ProfileNotification struct {
	ID                       primitive.ObjectID `bson:"_id" json:"id"`
	DisplayNotificationCount int64              `bson:"displayNotificationCount" json:"displayNotificationCount"`
	RecipientProfileID       string             `bson:"recipientProfileId" json:"recipientProfileId"`
	Notifications            []Notification     `bson:"notifications" json:"notifications"`
}
