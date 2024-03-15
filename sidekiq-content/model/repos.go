package model

import (
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/cache"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/database"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/mongodatabase"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/thingsqs"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-models/model"
	authrpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-auth-server/v1"
	notfrpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-notification/v1"
	peoplerpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"
	searchrpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-search/v1"
)

// Repos container to hold handles for cache / db repos
type Repos struct {
	MasterDB                      *database.Database
	ReplicaDB                     *database.Database
	Cache                         *cache.Cache
	Storage                       model.FileStorage
	TmpStorage                    model.FileStorage
	MongoDB                       *mongodatabase.DBConfig
	ThingSQS                      *thingsqs.SQSConn
	SearchGrpcServiceClient       searchrpc.SearchServiceClient
	PeopleGrpcServiceClient       peoplerpc.AccountServiceClient
	NotificationGrpcServiceClient notfrpc.NotificationServiceClient
	AuthServiceClient             authrpc.AuthServiceClient
}
