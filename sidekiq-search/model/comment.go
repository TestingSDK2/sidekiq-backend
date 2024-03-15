package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Comment struct {
	ID               primitive.ObjectID `json:"_id" bson:"_id"`
	ProfileID        int                `json:"-" bson:"profileID"`
	Message          string             `json:"message,omitempty" bson:"message"`
	CreateDate       time.Time          `json:"createDate" bson:"createDate"`
	LastModifiedDate time.Time          `json:"lastModifiedDate" bson:"lastModifiedDate"`
	AddedTime        string             `json:"-"`
	EditTime         string             `json:"-"`
}

type ReactionList struct {
	ConciseProfile *ConciseProfile `json:"profileInfo"`
	Comment        *Comment        `json:"commentInfo,omitempty" bson:"comments"`
	Likes          string          `json:"-" bson:"likes"`
}
