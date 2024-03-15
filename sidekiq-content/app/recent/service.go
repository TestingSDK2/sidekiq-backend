package recent

import (
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/board"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/config"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/post"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/profile"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/storage"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/cache"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/database"
	repo "github.com/ProImaging/sidekiq-backend/sidekiq-content/model"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/mongodatabase"
	"github.com/ProImaging/sidekiq-backend/sidekiq-models/model"

	peoplerpc "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"
)

type Service interface {
	AddToDashBoardRecent(thing model.Recent) error
	FetchDashBoardRecentThings(search string, profileID int, sortBy, orderBy string, limitInt, pgInt int, isPagination bool) (map[string]interface{}, error)
	DeleteRecentItems(profileID int, req model.RecentDeletePayload) error
}

type service struct {
	config         *config.Config
	mongodb        *mongodatabase.DBConfig
	dbMaster       *database.Database
	dbReplica      *database.Database
	boardService   board.Service
	profileService profile.Service
	storageService storage.Service
	postService    post.Service
	cache          *cache.Cache
	peopleRpc      peoplerpc.AccountServiceClient
}

func NewService(repos *repo.Repos, conf *config.Config) Service {
	return &service{
		config:         conf,
		mongodb:        repos.MongoDB,
		dbMaster:       repos.MasterDB,
		dbReplica:      repos.ReplicaDB,
		boardService:   board.NewService(repos, conf),
		profileService: profile.NewService(repos, conf),
		storageService: storage.NewService(repos, conf),
		postService:    post.NewService(repos, conf),
		cache:          repos.Cache,
		peopleRpc:      repos.PeopleGrpcServiceClient,
	}
}

func (s *service) AddToDashBoardRecent(thing model.Recent) error {
	return addToDashBoardRecent(s.mongodb, s.dbMaster, thing)
}

func (s *service) FetchDashBoardRecentThings(search string, profileID int, sortBy, orderBy string, limitInt, pgInt int, isPagination bool) (map[string]interface{}, error) {
	return fetchDashBoardRecentThings(s.mongodb, s.cache, s.peopleRpc, s.storageService, s.boardService, s.postService, search, profileID, sortBy, orderBy, limitInt, pgInt, isPagination)
}

func (s *service) DeleteRecentItems(profileID int, req model.RecentDeletePayload) error {
	return deleteRecentItems(s.mongodb, s.dbMaster, req, profileID)
}
