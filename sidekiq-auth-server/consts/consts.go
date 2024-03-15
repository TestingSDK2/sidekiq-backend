package consts

// thing type
const (
	Board           = "Board"
	Post            = "Post"
	Collection      = "thCollection"
	Note            = "thNote"
	Task            = "thTask"
	File            = "thFile"
	Trash           = "Trash"
	Request         = "Request"
	Connection      = "Connection"
	Recent          = "Recent"
	SearchHistory   = "SearchHistory"
	BoardThingsTags = "BoardThingsTags"
	Bookmark        = "Bookmark"
	Activity        = "Activity"
	Notification    = "Notification"
	BoardType       = "BOARD"
	FileType        = "FILE"
	NoteType        = "NOTE"
	TaskType        = "TASK"
	SearchResult    = "SearchResult"
)

// roles
const (
	Owner      = "owner"
	Admin      = "admin"
	Author     = "author"
	Guest      = "guest"
	Subscriber = "subscriber"
	Viewer     = "viewer"
	Blocked    = "blocked"
)

// priority
const (
	High   = "HIGH"
	Normal = "NORMAL"
	Low    = "LOW"
)

// visible
const (
	Private  = "PRIVATE"
	Member   = "MEMBERS"
	Contacts = "CONTACTS"
	Public   = "PUBLIC"
)

// state
const (
	Active  = "ACTIVE"
	Draft   = "DRAFT"
	Hidden  = "HIDE"
	Archive = "ARCHIVE"
)

// saving
const (
	NoSaving       = 0
	SaveToFile     = 1
	SaveToBookmark = 2
)

// Notification
const (
	DeleteConnection        = "DeleteConnection"
	BoardFollowed           = "BoardFollowed"
	AcceptConnectionRequest = "AcceptConnectionRequest"
	TaskInitiated           = "TaskInitiated"
	TaskStatusUpdated       = "TaskStatusUpdated"
	TaskUpdated             = "TaskUpdated"
	AddComment              = "AddComment"
)

// FILE status
const (
	FileCOMPLETE   = "COMPLETE"
	FilePROCESSING = "PROCESSING"
)

//Post Thing Blocked or Unblocked event

const (
	PostThingBlocked   = "PostThingBlocked"
	PostThingUnBlocked = "PostThingUnblocked"
)
