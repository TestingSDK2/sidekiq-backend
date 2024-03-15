package collection

import (
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/config"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/storage"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/cache"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/database"
	repo "github.com/TestingSDK2/sidekiq-backend/sidekiq-content/model"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/mongodatabase"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-models/model"

	peoplerpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Service interface {
	AddCollection(payload model.Collection, profileID int, boardID, postID string) (map[string]interface{}, error)
	GetCollection(boardID, postID string, profileID int, owner string, tagsArr []string, uploadDate string, limit int, page, l string) (map[string]interface{}, error)
	UpdateCollection(payload model.UpdateCollection, boardID, postID,
		collectionID string, profileID int) (map[string]interface{}, error)
	GetCollectionByID(boardID, postID, collectionID string, profileID int) (map[string]interface{}, error)
	UpdateCollecitonStatusByID(boardID, postID, collectionID, status string, profileID int) (map[string]interface{}, error)
	AppendThingInCollection(payload model.Collection, boardID string, profileID int) (map[string]interface{}, error)
	FetchFilesByCollection(boardID, postID, collectionID, fileName string, profileID, limit int, page, l string, ownerInfo *peoplerpc.ConciseProfileReply) (map[string]interface{}, error)
	DeleteCollectionMedia(colId, thingID string) (map[string]interface{}, error)
	EditCollectionMedia(payload model.UpdateCollection, profileID, thingID, thingType, boardID, postID string) (map[string]interface{}, error)
	DeleteCollection(boardID, postID, collectionID, profileID string) (map[string]interface{}, error)
	UpdateCollectionById(collectionID primitive.ObjectID, payload map[string]interface{}) error
}

type service struct {
	config         *config.Config
	mongodb        *mongodatabase.DBConfig
	dbMaster       *database.Database
	dbReplica      *database.Database
	storageService storage.Service
	cache          *cache.Cache
	peopleRpc      peoplerpc.AccountServiceClient
}

// NewService - creates new File service
func NewService(repos *repo.Repos, conf *config.Config) Service {
	return &service{
		config:         conf,
		mongodb:        repos.MongoDB,
		dbMaster:       repos.MasterDB,
		dbReplica:      repos.ReplicaDB,
		storageService: storage.NewService(repos, conf),
		cache:          repos.Cache,
		peopleRpc:      repos.PeopleGrpcServiceClient,
	}
}

func (s *service) AddCollection(payload model.Collection, profileID int, boardID, postID string) (map[string]interface{}, error) {
	return addCollection(s.cache, s.mongodb, s.dbMaster, s.peopleRpc, s.storageService, payload, profileID, boardID, postID)
}

func (s *service) GetCollection(boardID, postID string, profileID int, owner string, tagsArr []string, uploadDate string, limit int, page, l string,
) (map[string]interface{}, error) {
	return getCollection(s.cache, s.mongodb, s.dbMaster, s.peopleRpc, s.storageService, boardID, postID, profileID, owner, tagsArr, uploadDate, limit, page, l)
}

func (s *service) UpdateCollection(payload model.UpdateCollection, boardID, postID, collectionID string,
	profileID int) (map[string]interface{}, error) {
	return updateCollection(s.cache, payload, s.mongodb, boardID, postID, collectionID, profileID, s.storageService)
}

func (s *service) GetCollectionByID(boardID, postID, collectionID string, profileID int) (map[string]interface{}, error) {
	return getCollectionByID(s.cache, s.mongodb, s.dbMaster, s.peopleRpc, s.storageService, boardID, postID, collectionID, profileID)
}

func (s *service) UpdateCollecitonStatusByID(boardID, postID, collectionID, status string, profileID int) (map[string]interface{}, error) {
	return updateCollecitonStatusByID(s.cache, s.mongodb, s.dbMaster, s.peopleRpc, s.storageService, boardID, postID, collectionID, status, profileID)
}

func (s *service) AppendThingInCollection(payload model.Collection, boardID string, profileID int) (map[string]interface{}, error) {
	return appendThingInCollection(s.mongodb, s.peopleRpc, s.storageService, payload, boardID, profileID)
}

func (s *service) FetchFilesByCollection(boardID, postID, collectionID, fileName string, profileID, limit int, page, l string, ownerInfo *peoplerpc.ConciseProfileReply) (map[string]interface{}, error) {
	return getFilesByCollection(s.mongodb, s.dbMaster, s.cache, s.peopleRpc, s.storageService, boardID, postID, collectionID, fileName, profileID, limit, page, l, ownerInfo)
}

func (s *service) DeleteCollectionMedia(colId, thingID string) (map[string]interface{}, error) {
	return deleteCollectionMedia(s.mongodb, colId, thingID)
}

func (s *service) EditCollectionMedia(payload model.UpdateCollection, profileID, thingID, thingType, boardID, postID string) (map[string]interface{}, error) {
	return editCollectionMedia(s.cache, s.mongodb, payload, profileID, thingID, thingType, boardID, postID)
}

func (s *service) DeleteCollection(boardID, postID, collectionID, profileID string) (map[string]interface{}, error) {
	return deleteCollection(s.mongodb, s.cache, boardID, postID, collectionID, profileID)
}

func (s *service) UpdateCollectionById(collectionID primitive.ObjectID, payload map[string]interface{}) error {
	return updateCollectionById(s.mongodb, collectionID, payload)
}
