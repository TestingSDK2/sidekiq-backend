package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Recent struct {
	ThingID             primitive.ObjectID `json:"thingID" bson:"thingID"`
	ProfileID           string             `json:"profileID" bson:"profileID"`
	BoardID             primitive.ObjectID `json:"boardID" bson:"boardID"`
	DisplayTitle        string             `json:"displayTitle" bson:"displayTitle"`
	ThingType           string             `json:"thingType" bson:"thingType"`
	LastViewedDate      time.Time          `json:"lastViewedDate" bson:"lastViewedDate"`
	LastAddedDate       time.Time          `json:"lastAddedDate" bson:"lastAddedDate"`
	ExpectedExpiredDate time.Time          `json:"expectedExpiredDate" bson:"expectedExpiredDate"`
	// Thing               map[string]interface{} `json:"thing" bson:"thing"`
	OwnerInfo ConciseProfile `json:"ownerInfo"  bson:"-"`
}

type RecentDeletePayload struct {
	RecentIds []string `json:"recentIds"`
}
