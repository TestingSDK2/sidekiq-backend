package model

import (
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-auth-server/cache"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-auth-server/database"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-auth-server/mongodatabase"
	acProtobuf "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"
)

// Repos container to hold handles for cache / db repos
type Repos struct {
	MasterDB             *database.Database
	ReplicaDB            *database.Database
	Cache                *cache.Cache
	MongoDB              *mongodatabase.DBConfig
	AccountServiceClient acProtobuf.AccountServiceClient
}
