package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ConnectionRequest struct {
	ID         primitive.ObjectID `json:"_id" bson:"_id"`
	ProfileID  string             `json:"profileID" bson:"profileID"`
	AssigneeID string             `json:"assigneeID,omitempty" bson:"assigneeID,omitempty"`
	// GeneratorID string             `json:"generatorID,omitempty" bson:"generatorID,omitempty"`
	Code       string    `json:"code" bson:"code"`
	Duration   string    `json:"duration" bson:"duration" `
	QR         string    `json:"qr" bson:"qr"`
	Email      string    `json:"email" bson:"email"`
	ExpiryDate time.Time `json:"expiryDate" bson:"expiryDate"`
	CreateDate time.Time `json:"createDate" bson:"createDate"`
}

type Connection struct {
	ID                  primitive.ObjectID `json:"_id" bson:"_id"`
	ProfileID           string             `json:"profileID" bson:"profileID"`
	ConnectionProfileID string             `json:"connectionID" bson:"connectionID"`
	Tags                []string           `json:"tags" bson:"tags"`
	Photo               string             `json:"photo" bson:"photo,omitempty"`
	Relationship        string             `json:"relationship" bson:"relationship"`
	Abv                 string             `json:"abv" bson:"abv,omitempty"`
	FirstName           string             `json:"firstName" bson:"firstName"`
	LastName            string             `json:"lastName" bson:"lastName"`
	NickName            string             `json:"nickName" bson:"nickName,omitempty"`
	ScreenName          string             `json:"screenName" bson:"screenName,omitempty"`
	Bio                 string             `json:"bio" bson:"bio,omitempty"`
	Phone1              string             `json:"phone1" bson:"phone1,omitempty"`
	Phone2              string             `json:"phone2" bson:"phone2,omitempty"`
	Email1              string             `json:"email1" bson:"email1,omitempty"`
	Email2              string             `json:"email2" bson:"email2,omitempty"`
	Address1            string             `json:"address1" bson:"address1,omitempty"`
	Address2            string             `json:"address2" bson:"address2,omitempty"`
	City                string             `json:"city" bson:"city,omitempty"`
	State               string             `json:"state" bson:"state,omitempty"`
	Zip                 string             `json:"zip" bson:"zip,omitempty"`
	Country             string             `json:"country" bson:"country,omitempty"`
	Birthday            string             `json:"birthday" bson:"birthday,omitempty"`
	Gender              int                `json:"gender" bson:"gender,omitempty"`
	Notes               string             `json:"notes" bson:"notes"`
	LinkedToIDs         []int              `json:"linkedToIDs" bson:"linkedToIDs"`
	MetOnDate           time.Time          `json:"metOnDate" bson:"metOnDate"`
	MetAtLocation       []float64          `json:"metAtLocation" bson:"metAtLocation,omitempty"`
	MetNote             string             `json:"metNote" bson:"metNote,omitempty"`
	AutoAddMeToBoard    bool               `json:"autoAddMeToBoard" bson:"autoAddMeToBoard"`
	CreateDate          time.Time          `json:"createDate" bson:"createDate"`
	IsActive            bool               `json:"isActive" bson:"isActive"`
	IsBlocked           bool               `json:"isBlocked" bson:"isBlocked"`
	IsArchived          bool               `json:"isArchived" bson:"isArchived"`
	Thumbs              Thumbnails         `json:"thumbs"`
}
