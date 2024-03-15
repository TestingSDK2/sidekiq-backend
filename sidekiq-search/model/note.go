package model

import (
	"encoding/json"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Note struct {
	// common fields
	Id                      primitive.ObjectID `json:"_id" bson:"_id"`
	PostID                  primitive.ObjectID `json:"postID" bson:"postID"`
	BoardID                 primitive.ObjectID `json:"boardID" bson:"boardID"`
	CollectionID            primitive.ObjectID `json:"collectionID" bson:"collectionID"`
	Title                   string             `json:"title" bson:"title"`
	Type                    string             `json:"type" bson:"type"`
	Description             string             `json:"description" bson:"description"`
	Tags                    []string           `json:"tags" bson:"tags"`
	Owner                   string             `json:"owner" bson:"owner"`
	State                   string             `json:"state" bson:"state"`
	Priority                string             `json:"priority" bson:"priority"`
	PostStartDate           string             `json:"postStartDate" bson:"postStartDate"`
	ProprietaryRegisterDate string             `json:"proprietaryRegisterDate" bson:"proprietaryRegisterDate"`
	ViewCount               int                `json:"viewCount" bson:"viewCount"`
	CreateDate              time.Time          `json:"createDate" bson:"createDate"`
	ModifiedDate            time.Time          `json:"modifiedDate" bson:"modifiedDate"`
	SortDate                string             `json:"sortDate" bson:"sortDate"`
	DeletedDate             time.Time          `json:"deleteDate" bson:"deleteDate"`
	ExpiryDate              time.Time          `json:"expiryDate" bson:"expiryDate"`
	Sort                    int                `json:"sort" bson:"sort"`
	Story                   Story              `json:"story" bson:"story"`
	PPV                     float64            `json:"ppv" bson:"ppv"`
	Visible                 string             `json:"visible" bson:"visible"`
	Shareable               bool               `json:"shareable" bson:"shareable"`
	Searchable              string             `json:"searchable" bson:"searchable"`
	Saving                  int                `json:"saving" bson:"saving"`
	Reactions               bool               `json:"reactions" bson:"reactions"`
	TotalComments           int                `json:"totalComments" bson:"totalComments"`
	TotalLikes              int                `json:"totalLikes" db:"totalLikes"`
	IsLiked                 bool               `json:"isLiked" db:"isLiked"`
	Comments                []Comment          `json:"comments" bson:"comments"`
	Likes                   []string           `json:"likes" db:"likes"`
	OwnerInfo               *ConciseProfile    `json:"ownerInfo" bson:"ownerInfo"`
	PostEndDate             string             `json:"postEndDate" bson:"postEndDate"`
	PostScheduleDate        string             `json:"postScheduleDate" bson:"postScheduleDate"`
	PostScheduleTime        string             `json:"postScheduleTime" bson:"postScheduleTime"`
	TaggedPeople            []ConciseProfile   `json:"taggedPeople" bson:"taggedPeople"`
	LockedDate              string             `json:"lockedDate" bson:"lockedDate"`
	Location                string             `json:"location" bson:"location"`
	EditBy                  string             `json:"editBy" bson:"editBy"`
	EditDate                *time.Time         `json:"editDate" bson:"editDate"`

	// note specific fields
	Body     string `json:"body" bson:"body"` // stores HTML format of the rich text
	Body_raw string `json:"body_raw" bson:"body_raw"`
}

func (b Note) ToMap() (dat map[string]interface{}) {
	d, _ := json.Marshal(b)
	json.Unmarshal(d, &dat)
	return
}
