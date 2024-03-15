package notification

import (
	"github.com/ProImaging/sidekiq-backend/sidekiq-models/model"
	"github.com/ProImaging/sidekiq-backend/sidekiq-notification/app/config"
	"github.com/ProImaging/sidekiq-backend/sidekiq-notification/cache"
	"github.com/ProImaging/sidekiq-backend/sidekiq-notification/database"
	repo "github.com/ProImaging/sidekiq-backend/sidekiq-notification/model"
	"github.com/ProImaging/sidekiq-backend/sidekiq-notification/mongodatabase"
	accountProtobuf "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"
	realtimeProtobuf "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-realtime/v1"
)

// Service - defines Profile service
type Service interface {
	MarkNotificationAsRead(notificationId, profileID string) error
	MarkAllNotificationAsRead(profileID string) error
	GetNotificationList(profileID string) ([]model.Notification, error)
	GetNotificationDisplayCount(profileID string) (int64, error)
	NotificationHandler(receiverID []int32, senderId int, thingType, thingID, actionType, message string) error
}

type service struct {
	config                *config.Config
	dbMaster              *database.Database
	dbReplica             *database.Database
	mongodb               *mongodatabase.DBConfig
	cache                 *cache.Cache
	accountServiceClient  accountProtobuf.AccountServiceClient
	realtimeServiceClient realtimeProtobuf.DeliveryServiceClient
}

// NewService - creates new Profile service
func NewService(repos *repo.Repos, conf *config.Config) Service {
	return &service{
		config:                conf,
		mongodb:               repos.MongoDB,
		dbMaster:              repos.MasterDB,
		dbReplica:             repos.ReplicaDB,
		cache:                 repos.Cache,
		accountServiceClient:  repos.PeopleServiceClient,
		realtimeServiceClient: repos.RealtimeServiceClient,
	}
}

func (s *service) NotificationHandler(receiverIDs []int32, senderId int, thingType, thingID, actionType, message string) error {
	return notificationHandler(s.mongodb, s.dbMaster, receiverIDs, senderId, s.accountServiceClient, s.realtimeServiceClient, thingType, thingID, actionType, message)
}

func (s *service) MarkNotificationAsRead(notificationId, profileID string) error {
	return markNotificationAsRead(s.mongodb, s.dbMaster, notificationId, profileID)
}

func (s *service) GetNotificationList(profileID string) ([]model.Notification, error) {
	return getNotificationList(s.mongodb, s.dbMaster, profileID)
}

func (s *service) MarkAllNotificationAsRead(profileID string) error {
	return markAllNotificationAsRead(s.mongodb, s.dbMaster, profileID)
}

func (s *service) GetNotificationDisplayCount(profileID string) (int64, error) {
	return getNotificationDisplayCount(s.mongodb, s.dbMaster, profileID)
}
