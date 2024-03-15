package search

import (

	// "github.com/TestingSDK2/sidekiq-backend/sidekiq-search/app/board"

	model "github.com/TestingSDK2/sidekiq-backend/sidekiq-models"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-search/app/config"
	// "github.com/TestingSDK2/sidekiq-backend/sidekiq-search/app/profile"
	// "github.com/TestingSDK2/sidekiq-backend/sidekiq-search/app/storage"
	contentrpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-content/v1"
	peoplerpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-search/database"
	repo "github.com/TestingSDK2/sidekiq-backend/sidekiq-search/model"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-search/mongodatabase"
)

type Service interface {
	AutoComplete(profileID int, query string) (map[string]interface{}, error)
	FTSOnDashboard(searchFilter *model.GlobalSearchFilter, profileID int, searchKeyword, page, limit, sortBy, orderBy string, isAutoComplete bool) (map[string]interface{}, error)
	AddToSearchHistory(profileID int, query string) (map[string]interface{}, error)
	FetchSearchHistory(profileID int) (map[string]interface{}, error)
	UpdateSearchResults(data map[string]interface{}, updateType string, args ...string) error
}

type service struct {
	config         *config.Config
	mongodb        *mongodatabase.DBConfig
	dbMaster       *database.Database
	dbReplica      *database.Database
	boardService   contentrpc.BoardServiceClient
	profileService peoplerpc.AccountServiceClient
	// storageService storage.Service
}

// NewService - creates new Board service
func NewService(repos *repo.Repos, conf *config.Config) Service {
	return &service{
		config:         conf,
		mongodb:        repos.MongoDB,
		dbMaster:       repos.MasterDB,
		dbReplica:      repos.ReplicaDB,
		boardService:   repos.ContentGrpcServiceClient,
		profileService: repos.PeopleServiceClient,
		// storageService: storage.NewService(repos, conf),
	}
}

func (s *service) AutoComplete(profileID int, query string) (map[string]interface{}, error) {
	// return autoComplete(s.mongodb, s.profileService, s.storageService, profileID, query)
	return autoComplete(s.mongodb, s.profileService, profileID, query)
}

func (s *service) FTSOnDashboard(searchFilter *model.GlobalSearchFilter, profileID int, query, page, limit, sortBy, orderBy string, isAutoComplete bool) (map[string]interface{}, error) {
	return globalFTS(s.mongodb, s.dbMaster, s.boardService, s.profileService, searchFilter, profileID, query, page, limit, sortBy, orderBy)
}

func (s *service) AddToSearchHistory(profileID int, query string) (map[string]interface{}, error) {
	return addToSearchHistory(s.mongodb, profileID, query)
}

func (s *service) FetchSearchHistory(profileID int) (map[string]interface{}, error) {
	return fetchSearchHistory(s.mongodb, profileID)
}

func (s *service) UpdateSearchResults(data map[string]interface{}, updateType string, args ...string) error {
	return updateSearchResults(s.mongodb, data, updateType, args)
}
