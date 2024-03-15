package model

import (
	"github.com/ProImaging/sidekiq-backend/sidekiq-models/model"
	"github.com/ProImaging/sidekiq-backend/sidekiq-people/cache"
	"github.com/ProImaging/sidekiq-backend/sidekiq-people/database"
	"github.com/ProImaging/sidekiq-backend/sidekiq-people/mongodatabase"
	authProtobuf "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-auth-server/v1"
	contentProtobuf "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-content/v1"
	notiProtobuf "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-notification/v1"
)

// Repos container to hold handles for cache / db repos
type Repos struct {
	MasterDB                  *database.Database
	ReplicaDB                 *database.Database
	Cache                     *cache.Cache
	MongoDB                   *mongodatabase.DBConfig
	Storage                   model.FileStorage
	TmpStorage                model.FileStorage
	AuthServiceClient         authProtobuf.AuthServiceClient
	NotificationServiceClient notiProtobuf.NotificationServiceClient
	ContentServiceClient      contentProtobuf.BoardServiceClient
}
