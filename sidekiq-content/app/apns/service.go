package apns

import (
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/cache"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/database"
	repo "github.com/ProImaging/sidekiq-backend/sidekiq-content/model"
	"github.com/ProImaging/sidekiq-backend/sidekiq-models/model"
	"github.com/ProImaging/sidekiq-backend/sidekiq-models/model/notification"
)

// Service defines service for operating on Departments
type Service interface {
	FetchPushSubscriptions(userID int) []*notification.ApplePushSubscription
	CreatePushSubscription(userID int, deviceToken string) (int, error)
	RemovePushSubscription(userID int, deviceToken string) error
	GeneratePushPackage(user *model.Account) (string, error)
}

type service struct {
	dbMaster     *database.Database
	dbReplica    *database.Database
	cache        *cache.Cache
	tmpFileStore model.FileStorage
}

// NewtService create new department Service
func NewService(repos *repo.Repos) Service {
	svc := &service{
		dbMaster:     repos.MasterDB,
		dbReplica:    repos.ReplicaDB,
		cache:        repos.Cache,
		tmpFileStore: repos.TmpStorage,
	}
	return svc
}
