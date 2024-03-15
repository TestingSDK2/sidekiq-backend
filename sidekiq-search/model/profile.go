package model

import (
	"time"
)

type ConciseProfile struct {
	Id                 int        `json:"id,omitempty" db:"id"`
	UserID             int        `json:"-" db:"accountID"`
	FirstName          string     `json:"firstName,omitempty" db:"firstName"`
	LastName           string     `json:"lastName,omitempty" db:"lastName"`
	Photo              string     `json:"photo" db:"photo"`
	ScreenName         string     `json:"screenName,omitempty" db:"screenName"`
	UserName           string     `json:"userName,omitempty" db:"userName"`
	Email              string     `json:"email,omitempty" db:"email"`
	Phone              string     `json:"phone,omitempty" db:"phone"`
	Type               string     `json:"type,omitempty"`
	Shareable          bool       `json:"shareable" db:"shareable"`
	DefaultThingsBoard string     `json:"defaultThingsBoard" db:"defaultThingsBoard"`
	Thumbs             Thumbnails `json:"thumbs"`
}

type ExternalProfile struct {
	Id                 int          `json:"id,omitempty" db:"id"`
	AccountID          int          `json:"-" db:"accountID"`
	AccountType        int          `json:"-" db:"accountType"`
	AccountFirstName   string       `json:"-" db:"accountFirstName"` // Take this value from account table.
	AccountLastName    string       `json:"-" db:"accountLastName"`
	FirstName          string       `json:"firstName,omitempty" db:"firstName"` // profile table info
	LastName           string       `json:"lastName,omitempty" db:"lastName"`
	Photo              string       `json:"photo,omitempty" db:"photo"`
	ScreenName         string       `json:"screenName,omitempty" db:"screenName"`
	UserName           string       `json:"userName,omitempty" db:"userName"`
	Email              string       `json:"email,omitempty" db:"email"`
	Phone              string       `json:"phone,omitempty" db:"phone"`
	Type               string       `json:"type,omitempty"`
	Shareable          bool         `json:"shareable" db:"shareable"`
	DefaultThingsBoard string       `json:"defaultThingsBoard" db:"defaultThingsBoard"`
	OwnerDetails       OwnerDetails `json:"ownerDetails"`
	IsOrganization     bool         `json:"isOrganization"`
	IsPersonal         bool         `json:"isPersonal"`
	Thumbs             Thumbnails   `json:"thumbs"`
}

type OwnerDetails struct {
	Name   string     `json:"name,omitempty" db:"organizationName"`
	Photo  string     `json:"photo,omitempty" db:"photo"`
	Thumbs Thumbnails `json:"thumbs"`
}

type Profile struct {
	ID                      int       `json:"id" db:"id"`
	AccountID               int       `json:"accountID" db:"accountID"`
	DefaultThingsBoard      string    `json:"defaultThingsBoard" db:"defaultThingsBoard"`
	Photo                   string    `json:"photo" db:"photo"`
	FirstName               string    `json:"firstName" db:"firstName"`
	LastName                string    `json:"lastName" db:"lastName"`
	ScreenName              string    `json:"screenName" db:"screenName"`
	Bio                     string    `json:"bio" db:"bio"`
	Phone1                  string    `json:"phone1" db:"phone1"`
	Phone2                  string    `json:"phone2" db:"phone2"`
	Email1                  string    `json:"email1" db:"email1"`
	Email2                  string    `json:"email2" db:"email2"`
	Address1                string    `json:"address1" db:"address1"`
	Address2                string    `json:"address2" db:"address2"`
	City                    string    `json:"city" db:"city"`
	State                   string    `json:"state" db:"state"`
	Zip                     string    `json:"zip" db:"zip"`
	Country                 string    `json:"country" db:"country"`
	Birthday                string    `json:"birthday" db:"birthday"`
	Gender                  int       `json:"gender" db:"gender"`
	Notes                   string    `json:"notes" bson:"notes"`
	Visibility              string    `json:"visibility" db:"visibility"`
	SharedInfo              string    `json:"sharedInfo" db:"sharedInfo"`
	Shareable               bool      `json:"shareable" db:"shareable"`
	Searchable              bool      `json:"searchable" db:"searchable"`
	Tags                    string    `json:"tags" db:"tags"`
	TagsArr                 []string  `json:"tagsArr"`
	FollowMe                bool      `json:"followMe" db:"followMe"`
	ShowConnections         bool      `json:"showConnections" db:"showConnections"`
	ShowBoards              bool      `json:"showBoards" db:"showBoards"`
	ShowThingsFollowed      bool      `json:"showThingsFollowed" db:"showThingsFollowed"`
	ApproveGroupMemberships bool      `json:"approveGroupMemberships" db:"approveGroupMemberships"`
	TimeZone                string    `json:"timeZone" db:"timeZone"`
	NotificationsFromTime   string    `json:"notificationsFromTime" db:"notificationsFromTime"`
	NotificationsToTime     string    `json:"notificationsToTime" db:"notificationsToTime"`
	ManagedByID             int       `json:"managedByID" db:"managedByID"`
	CreateDate              time.Time `json:"createDate" db:"createDate"`
	ModifiedDate            time.Time `json:"modifiedDate" db:"modifiedDate"`
	DeletedDate             time.Time `json:"deleteDate" db:"deleteDate"`
	IsActive                string    `json:"isActive" db:"isActive"`
	ConnectCodeExpiration   string    `json:"connectCodeExpiration" db:"connectCodeExpiration"`

	NotificationSettings NotificationSettings `json:"notificationSettings"`
	ShareableSettings    ShareableSettings    `json:"shareableSettings" db:"shareableSettings"`
	CoManager            ConciseProfile       `json:"comanager"`
	Thumbs               Thumbnails           `json:"thumbs"`
	TotalNotifications   int64                `json:"totalNotifications"`
}

