package model

import (
	authrpc "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-auth-server/v1"
	contentrpc "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-content/v1"
	peoplerpc "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"
	"github.com/ProImaging/sidekiq-backend/sidekiq-search/cache"
	"github.com/ProImaging/sidekiq-backend/sidekiq-search/database"
	"github.com/ProImaging/sidekiq-backend/sidekiq-search/mongodatabase"
	"github.com/ProImaging/sidekiq-backend/sidekiq-search/thingsqs"
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
