package task

import (
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/board"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/config"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/profile"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/storage"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/cache"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/database"
	repo "github.com/TestingSDK2/sidekiq-backend/sidekiq-content/model"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/mongodatabase"
	model "github.com/TestingSDK2/sidekiq-backend/sidekiq-models"

	// peoplerpc "github.com/sidekiq-people/proto/people"
	peoplerpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Service interface {
	FetchTasksOfBoard(boardID string, profileID int, owner string, tagArr []string, uploadDate string, limit int, page, l string) (map[string]interface{}, error)
	AddTask(postID primitive.ObjectID, task map[string]interface{}) (map[string]interface{}, error)
	AddTasks(postID primitive.ObjectID, tasks []interface{}) error
	AddTaskInCollection(collectionID string, payload model.Task, profileID int) (map[string]interface{}, error)
	UpdateTask(payload map[string]interface{}, boardID, postID, taskID string, profileID int) (map[string]interface{}, error)
	DeleteTask(boardID, taskID string, profileID int) (map[string]interface{}, error)
	GetTaskByID(taskID string, profileID int) (map[string]interface{}, error)
	GetActionTask(profileID int, sortBy, orderBy string, limitInt, pageInt int, filterBy string) (map[string]interface{}, error)
	FetchTasksByProfile(boardID string, profileID, limit int, publicOnly bool) (map[string]interface{}, error)
	FetchTasksByPost(boardID, postID string) ([]map[string]interface{}, error)
	DeleteTasksOnPost(postID string) error
}

type service struct {
	config       *config.Config
	mongodb      *mongodatabase.DBConfig
	dbMaster     *database.Database
	dbReplica    *database.Database
	boardService board.Service
	// postService    post.Service
	profileService profile.Service
	storageService storage.Service
	cache          *cache.Cache
	peopleRpc      peoplerpc.AccountServiceClient
}

// NewService - creates new File service
func NewService(repos *repo.Repos, conf *config.Config) Service {
	return &service{
		config:       conf,
		mongodb:      repos.MongoDB,
		dbMaster:     repos.MasterDB,
		dbReplica:    repos.ReplicaDB,
		boardService: board.NewService(repos, conf),
		// postService:    post.NewService(repos, conf),
		profileService: profile.NewService(repos, conf),
		storageService: storage.NewService(repos, conf),
		cache:          repos.Cache,
		peopleRpc:      repos.PeopleGrpcServiceClient,
	}
}

func (s *service) AddTasks(postID primitive.ObjectID, tasks []interface{}) error {
	return addTasks(s.mongodb, s.peopleRpc, s.storageService, postID, tasks)
}

func (s *service) FetchTasksOfBoard(boardID string, profileID int, owner string, tagArr []string, uploadDate string, limit int, page, l string) (map[string]interface{}, error) {
	return getTasksOfBoard(s.mongodb, s.dbMaster, s.cache, s.boardService, s.peopleRpc, s.storageService, boardID, profileID, owner, tagArr, uploadDate, limit, page, l)
}

func (s *service) AddTask(postID primitive.ObjectID, task map[string]interface{}) (map[string]interface{}, error) {
	return addTask(s.mongodb, s.peopleRpc, s.storageService, postID, task)
}

func (s *service) UpdateTask(payload map[string]interface{}, boardID, postID, taskID string, profileID int) (map[string]interface{}, error) {
	return updateTask(s.mongodb, s.cache, payload, boardID, postID, taskID, profileID)
}

func (s *service) DeleteTask(boardID, taskID string, profileID int) (map[string]interface{}, error) {
	return deleteTask(s.mongodb, s.cache, boardID, taskID, profileID)
}

func (s *service) GetTaskByID(taskID string, profileID int) (map[string]interface{}, error) {
	return getTaskByID(s.mongodb, s.peopleRpc, s.storageService, taskID, profileID)
}

func (s *service) FetchTasksByProfile(boardID string, profileID, limit int, publicOnly bool) (map[string]interface{}, error) {
	return fetchTasksByProfile(s.cache, s.mongodb, s.dbMaster, boardID, profileID, limit, publicOnly)
}

func (s *service) AddTaskInCollection(collectionID string, payload model.Task, profileID int) (map[string]interface{}, error) {
	return addTaskInCollection(s.mongodb, s.peopleRpc, s.storageService, s.cache, collectionID, payload, profileID)
}

func (s *service) FetchTasksByPost(boardID, postID string) ([]map[string]interface{}, error) {
	return fetchTasksByPost(s.mongodb, boardID, postID, s.storageService, s.peopleRpc)
}

func (s *service) DeleteTasksOnPost(postID string) error {
	return deleteTasksOnPost(s.mongodb, postID)
}

func (s *service) GetActionTask(profileID int, sortBy, orderBy string, limitInt, pageInt int, filterBy string) (map[string]interface{}, error) {
	return getActionTask(s.mongodb, s.cache, s.peopleRpc, s.storageService, profileID, sortBy, orderBy, limitInt, pageInt, filterBy)
}