type OrgStaff struct {
	Id                    int        `json:"-" db:"id"`
	ProfileID             int        `json:"-" db:"profileID"`
	Photo                 string     `json:"photo" db:"photo"`
	Abv                   string     `json:"abv" db:"abv"`
	FirstName             string     `json:"firstName" db:"firstName"`
	LastName              string     `json:"lastName" db:"lastName"`
	NickName              string     `json:"nickName" db:"nickName"`
	Bio                   string     `json:"bio" db:"bio"`
	Phone1                string     `json:"phone1" db:"phone1"`
	Phone2                string     `json:"phone2" db:"phone2"`
	Email1                string     `json:"email1" db:"email1"`
	EmergencyEmail        string     `json:"emergencyEmail" db:"emergencyEmail"`
	Address               string     `json:"address1" db:"address1"`
	Address2              string     `json:"address2" db:"address2"`
	City                  string     `json:"city" db:"city"`
	State                 string     `json:"state" db:"state"`
	Zip                   string     `json:"zip" db:"zip"`
	Country               string     `json:"country" db:"country"`
	Gender                int        `json:"gender" db:"gender"`
	Birthday              string     `json:"birthday" db:"birthday"`
	EmergencyContact      string     `json:"emergencyContact" db:"emergencyContact"`
	EmergencyContactPhone string     `json:"emergencyContactPhone" db:"emergencyContactPhone"`
	EmergencyContactID    string     `json:"emergencyContactID" db:"emergencyContactID"`
	Notes                 string     `json:"notes" db:"notes"`
	StartDate             string     `json:"startDate" db:"startDate"`
	EndDate               string     `json:"endDate" db:"endDate"`
	JobTitle              string     `json:"jobTitle" db:"jobTitle"`
	Skills                string     `json:"skills" db:"skills"`
	Interests             string     `json:"interests" db:"interests"`
	ReportsToID           string     `json:"reportsToID" db:"reportsToID"`
	OrgID                 int        `json:"-" db:"orgID"`
	Thumbs                Thumbnails `json:"thumbs"`
}

type NotificationSettings struct {
	ID                 int  `json:"id" db:"id"`
	ProfileID          int  `json:"profileID" db:"profileID"`
	IsAllNotifications bool `json:"isAllNotifications" db:"isAllNotifications"`
	IsChatMessage      bool `json:"isChatMessage" db:"isChatMessage"`
	IsMention          bool `json:"isMention" db:"isMention"`
	IsInvite           bool `json:"isInvite" db:"isInvite"`
	IsBoardJoin        bool `json:"isBoardJoin" db:"isBoardJoin"`
	IsComment          bool `json:"isComment" db:"isComment"`
	IsReaction         bool `json:"isReaction" db:"isReaction"`
}

type UpdateProfileSettings struct {
	Profile    Profile `json:"profileSettings"`
	UpdateType string  `json:"updateType"`
}

type BasicProfileInfo struct {
	ID          int        `json:"connectionProfileID" db:"connectionProfileID"`
	FirstName   string     `json:"firstName" db:"firstName"`
	LastName    string     `json:"lastName" db:"lastName"`
	ScreenName  string     `json:"screenName" db:"screenName"`
	Photo       string     `json:"photo" db:"photo"`
	ManagedByID int        `json:"managedByID" db:"managedByID"`
	Thumbs      Thumbnails `json:"thumbs"`
}

