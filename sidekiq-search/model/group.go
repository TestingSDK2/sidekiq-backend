package model

import (
	"encoding/json"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ChatGroup struct {
	Id      primitive.ObjectID `json:"_id" bson:"_id"`
	BoardID primitive.ObjectID `json:"boardID" bson:"boardID"`
	Name    string             `json:"name" bson:"name"` //group name if any
	Type    int                `json:"type" bson:"type"` //chat type: 0 for one to one chat or 1 for group chat
	// Admin            string             `json:"admin" bson:"admin"`
	Members          []string  `json:"members" bson:"members"`
	CreateDate       time.Time `json:"createDate" bson:"createDate"`
	LastModifiedDate time.Time `json:"lastModifiedDate" bson:"lastModifiedDate"`
	DeletedDate      time.Time `json:"deleteDate" bson:"deleteDate"`
	IsActive         bool      `json:"isActive" bson:"isActive"`
}

// ToJSON converts discussion to json string
func (a *ChatGroup) ToJSON() string {
	json, _ := json.Marshal(a)
	return string(json)
}
