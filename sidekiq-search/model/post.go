package model

import (
	"encoding/json"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Post struct {
	Id               primitive.ObjectID `json:"_id" bson:"_id"`
	BoardID          primitive.ObjectID `json:"boardID" bson:"boardID"`
	ThingOptSettings bool               `json:"thingOptSettings" bson:"thingOptSettings"`
	Tags             []string           `json:"tags" bson:"tags"`
	Title            string             `json:"title" bson:"title"`
	Description      string             `json:"description" bson:"description"`
	State            string             `json:"state" bson:"state"`
	Owner            string             `json:"owner" bson:"owner"`
	Priority         string             `json:"priority" bson:"priority"`
	PostStartDate    string             `json:"postStartDate" bson:"postStartDate"`
	PostEndDate      string             `json:"postEndDate" bson:"postEndDate"`
	LockedDate       string             `json:"lockedDate" bson:"lockedDate"`
	CreateDate       time.Time          `json:"createDate" bson:"createDate"`
	ModifiedDate     time.Time          `json:"modifiedDate" bson:"modifiedDate"`
	SortDate         string             `json:"sortDate" bson:"sortDate"`
	DeletedDate      time.Time          `json:"deleteDate" bson:"deleteDate"`
	Sequence         int                `json:"sequence" bson:"sequence"`
	PublicStartDate  string             `json:"publicStartDate" bson:"publicStartDate"`
	PublicEndDate    string             `json:"publicEndDate" bson:"publicEndDate"`
	Shareable        bool               `json:"shareable" bson:"shareable"`
	Searchable       string             `json:"searchable" bson:"searchable"`
	Bookmark         bool               `json:"bookmark" bson:"bookmark"`
	Visible          string             `json:"visible" bson:"visible"`
	Reactions        bool               `json:"reactions" bson:"reactions"`
	Comments         []Comment          `json:"comments" bson:"comments"`
	Likes            []string           `json:"likes" bson:"likes"`
	ViewCount        int                `json:"viewCount" bson:"viewCount"`
	Type             string             `json:"type" bson:"type"`
	TotalComments    int                `json:"totalComments" bson:"totalComments"`
	TotalLikes       int                `json:"totalLikes" bson:"totalLikes"`
	IsLiked          bool               `json:"isLiked" bson:"isLiked"`
	IsReactions      bool               `json:"isReactions" bson:"isReactions"`
	OwnerInfo        *ConciseProfile    `json:"ownerInfo" bson:"ownerInfo"`
	TaggedPeople     []ConciseProfile   `json:"taggedPeople" bson:"taggedPeople"`
	Location         string             `json:"location" bson:"location"`
	Things           interface{}        `json:"things"`
	IsBookmarked     bool               `json:"isBookmarked" bson:"isBookmarked"`
	Hidden           bool               `json:"hidden" bson:"hidden"`
	BookmarkID       string             `json:"bookmarkID" bson:"bookmarkID"`
	IsCoverImage     bool               `json:"isCoverImage" bson:"isCoverImage"`
	CoverImageUrl    string             `json:"coverImageUrl" bson:"-"`
	Thumbs           Thumbnails         `json:"thumbs" bson:"-"`
	FileExt          string             `json:"fileExt" bson:"fileExt"`
}

func (b Post) ToMap() (dat map[string]interface{}) {
	d, _ := json.Marshal(b)
	json.Unmarshal(d, &dat)
	return
}

type PostThingsPayload struct {
	Notes       []interface{} `json:"notes"`
	Tasks       []interface{} `json:"tasks"`
	Collections []Collection  `json:"collections"`
}

type PostThingUpdate struct {
	UpdateThing []map[string]interface{} `json:"updateThing"`
}

type PostThingDelete struct {
	DeleteThing []map[string]interface{} `json:"deleteThing"`
}

type PostThingEvent struct {
	ThingID   string     `json:"thingID"`
	ThingType string     `json:"thingType"`
	EditBy    string     `json:"editBy"`
	EditDate  *time.Time `json:"editDate"`
}

type PostThingEventMember struct {
	EventType  string          `json:"eventType"`
	EditByInfo *ConciseProfile `json:"editByInfo"`
	ThingType  string          `json:"thingType"`
	ThingID    string          `json:"thingID"`
	BoardID    string          `json:"boardID"`
	PostID     string          `json:"postID"`
	EditDate   *time.Time      `json:"editDate"`
}