// type BasicProfileInfo struct {
// 	ID         int    `json:"id" db:"id"`
// 	FirstName  string `json:"firstName" db:"firstName"`
// 	LastName   string `json:"lastName" db:"lastName"`
// 	ScreenName string `json:"screenName" db:"screenName"`
// 	Photo      string `json:"photo" db:"photo"`
// }

type FetchPeopleInfo struct {
	ConnectionProfileID int    `json:"connectionProfileID" db:"connectionProfileID"`
	FullName            string `json:"fullName" db:"fullName"`
	ScreenName          string `json:"screenName" db:"screenName"`
	Photo               string `json:"photo" db:"photo"`
	BoardTitle          string `json:"boardTitle" db:"boardTitle"`
	BoardID             string `json:"boardID" db:"boardID"`
}

type ProfileWithCoManager struct {
	ID             int                `json:"id" db:"id" `
	Name           string             `json:"name" db:"name"`
	Photo          string             `json:"photo" db:"photo"`
	ManagedByID    int                `json:"managedByID" db:"managedByID"`
	CoManagerName  string             `json:"comanagerName" db:"comanagerName" `
	ComanagerPhoto string             `json:"comanagerPhoto" db:"comanagerPhoto"`
	RequestInfo    *ConnectionRequest `json:"requestInfo"`
	ScreenName     string             `json:"screenName"`
	Thumbs         Thumbnails         `json:"thumbs"`
}

type ShareableSettings struct {
	ID         int  `json:"id" db:"id"`
	ProfileID  int  `json:"profileID" db:"profileID"`
	FirstName  bool `json:"firstName" db:"firstName"`
	LastName   bool `json:"lastName" db:"lastName"`
	ScreenName bool `json:"screenName" db:"screenName"`
	Bio        bool `json:"bio" db:"bio"`
	Email      bool `json:"email" db:"email"`
	Phone      bool `json:"phone" db:"phone"`
	Address    bool `json:"address1" db:"address1"`
	Address2   bool `json:"address2" db:"address2"`
	Gender     bool `json:"gender" db:"gender"`
	Birthday   bool `json:"birthday" db:"birthday"`
}

type FollowersInfo struct {
	BasicProfileInfo
	BoardInfo
}

type FollowingInfo struct {
	BasicProfileInfo
	BoardDetails []BoardInfo `json:"boardDetails"`
}

type MembershipInfo struct {
	AccountInfo struct {
		PlanDetails string  `json:"planDetails" db:"description"`
		Fee         float32 `json:"fee" db:"fee"`
		Profiles    int     `json:"profiles" db:"profiles"`
	} `json:"accountInfo"`
	CancellationPolicy string `json:"cancellationPolicy"`
	ExpirationDate     string `json:"expirationDate" db:"expirationDate"`
}

type OrganizationProfiles struct {
	ComanagerInfo BasicProfileInfo `json:"comanagerProfileInfo"`
	ProfileInfo   BasicProfileInfo `json:"profileInfo"`
}
type BoardInfo struct {
	BoardTitle string `json:"boardTitle" db:"boardTitle"`
	BoardID    string `json:"boardID" db:"boardID"`
}

type SearchHistory struct {
	ProfileID string   `json:"profileID" bson:"profileID"`
	History   []string `json:"history" bson:"history"`
}
type ProfileView struct {
	FirstName  string `json:"firstName" db:"firstName"`
	LastName   string `json:"lastName" db:"lastName"`
	ScreenName string `json:"screenName" db:"screenName"`
	Visibility string `json:"visibility" db:"visibility"`
	Email      string `json:"email" db:"email1"`
	// Email2      string `json:"email2" db:"email2"`
	Address1   string `json:"address1" db:"address1"`
	Address2   string `json:"address2" db:"address2"`
	City       string `json:"city" db:"city"`
	State      string `json:"state" db:"state"`
	Zip        string `json:"zip" db:"zip"`
	Country    string `json:"country" db:"country"`
	Gender     int    `json:"gender" db:"gender"`
	ShowBoards int    `json:"showBoards" db:"showBoards"`
	Birthday   string `json:"birthday" db:"birthday"`
	Bio        string `json:"bio" db:"bio"`
	Phone      string `json:"phone" db:"phone1"`
	// Phone2      string `json:"phone2" db:"phone2"`
	Shareable   int    `json:"shareable" db:"shareable"`
	IsConnected bool   `json:"isConnected"`
	IsPrivate   bool   `json:"-"`
	Photo       string `json:"photo"`
}
