package post

import (
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/board"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/config"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/file"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/note"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/profile"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/storage"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/task"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/cache"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/database"
	repo "github.com/TestingSDK2/sidekiq-backend/sidekiq-content/model"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/mongodatabase"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-models/model"

	// peoplerpc "github.com/sidekiq-people/proto/people"
	peoplerpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Service interface {
	FindPost(boardID, postID string) (map[string]interface{}, error)
	FindPostByPostID(postID string) (map[string]interface{}, error)
	AddPost(profile int, boardID string, post model.Post) (map[string]interface{}, error)
	GetPostsOfBoard(profileID int, boardID, page, limit, filterBy, sortBy string) (map[string]interface{}, error)
	GetPostThings(boardID, postID string, profileID int) ([]map[string]interface{}, error)
	DeletePost(postID string) error
	MovePost(post model.Post, postID, trgtBoard string) error
	UpdatePostSettings(profileID int, postID string, post model.Post, payload map[string]interface{}) (map[string]interface{}, error)
	GetFirstPostThing(postID, boardID string, profileID int) (map[string]interface{}, error)
	GetThumbnailAndImageforPostThing(postID, boardID string, profileID int, reqThing map[string]interface{}) (map[string]interface{}, error)
	UpdatePostThing(reqThings []map[string]interface{}, postObjId primitive.ObjectID, profileID string) error
	UpdatePostThingUnblocked(postObjId primitive.ObjectID, profileID string) error
	DeleteSelectedPostThing(reqThings []map[string]interface{}, postObjId primitive.ObjectID) error
}
type service struct {
	config         *config.Config
	mongodb        *mongodatabase.DBConfig
	dbMaster       *database.Database
	dbReplica      *database.Database
	profileService profile.Service
	taskService    task.Service
	noteService    note.Service
	fileService    file.Service
	storageService storage.Service
	boardService   board.Service
	cache          *cache.Cache
	peopleRpc      peoplerpc.AccountServiceClient
}

func NewService(repos *repo.Repos, conf *config.Config) Service {
	return &service{
		config:         conf,
		mongodb:        repos.MongoDB,
		dbMaster:       repos.MasterDB,
		dbReplica:      repos.ReplicaDB,
		profileService: profile.NewService(repos, conf),
		storageService: storage.NewService(repos, conf),
		taskService:    task.NewService(repos, conf),
		noteService:    note.NewService(repos, conf),
		fileService:    file.NewService(repos, conf),
		boardService:   board.NewService(repos, conf),
		cache:          repos.Cache,
		peopleRpc:      repos.PeopleGrpcServiceClient,
	}
}

func (s *service) GetFirstPostThing(postID, boardID string, profileID int) (map[string]interface{}, error) {
	return getFirstPostThingV2(s.mongodb, s.boardService, s.peopleRpc, s.storageService, postID, boardID, profileID)
}

func (s *service) GetPostsOfBoard(profileID int, boardID, limit, page, filterBy, sortBy string) (map[string]interface{}, error) {
	return getPostsOfBoardV2(s.mongodb, s.peopleRpc, s.cache, s.boardService, profileID, boardID, limit, page, filterBy, sortBy)
}

func (s *service) GetThumbnailAndImageforPostThing(postID, boardID string, profileID int, reqThing map[string]interface{}) (map[string]interface{}, error) {
	return getThumbnailAndImageforPostThing(s.mongodb, s.boardService, s.peopleRpc, s.storageService, postID, boardID, profileID, reqThing)
}

func (s *service) FindPost(boardID, postID string) (map[string]interface{}, error) {
	return findPost(s.mongodb, boardID, postID)
}

func (s *service) FindPostByPostID(postID string) (map[string]interface{}, error) {
	return findPostByPostID(s.mongodb, postID)
}

func (s *service) AddPost(profileID int, boardID string, post model.Post) (map[string]interface{}, error) {
	return addPost(s.mongodb, s.cache, s.peopleRpc, s.storageService, profileID, boardID, post)
}

func (s *service) GetPostThings(boardID, postID string, profileID int) ([]map[string]interface{}, error) {
	return getPostThings(s.mongodb, s.cache, s.peopleRpc, s.noteService, s.taskService, s.fileService, boardID, postID, profileID)
}

func (s *service) DeletePost(postID string) error {
	return deletePost(s.mongodb, postID)
}

func (s *service) MovePost(post model.Post, postID, trgtBoard string) error {
	return movePost(s.mongodb, post, postID, trgtBoard, s.storageService, s.boardService, s.peopleRpc)
}

func (s *service) UpdatePostSettings(profileID int, postID string, post model.Post, payload map[string]interface{}) (map[string]interface{}, error) {
	return updatePostSettings(s.mongodb, s.cache, profileID, postID, post, payload)
}

func (s *service) UpdatePostThing(reqThings []map[string]interface{}, postObjId primitive.ObjectID, profileID string) error {
	return updatePostThing(s.mongodb, reqThings, postObjId, profileID)
}

func (s *service) DeleteSelectedPostThing(reqThings []map[string]interface{}, postObjId primitive.ObjectID) error {
	return deleteSelectedPostThing(s.mongodb, reqThings, postObjId)
}

func (s *service) UpdatePostThingUnblocked(postObjId primitive.ObjectID, profileID string) error {
	return updatePostThingUnblocked(s.mongodb, postObjId, profileID)
}
