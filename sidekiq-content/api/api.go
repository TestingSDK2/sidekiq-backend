package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/api/board"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/api/collection"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/api/common"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/api/dashboard"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/api/file"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/api/note"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/api/post"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/api/storage"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/api/task"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/api/thing"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/api/thingactivity"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/cache"
)

// API sidekiq api
type API struct {
	App    *app.App
	Config *common.Config
	Cache  *cache.Cache
}

// New creates a new api
func New(a *app.App) (api *API, err error) {
	api = &API{App: a}
	api.Config, err = common.InitConfig()
	if err != nil {
		return nil, err
	}
	return api, nil
}

// Init initializes the api
func (a *API) Init(r *mux.Router) {
	// SERVER-STATUS
	r.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"OKK","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
	})

	/* ****************** DASHBOARD ****************** */
	dashBoardAPI := dashboard.New(a.Config,
		a.App.RecentThingsService, a.App.BoardService, a.App.ProfileService,
		a.App.StorageService, a.App.Repos)
	r.Handle("/dashboard", a.handler(dashBoardAPI.FetchAll, true)).Methods(http.MethodGet)

	/*  ****************** BOARD ****************** */
	boardAPI := board.New(a.Config, a.App.BoardService, a.App.RecentThingsService,
		a.App.TaskService, a.App.NoteService,
		a.App.FileService, a.App.ProfileService, a.App.CollectionService,
		a.App.StorageService, a.App.ThingActivityService, a.App.Repos,
		a.App.ThingService, a.App.MessageService, a.App.PostService)
	r.Handle("/board/search", a.handler(boardAPI.SearchBoards, true)).Methods(http.MethodGet)
	r.Handle("/board", a.handler(boardAPI.FetchBoards, true)).Methods(http.MethodGet)
	r.Handle("/board/listing", a.handler(boardAPI.FetchBoardsListing, true)).Methods(http.MethodGet)
	r.Handle("/recent/remove", a.handler(boardAPI.RecentRemove, true)).Methods(http.MethodDelete)
	r.Handle("/board", a.handler(boardAPI.AddBoard, false, false)).Methods(http.MethodPost)

	// SUB-BOARDS
	r.Handle("/board/sub", a.handler(boardAPI.FetchSubBoardsOfProfile, true)).Methods(http.MethodGet)
	r.Handle("/board/{boardID}/sub", a.handler(boardAPI.FetchSubBoardsOfBoard, true)).Methods(http.MethodGet)

	// SHARED BOARDS
	r.Handle("/board/shared", a.handler(boardAPI.GetSharedBoards, true)).Methods(http.MethodGet)

	// BOARD INVITATION
	r.Handle("/board/invitations", a.handler(boardAPI.ListBoardInvitations, true)).Methods(http.MethodGet)
	r.Handle("/board/invitations/handle", a.handler(boardAPI.HandleBoardInvitation, true)).Methods(http.MethodPut)

	r.Handle("/board/{boardID}", a.handler(boardAPI.FetchBoardByID, false, false)).Methods(http.MethodPost)
	r.Handle("/board/{boardID}", a.handler(boardAPI.UpdateBoard, true)).Methods(http.MethodPut)
	r.Handle("/board/{boardID}", a.handler(boardAPI.DeleteBoard, true)).Methods(http.MethodDelete)

	// BOARD THINGS
	r.Handle("/board/{boardID}/things", a.handler(boardAPI.FetchBoardThings, true)).Methods(http.MethodPost) // NOT USED
	r.Handle("/board/{boardID}/things/profile", a.handler(boardAPI.FetchBoardThingsByProfile, true)).Methods(http.MethodGet)
	// r.Handle("/board/{boardID}/things/view", a.handler(boardAPI.FetchPublicBoardThings, true)).Methods(http.MethodGet)

	// MEMBERS
	r.Handle("/board/{boardID}/members/list", a.handler(boardAPI.FetchBoardMembers, false)).Methods(http.MethodGet)
	r.Handle("/board/{boardID}/members/invite", a.handler(boardAPI.InviteMembers, true)).Methods(http.MethodPost)
	r.Handle("/board/{boardID}/members/role/update", a.handler(boardAPI.ChangeProfileRole, true)).Methods(http.MethodPut)
	r.Handle("/board/{boardID}/members/list/connections", a.handler(boardAPI.FetchMembersFromConnections, true)).Methods(http.MethodGet)
	r.Handle("/board/{boardID}/members/block", a.handler(boardAPI.BlockMembers, true)).Methods(http.MethodPut)
	r.Handle("/board/{boardID}/members/block", a.handler(boardAPI.ListBlockedMembers, true)).Methods(http.MethodGet)
	r.Handle("/board/{boardID}/members/unblock", a.handler(boardAPI.UnblockMembers, true)).Methods(http.MethodPut)
	r.Handle("/board/{boardID}/members/remove", a.handler(boardAPI.RemoveMembers, true)).Methods(http.MethodPut)
	r.Handle("/board/{boardID}/members/viewer", a.handler(boardAPI.AddViewerInBoard, true)).Methods(http.MethodPut)

	// FOLLOWERS
	r.Handle("/board/{boardID}/follow", a.handler(boardAPI.GetBoardFollowers, true)).Methods(http.MethodGet)
	r.Handle("/board/{boardID}/unfollow", a.handler(boardAPI.BoardUnfollow, true)).Methods(http.MethodGet)
	r.Handle("/board/{boardID}/follow", a.handler(boardAPI.BoardFollow, true)).Methods(http.MethodPost)

	// SEARCH
	// r.Handle("/board/{boardID}/search", a.handler(boardAPI.FullTextSearch, true)).Methods(http.MethodGet)
	// r.Handle("/board/{boardID}/filter", a.handler(boardAPI.BoardFilter, true)).Methods(http.MethodPost)
	r.Handle("/board/{boardID}/settings", a.handler(boardAPI.BoardSettings, true)).Methods(http.MethodPost)
	r.Handle("/board/{boardID}/things/owner", a.handler(boardAPI.FetchBoardThingOwners, true)).Methods(http.MethodGet)
	r.Handle("/board/{boardID}/things/ext", a.handler(boardAPI.FetchBoardThingExt, true)).Methods(http.MethodGet)

	/* ****************** POST ****************** */
	postAPI := post.New(a.Config, a.App.BoardService, a.App.PostService,
		a.App.NoteService, a.App.TaskService, a.App.ProfileService,
		a.App.RecentThingsService, a.App.StorageService, a.App.ThingService,
		a.App.ThingActivityService, a.App.Repos, a.App.CollectionService)
	r.Handle("/board/{boardID}/post", a.handler(postAPI.AddPost, true)).Methods(http.MethodPost)
	r.Handle("/board/{boardID}/post", a.handler(postAPI.FetchPostsOfBoard, true)).Methods(http.MethodGet)
	r.Handle("/board/{boardID}/post/{postID}", a.handler(postAPI.DeletePost, true)).Methods(http.MethodDelete)
	r.Handle("/board/{boardID}/post/{postID}/settings", a.handler(postAPI.UpdatePostSettings, true)).Methods(http.MethodPut)
	r.Handle("/board/{boardID}/post/{postID}/things", a.handler(postAPI.AddThingsOnPost, true)).Methods(http.MethodPost)
	r.Handle("/board/{boardID}/post/{postID}/things", a.handler(postAPI.FetchPostThings, true)).Methods(http.MethodGet)
	r.Handle("/board/{boardID}/post/{postID}/things", a.handler(postAPI.UpdateThings, true)).Methods(http.MethodPut)
	r.Handle("/board/{boardID}/post/{postID}/things", a.handler(postAPI.DeleteSelectedThings, true)).Methods(http.MethodDelete)
	r.Handle("/board/{boardID}/post/{postID}/things/unblocked", a.handler(postAPI.UpdatePostThingsUnblocked, true)).Methods(http.MethodPut)
	r.Handle("/board/{boardID}/post/{postID}/send/thing/event/board/member", a.handler(postAPI.SendThingEvent, true)).Methods(http.MethodPost)
	r.Handle("/board/{boardID}/post/{postID}/move", a.handler(postAPI.MovePost, true)).Methods(http.MethodPut)

	/* ****************** FILE METADATA (Mongo) ****************** */
	fileAPI := file.New(a.Config, a.App.FileService,
		a.App.BoardService, a.App.ProfileService, a.App.StorageService,
		a.App.ThingActivityService, a.App.PostService, a.App.Repos,
		a.App.ThingService)
	r.Handle("/board/{boardID}/file", a.handler(fileAPI.FetchFilesByBoard, true)).Methods(http.MethodGet)
	r.Handle("/board/{boardID}/file", a.handler(fileAPI.AddFile, true)).Methods(http.MethodPost)
	r.Handle("/board/{boardID}/file/{fileID}", a.handler(fileAPI.GetFileByID, true)).Methods(http.MethodGet)
	r.Handle("/board/{boardID}/post/{postID}/file/{fileID}", a.handler(fileAPI.UpdateFile, true)).Methods(http.MethodPut)
	r.Handle("/board/{boardID}/post/{postID}/file/{fileID}", a.handler(fileAPI.DeleteFile, true)).Methods(http.MethodDelete)
	r.Handle("/board/{boardID}/file/{fileName}", a.handler(fileAPI.FetchFileByName, true)).Methods(http.MethodGet)

	/* ****************** NOTE ****************** */
	noteAPI := note.New(a.Config, a.App.BoardService,
		a.App.PostService, a.App.NoteService, a.App.ProfileService,
		a.App.RecentThingsService, a.App.StorageService, a.App.ThingService,
		a.App.ThingActivityService, a.App.Repos)
	r.Handle("/board/{boardID}/post/{postID}/note", a.handler(noteAPI.AddNote, true)).Methods(http.MethodPost)
	r.Handle("/board/{boardID}/post/{postID}/note/{noteID}", a.handler(noteAPI.GetNoteByID, true)).Methods(http.MethodGet)
	r.Handle("/board/{boardID}/note", a.handler(noteAPI.FetchNotesByBoard, true)).Methods(http.MethodGet)
	r.Handle("/board/{boardID}/post/{postID}/note/{noteID}", a.handler(noteAPI.UpdateNote, true)).Methods(http.MethodPut)
	r.Handle("/board/{boardID}/post/{postID}/note/{noteID}", a.handler(noteAPI.DeleteNote, true)).Methods(http.MethodDelete)

	/* ****************** TASK ****************** */
	taskAPI := task.New(a.Config, a.App.BoardService,
		a.App.TaskService, a.App.ProfileService, a.App.RecentThingsService,
		a.App.StorageService, a.App.ThingService,
		a.App.ThingActivityService, a.App.Repos,
		a.App.PostService)
	r.Handle("/board/{boardID}/task", a.handler(taskAPI.FetchTasksOfBoard, true)).Methods(http.MethodGet)
	r.Handle("/board/{boardID}/post/{postID}/task", a.handler(taskAPI.AddTask, true)).Methods(http.MethodPost)
	r.Handle("/board/{boardID}/post/{postID}/task/{taskID}", a.handler(taskAPI.GetTaskByID, true)).Methods(http.MethodGet)
	r.Handle("/board/{boardID}/post/{postID}/task/{taskID}", a.handler(taskAPI.UpdateTask, true)).Methods(http.MethodPut)
	r.Handle("/board/{boardID}/post/{postID}/task/{taskID}", a.handler(taskAPI.DeleteTask, true)).Methods(http.MethodDelete)
	r.Handle("/action/list", a.handler(taskAPI.ActionList, true)).Methods(http.MethodGet)

	/* ****************** BOARD POST MEDIA (WASABI) ****************** */
	storageAPI := storage.New(a.Config, a.App.FileService, a.App.StorageService,
		a.App.ProfileService, a.App.RecentThingsService, a.App.BoardService,
		a.App.CollectionService, a.App.TaskService, a.App.NoteService,
		a.App.ThingActivityService, a.App.Repos, a.App.PostService,
		a.App.ThingService)

	// BOARD MEDIA
	r.Handle("/board/{boardID}/post/{postID}/media", a.handler(storageAPI.GetAllBoardPostMedia, true)).Methods(http.MethodGet)
	r.Handle("/board/{boardID}/post/{postID}/media/{mediaID}", a.handler(storageAPI.GetBoardPostMedia, true)).Methods(http.MethodGet)
	r.Handle("/board/{boardID}/post/{postID}/media/{mediaID}", a.handler(storageAPI.DeleteMedia, true)).Methods(http.MethodDelete)
	r.Handle("/board/{boardID}/post/{postID}/upload/{id}", a.handler(storageAPI.GetMediaUploadStatus, true)).Methods(http.MethodGet)
	r.Handle("/board/{boardID}/post/{postID}/upload/{id}/progress", a.handler(storageAPI.GetUploadProgress, true)).Methods(http.MethodGet)
	r.Handle("/user/storage", a.handler(storageAPI.ComputeCloudStorage, false)).Methods(http.MethodPost)
	r.Handle("/media/chunks/remove", a.handler(storageAPI.RemoveMediaChunks, true)).Methods(http.MethodDelete)

	// BOARD COVER
	r.Handle("/board/{boardID}/media/cover", a.handler(storageAPI.GetBoardCover, true)).Methods(http.MethodGet)
	r.Handle("/board/{boardID}/media/cover", a.handler(storageAPI.DeleteBoardCover, true)).Methods(http.MethodDelete)

	/* ****************** COLLECTION ****************** */
	collectionAPI := collection.New(a.Config, a.App.FileService,
		a.App.StorageService, a.App.ProfileService, a.App.CollectionService,
		a.App.TaskService, a.App.NoteService, a.App.RecentThingsService,
		a.App.ThingActivityService, a.App.Repos, a.App.BoardService,
		a.App.PostService, a.App.ThingService)

	r.Handle("/board/{boardID}/post/{postID}/collection", a.handler(collectionAPI.GetCollection, true)).Methods(http.MethodGet)
	r.Handle("/board/{boardID}/post/{postID}/collection", a.handler(collectionAPI.AddCollection, true)).Methods(http.MethodPost)
	r.Handle("/board/{boardID}/post/{postID}/collection/{collectionID}", a.handler(collectionAPI.GetCollectionByID, true)).Methods(http.MethodGet)
	r.Handle("/board/{boardID}/post/{postID}/collection/{collectionID}/status", a.handler(collectionAPI.UpdateCollectionStatus, true)).Methods(http.MethodPut)
	r.Handle("/board/{boardID}/post/{postID}/collection/{collectionID}/upload/{id}", a.handler(collectionAPI.GetMediaUploadStatusCollection, true)).Methods(http.MethodGet)
	r.Handle("/board/{boardID}/post/{postID}/collection/{collectionID}", a.handler(collectionAPI.UpdateCollection, true)).Methods(http.MethodPut)
	r.Handle("/board/{boardID}/post/{postID}/collection/{collectionID}", a.handler(collectionAPI.DeleteCollection, true)).Methods(http.MethodDelete)
	r.Handle("/board/{boardID}/post/{postID}/collection/{collectionID}/file", a.handler(collectionAPI.FetchFilesByCollection, true)).Methods(http.MethodGet)
	r.Handle("/board/{boardID}/post/{postID}/collection/{collectionID}/file/{fileID}", a.handler(collectionAPI.EditCollectionMedia, true)).Methods(http.MethodPut)
	r.Handle("/board/{boardID}/post/{postID}/collection/{collectionID}/file/{fileID}", a.handler(collectionAPI.DeleteCollectionMedia, true)).Methods(http.MethodDelete)

	/* ****************** THING ****************** */
	thingAPI := thing.New(a.Config, a.App.ThingService,
		a.App.ProfileService, a.App.BoardService, a.App.TaskService,
		a.App.NoteService, a.App.FileService, a.App.StorageService,
		a.App.RecentThingsService, a.App.Repos.MongoDB, a.App.Repos,
		a.App.PostService, a.App.CollectionService)

	r.Handle("/board/{boardID}/thing/{thingType}/{thingID}", a.handler(thingAPI.OpenThing, true)).Methods(http.MethodGet)
	r.Handle("/board/{boardID}/thing/{thingID}/{thingType}/{reactionType}", a.handler(thingAPI.FetchReactions, true)).Methods(http.MethodGet)
	r.Handle("/board/{boardID}/thing/{thingID}/{thingType}/like", a.handler(thingAPI.LikeThing, true)).Methods(http.MethodPost)
	r.Handle("/board/{boardID}/thing/{thingID}/{thingType}/like", a.handler(thingAPI.DislikeThing, true)).Methods(http.MethodDelete)
	r.Handle("/board/{boardID}/thing/{thingID}/{thingType}/comment", a.handler(thingAPI.AddComment2, true)).Methods(http.MethodPost)
	r.Handle("/board/{boardID}/thing/{thingID}/{thingType}/comment/{commentID}", a.handler(thingAPI.EditComment, true)).Methods(http.MethodPut)
	r.Handle("/board/{boardID}/thing/{thingID}/{thingType}/comment/{commentID}", a.handler(thingAPI.DeleteComment, true)).Methods(http.MethodDelete)
	r.Handle("/board/{boardID}/post/{postID}/thing/{thingID}/{thingType}/update", a.handler(thingAPI.UpdateThing, true)).Methods(http.MethodPut)

	// THING BOOKMARK
	r.Handle("/bookmark", a.handler(thingAPI.AddBookmark, true)).Methods(http.MethodPost)
	r.Handle("/bookmark", a.handler(thingAPI.GetAllBookmarks, true)).Methods(http.MethodGet)
	r.Handle("/bookmark", a.handler(thingAPI.DeleteAllBookmarks, true)).Methods(http.MethodDelete)
	r.Handle("/bookmark/{bookmarkID}", a.handler(thingAPI.DeleteBookmark, true)).Methods(http.MethodDelete)

	activityAPI := thingactivity.New(a.Config, a.App.ThingActivityService)
	r.Handle("/activity/{thingID}", a.handler(activityAPI.ListAllThingActivities, true)).Methods(http.MethodGet)
}
