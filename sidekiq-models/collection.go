package model

import (
	"encoding/json"
	"time"

	peoplerpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Collection struct {
	// common fields
	Id                      primitive.ObjectID               `json:"_id" bson:"_id"`
	PostID                  primitive.ObjectID               `json:"postID" bson:"postID"`
	Title                   string                           `json:"title" bson:"title"`
	Type                    string                           `json:"type" bson:"type"`
	Description             string                           `json:"description" bson:"description"`
	Tags                    []string                         `json:"tags" bson:"tags"`
	Owner                   string                           `json:"owner" bson:"owner"`
	State                   string                           `json:"state" bson:"state"`
	Priority                string                           `json:"priority" bson:"priority"`
	PostStartDate           string                           `json:"postStartDate" bson:"postStartDate"`
	ProprietaryRegisterDate string                           `json:"proprietaryRegisterDate" bson:"proprietaryRegisterDate"`
	ViewCount               int                              `json:"viewCount" bson:"viewCount"`
	CreateDate              time.Time                        `json:"createDate" bson:"createDate"`
	ModifiedDate            time.Time                        `json:"modifiedDate" bson:"modifiedDate"`
	SortDate                string                           `json:"sortDate" bson:"sortDate"`
	DeletedDate             time.Time                        `json:"deleteDate" bson:"deleteDate"`
	Sort                    int                              `json:"sort" bson:"sort"`
	Story                   Story                            `json:"story" bson:"story"`
	PPV                     float64                          `json:"ppv" bson:"ppv"`
	Visible                 string                           `json:"visible" bson:"visible"`
	Shareable               bool                             `json:"shareable" bson:"shareable"`
	Searchable              string                           `json:"searchable" bson:"searchable"`
	Saving                  int                              `json:"saving" bson:"saving"`
	Reactions               bool                             `json:"reactions" bson:"reactions"`
	Comments                []Comment                        `json:"comments" bson:"comments"`
	OwnerInfo               *peoplerpc.ConciseProfileReply   `json:"ownerInfo" bson:"ownerInfo"`
	PostEndDate             string                           `json:"postEndDate" bson:"postEndDate"`
	PostScheduleDate        string                           `json:"postScheduleDate" bson:"postScheduleDate"`
	PostScheduleTime        string                           `json:"postScheduleTime" bson:"postScheduleTime"`
	TaggedPeople            []*peoplerpc.ConciseProfileReply `json:"taggedPeople" bson:"taggedPeople"`
	LockedDate              string                           `json:"lockedDate" bson:"lockedDate"`
	Location                string                           `json:"location"`
	Likes                   []string                         `json:"likes" bson:"likes"`
	TotalComments           int                              `json:"totalComments" bson:"totalComments"`
	TotalLikes              int                              `json:"totalLikes" bson:"totalLikes"`
	IsLiked                 bool                             `json:"isLiked" bson:"isLiked"`
	Pos                     int                              `json:"pos" bson:"pos"`
	CoverImage              string                           `json:"coverImage" bson:"coverImage"`
	FileProcStatus          string                           `json:"fileProcStatus" bson:"fileProcStatus"`
	EditBy                  string                           `json:"editBy" bson:"editBy"`
	EditDate                *time.Time                       `json:"editDate" bson:"editDate"`
	// things specific fields
	Things []Things `json:"things" db:"things"`
}
type UpdateCollection struct {
	Title string   `json:"title" bson:"title"`
	Tags  []string `json:"tags" bson:"tags"`
}
type Things struct {
	ThingID primitive.ObjectID `json:"thingID" bson:"thingID"`
	Type    string             `json:"type" db:"type"`
	URL     string             `json:"url" db:"url"`
}

func (b Collection) ToMap() (dat map[string]interface{}) {
	d, _ := json.Marshal(b)
	json.Unmarshal(d, &dat)
	return
}
