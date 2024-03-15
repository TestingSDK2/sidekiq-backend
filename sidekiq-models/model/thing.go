package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ThingReactions struct {
	Reactions  bool      `json:"reactions" bson:"reactions"`
	Comments   []Comment `json:"comments" bson:"comments"`
	Likes      []string  `json:"likes" bson:"likes"`
	Visibility string    `json:"visible" bson:"visible"`
}

type Bookmark struct {
	// db fields
	ID             primitive.ObjectID `json:"bookmarkID,omitempty" bson:"_id"`
	ProfileID      int                `json:"profileID,omitempty" bson:"profileID"`
	OwnerID        int                `json:"ownerID,omitempty" bson:"ownerID"`
	ThingType      string             `json:"thingType" bson:"thingType"`
	ThingID        string             `json:"thingID" bson:"thingID"`
	ThingLocation  string             `json:"thingLocation" bson:"thingLocation"`
	LastViewedDate string             `json:"lastViewedDate" bson:"lastViewedDate"`
	DeletedDate    time.Time          `json:"deleteDate" bson:"deleteDate,omitempty"`
	Flagged        bool               `json:"flagged" bson:"flagged"`
	CreateDate     time.Time          `json:"createDate" bson:"createDate"`

	// only json fields
	BoardID           string      `json:"boardID" bson:"-"`
	PostID            string      `json:"postID" bson:"-"`
	ThingUploadDate   string      `json:"thingUploadDate" bson:"-"`
	ThingTitle        string      `json:"title" bson:"-"`
	Tags              interface{} `json:"tags" bson:"-"`
	NewConciseProfile `json:"ownerInfo" bson:"-"`
	Things            interface{} `json:"things" bson:"-"`
}

type NewConciseProfile struct {
	FirstName  string `json:"firstName,omitempty" db:"firstName"`
	LastName   string `json:"lastName,omitempty" db:"lastName"`
	Photo      string `json:"photo" db:"photo"`
	ScreenName string `json:"screenName,omitempty" db:"screenName"`
	Email      string `json:"email,omitempty" db:"email"`
	Phone      string `json:"phone,omitempty" db:"phone"`
}

type SetBookmark struct {
	ThingID     string `json:"thingID" db:"thingID"`
	DisplayName string `json:"displayName" db:"displayName"`
	ThingType   string `json:"thingType" db:"thingType"`
}

type BookmarkInfo struct {
	DisplayName    string    `json:"displayName" bson:"title"`
	ThingOwner     string    `json:"ownerID" bson:"owner"`
	ThingExtension string    `json:"thingExtension,omitempty" bson:"fileExt"`
	UploadDate     time.Time `json:"uploadDate,omitempty" bson:"createDate"`
}
