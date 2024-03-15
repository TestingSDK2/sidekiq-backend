package model

import (
	authrpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-auth-server/v1"
	contentrpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-content/v1"
	peoplerpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-search/cache"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-search/database"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-search/mongodatabase"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-search/thingsqs"
)

// Repos container to hold handles for cache / db repos
type Repos struct {
	MasterDB                 *database.Database
	ReplicaDB                *database.Database
	Cache                    *cache.Cache
	Storage                  FileStorage
	TmpStorage               FileStorage
	MongoDB                  *mongodatabase.DBConfig
	ThingSQS                 *thingsqs.SQSConn
	ContentGrpcServiceClient contentrpc.BoardServiceClient
	PeopleServiceClient      peoplerpc.AccountServiceClient
	AuthServiceClient        authrpc.AuthServiceClient
}
