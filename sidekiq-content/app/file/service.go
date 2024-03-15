package file

import (
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/board"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/config"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/profile"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/storage"

	"github.com/ProImaging/sidekiq-backend/sidekiq-content/cache"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/database"
	repo "github.com/ProImaging/sidekiq-backend/sidekiq-content/model"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/mongodatabase"

	// peoplerpc "github.com/sidekiq-people/proto/people"
	peoplerpc "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"
)

// Service - defines File service
type Service interface {
	FetchFilesByBoard(boardID string, profileID int, fileType, owner string, tagsArr []string, uploadDate string, limit int, page, l string) (map[string]interface{}, error)
	FetchFilesByPost(boardID, postID string, ownerInfo *peoplerpc.ConciseProfileReply, profileID int, fileType, owner string, tagsArr []string, uploadDate string, limit int, page, l string) (map[string]interface{}, error)
	FetchFilesByPost2(boardID, postID string) ([]map[string]interface{}, error)
	FetchFileByMediaIDforPost(boardID, postID, mediaID string, ownerInfo *peoplerpc.ConciseProfileReply, profileID int) (map[string]interface{}, error)
	AddFile(boardID, postID string, file map[string]interface{}, profileID int) (map[string]interface{}, error)
	UpdateFile(file map[string]interface{}, boardID, postID, thingID string, profileID int) (map[string]interface{}, error)
	DeleteFile(boardID, postID, mediaID string, profileID int) (map[string]interface{}, error)
	FetchFileByName(boardID, fileName string, profileID int) (map[string]interface{}, error)
	GetFileByID(fileID string, profile int) (map[string]interface{}, error)
	FetchFilesByProfile(boardID string, profileID, limit int, publicOnly bool) (map[string]interface{}, error)
}

type service struct {
	config         *config.Config
	mongodb        *mongodatabase.DBConfig
	dbMaster       *database.Database
	dbReplica      *database.Database
	boardService   board.Service
	profileService profile.Service
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
		boardService:   board.NewService(repos, conf),
		profileService: profile.NewService(repos, conf),
		storageService: storage.NewService(repos, conf),
		cache:          repos.Cache,
		peopleRpc:      repos.PeopleGrpcServiceClient,
	}
}

func (s *service) FetchFilesByBoard(boardID string, profileID int, fileType, owner string, tagsArr []string, uploadDate string, limit int, page, l string) (map[string]interface{}, error) {
	return getFilesByBoard(s.mongodb, s.dbMaster, s.cache, s.boardService, s.peopleRpc, s.storageService, boardID, profileID, fileType, owner, tagsArr, uploadDate, limit, page, l)
}

func (s *service) FetchFilesByPost(boardID, postID string, ownerInfo *peoplerpc.ConciseProfileReply, profileID int, fileType, owner string, tagsArr []string, uploadDate string, limit int, page, l string) (map[string]interface{}, error) {
	return getFilesByPost(s.mongodb, s.dbMaster, s.cache, s.boardService, s.peopleRpc, s.storageService, boardID, postID, ownerInfo, profileID, fileType, owner, tagsArr, uploadDate, limit, page, l)
}

func (s *service) FetchFilesByPost2(boardID, postID string) ([]map[string]interface{}, error) {
	return getFilesByPost2(s.mongodb, s.dbMaster, s.cache, s.boardService, s.peopleRpc, s.storageService, boardID, postID)
}

func (s *service) FetchFileByMediaIDforPost(boardID, postID, mediaID string, ownerInfo *peoplerpc.ConciseProfileReply, profileID int) (map[string]interface{}, error) {
	return getFileByMediaIDforPost(s.mongodb, s.dbMaster, s.cache, s.boardService, s.peopleRpc, s.storageService, boardID, postID, mediaID, ownerInfo, profileID)
}

func (s *service) FetchFileByName(boardID, fileName string, profileID int) (map[string]interface{}, error) {
	return getFileByName(s.mongodb, s.cache, boardID, fileName, profileID)
}

func (s *service) AddFile(boardID, postID string, file map[string]interface{}, profileID int) (map[string]interface{}, error) {
	return addFile(s.cache, s.peopleRpc, s.storageService, s.mongodb, boardID, postID, file, profileID)
}

func (s *service) UpdateFile(file map[string]interface{}, boardID, postID, thingID string, profileID int) (map[string]interface{}, error) {
	return updateFile(s.cache, s.mongodb, file, boardID, postID, thingID, profileID)
}

func (s *service) DeleteFile(boardID, postID, mediaID string, profileID int) (map[string]interface{}, error) {
	return deleteFile(s.cache, s.mongodb, boardID, postID, mediaID, profileID)
}

func (s *service) GetFileByID(fileID string, profile int) (map[string]interface{}, error) {
	return getFileByID(s.mongodb, s.boardService, s.peopleRpc, s.storageService, fileID, profile)
}

func (s *service) FetchFilesByProfile(boardID string, profileID, limit int, publicOnly bool) (map[string]interface{}, error) {
	return fetchFilesByProfile(s.cache, s.mongodb, s.dbMaster, boardID, profileID, limit, publicOnly)
}
