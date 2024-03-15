package model

import (
	"github.com/ProImaging/sidekiq-backend/sidekiq-notification/cache"
	"github.com/ProImaging/sidekiq-backend/sidekiq-notification/database"
	"github.com/ProImaging/sidekiq-backend/sidekiq-notification/mongodatabase"
	accountProtobuf "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"
	realtimeProtobuf "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-realtime/v1"
)

// Repos container to hold handles for cache / db repos
type Repos struct {
	MasterDB              *database.Database
	ReplicaDB             *database.Database
	Cache                 *cache.Cache
	MongoDB               *mongodatabase.DBConfig
	PeopleServiceClient   accountProtobuf.AccountServiceClient
	RealtimeServiceClient realtimeProtobuf.DeliveryServiceClient
}
