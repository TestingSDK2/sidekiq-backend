package model

import (
	"encoding/json"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ThingActivity struct {
	Id        primitive.ObjectID `json:"_id" bson:"_id"`
	BoardID   primitive.ObjectID `json:"boardID" bson:"boardID"`
	PostID    primitive.ObjectID `json:"postID" bson:"postID"`
	ThingID   primitive.ObjectID `json:"thingID" bson:"thingID"`
	ThingType string             `json:"thingType" bson:"thingType"`
	ProfileID int                `json:"profileID" bson:"profileID"`
	Message   string             `json:"message" bson:"message"`
	Image     string             `json:"image"`
	// Name             string             `json:"name" bson:"name"`
	LastModifiedDate time.Time `json:"lastModifiedDate" bson:"lastModifiedDate"`
	DateModified     string    `json:"dateModified" bson:"dateModified"`
}

func (b ThingActivity) ToMap() (dat map[string]interface{}) {
	d, _ := json.Marshal(b)
	json.Unmarshal(d, &dat)
	return
}

func (b *ThingActivity) Create(boardID, postID primitive.ObjectID, profileID int, thingType, msg string) {
	b.Id = primitive.NewObjectID()
	b.BoardID = boardID
	b.ThingType = thingType
	b.ProfileID = profileID
	b.Message = msg
	b.LastModifiedDate = time.Now()
	b.DateModified = b.LastModifiedDate.Format("01-02-2006 15:04:05")
}
