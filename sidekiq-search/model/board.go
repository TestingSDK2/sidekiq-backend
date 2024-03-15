package model

import (
	"encoding/json"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Board model
type Board struct {
	Id               primitive.ObjectID `json:"_id" bson:"_id"`
	Owner            string             `json:"owner" bson:"owner"`
	State            string             `json:"state" bson:"state"`
	Visible          string             `json:"visible" bson:"visible"`
	Shareable        bool               `json:"shareable" bson:"shareable"`
	Searchable       string             `json:"searchable" bson:"searchable"`
	Bookmark         bool               `json:"bookmark" bson:"bookmark"`
	BookmarkID       string             `json:"bookmarkID" bson:"bookmarkID"`
	IsBookmarked     bool               `json:"isBookmarked" bson:"isBookmarked"`
	Reactions        bool               `json:"reactions" bson:"reactions"`
	ViewCount        int                `json:"viewCount" bson:"viewCount"`
	OwnerInfo        ConciseProfile     `json:"ownerInfo" bson:"ownerInfo"`
	TaggedPeople     []ConciseProfile   `json:"taggedPeople" bson:"taggedPeople"`
	PostScheduleDate string             `json:"postScheduleDate" bson:"postScheduleDate"`
	PostScheduleTime string             `json:"postScheduleTime" bson:"postScheduleTime"`
	ParentID         string             `json:"parentID" bson:"parentID"`
	CoverImage       string             `json:"coverImage" bson:"coverImage"`
	Password         string             `json:"password,omitempty" bson:"password"`
	Chat             bool               `json:"chat" bson:"chat"`
	IsComments       bool               `json:"isComments" bson:"isComments"`
	Title            string             `json:"title" bson:"title"`
	Type             string             `json:"type" bson:"type"`
	Description      string             `json:"description" bson:"description"`
	Tags             []string           `json:"tags" bson:"tags"`
	AllThingsTags    []string           `json:"allThingsTags" bson:"allThingsTags"`
	Admins           []string           `json:"admins" bson:"admins"`
	Authors          []string           `json:"authors" bson:"authors"`
	Viewers          []string           `json:"viewers" bson:"viewers"`
	Subscribers      []string           `json:"subscribers" bson:"subscribers"`
	Followers        []string           `json:"followers" bson:"followers"`
	Guests           []string           `json:"guests" bson:"guests"`
	Blocked          []string           `json:"blocked" bson:"blocked"`
	Sequence         int                `json:"sequence" bson:"sequence"`
	Comments         []Comment          `json:"comments" bson:"comments"`
	CreateDate       time.Time          `json:"createDate" bson:"createDate"`
	ModifiedDate     time.Time          `json:"modifiedDate" bson:"modifiedDate"`
	DeleteDate       time.Time          `json:"deleteDatw" bson:"deleteDate"`
	ExpiryDate       time.Time          `json:"expiryDate" bson:"expiryDate"`
	PublicStartDate  string             `json:"publicStartDate" bson:"publicStartDate"`
	PublicEndDate    string             `json:"publicEndDate" bson:"publicEndDate"`
	SortDate         string             `json:"sortDate" bson:"sortDate"`
	IsPassword       bool               `json:"isPassword" bson:"isPassword"`
	Location         string             `json:"location" bson:"location"`
	BoardFollowers   []ConciseProfile   `json:"boardFollowers"`
	BoardMembers     []ConciseProfile   `json:"boardMembers"`
	TotalFollowers   int                `json:"totalFollowers"`
	TotalTags        int                `json:"totalTags"`
	TotalMembers     int                `json:"totalMembers"`
	IsDefaultBoard   bool               `json:"isDefaultBoard" bson:"isDefaultBoard"`
	IsBoardFollower  bool               `json:"isBoardFollower"`
	Hidden           bool               `json:"hidden" bson:"hidden"`
	Role             string             `json:"role"`
	Likes            []string           `json:"likes" bson:"likes"`
	TotalComments    int                `json:"totalComments" bson:"-"`
	TotalLikes       int                `json:"totalLikes" bson:"-"`
	IsLiked          bool               `json:"isLiked" bson:"isLiked"`
	IsHidden         bool               `json:"isHidden" bson:"isHidden"`
	IsBoardShared    bool               `json:"isBoardShared" bson:"-"`
	Thumbnails       Thumbnails         `json:"thumbs" bson:"-"`
}

type BoardSearch struct {
	Id         primitive.ObjectID `json:"_id" bson:"_id"`
	Title      string             `json:"title" bson:"title"`
	CreateDate time.Time          `json:"createDate" bson:"createDate"`
	Tags       []string           `json:"tags" bson:"tags"`
}
type UpdatedBoard struct {
	Board          Board    `json:"board" bson:"Board"`
	RemovedMembers []string `json:"removedMembers" bson:"RemovedMembers"`
}

// BoardPermission; ["6b2342423482349ddg232"]:"owner"
type BoardPermission map[string]string

func (b *BoardPermission) ToJSON() string {
	json, _ := json.Marshal(b)
	return string(json)
}

type BoardMapping struct {
	BoardId  primitive.ObjectID `json:"boardID" bson:"boardID"`
	ParentID primitive.ObjectID `json:"parentID" bson:"parentID"`
}

type BoardFollowInfo struct {
	BoardTitle string `json:"boardTitle" db:"boardTitle"`
	BoardID    string `json:"boardID" db:"boardID"`
	OwnerID    int    `json:"ownerID" db:"ownerID"`
	CreateDate string `json:"createDate" db:"createDate"`
	ProfileID  int    `json:"profileID" db:"profileID"`
}
type BoardFilter struct {
	Page       int
	Limit      int
	Tags       []string
	FileExt    string
	Type       string
	Owner      string
	UploadDate string
	Location   string
	BoardID    primitive.ObjectID
}

type SearchFilter struct {
	FileType   string
	People     string
	UploadDate string
	Location   string
}

type BoardInvite struct {
	ID        int       `json:"id" db:"id"`
	SenderID  string    `json:"senderID" db:"senderID"`
	InviteeID string    `json:"inviteeID" db:"inviteeID"`
	BoardID   string    `json:"boardID" db:"boardID"`
	Role      string    `json:"role" db:"role"`
	CreatedAt time.Time `json:"createDate" db:"createDate"`
}

type ListInvitations struct {
	Name       string    `json:"name" db:"name"`
	FirstName  string    `json:"firstName" db:"firstName"`
	LastName   string    `json:"lastName" db:"lastName"`
	Photo      string    `json:"photo" db:"photo"`
	UserID     string    `json:"-" db:"accountID"`
	BoardID    string    `json:"boardID" db:"boardID"`
	BoardTitle string    `json:"boardTitle"`
	Role       string    `json:"role" db:"role"`
	CreatedAt  time.Time `json:"createDate" db:"createDate"`
}

type HandleBoardInvitation struct {
	BoardID string `json:"boardID"`
	Type    string `json:"type"`
}

type BoardMemberRole struct {
	AccountID  int        `db:"accountID"`
	ProfileID  string     `json:"connectionProfileID" db:"id" bson:"connectionID"`
	FirstName  string     `json:"firstName" db:"firstName" bson:"firstName"`
	LastName   string     `json:"lastName" db:"lastName" bson:"lastName"`
	ScreenName string     `json:"screenName" db:"screenName" bson:"screenName"`
	Role       string     `json:"role"`
	Photo      string     `json:"photo" db:"photo" bson:"photo"`
	Thumbs     Thumbnails `json:"thumbs"`
}

type BoardMemberRoleRequest struct {
	ProfileID string `json:"connectionProfileID"`
	Role      string `json:"role"`
}

type BoardMemberRequest struct {
	Data []BoardMemberRoleRequest `json:"data"`
}

type ChangeProfileRole struct {
	ProfileID string `json:"profileID"`
	OldRole   string `json:"oldRole"`
	NewRole   string `json:"newRole"`
}

func (b Board) ToMap() (dat map[string]interface{}) {
	d, _ := json.Marshal(b)
	json.Unmarshal(d, &dat)
	return
}

type BoardThingTags struct {
	ID      primitive.ObjectID  `bson:"_id,omitempty"`
	BoardID primitive.ObjectID  `bson:"boardID,omitempty"`
	Tags    map[string][]string `bson:"tags,omitempty"`
}
