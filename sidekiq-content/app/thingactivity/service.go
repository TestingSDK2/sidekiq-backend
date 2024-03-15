package thingactivity

import (
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/config"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/database"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/model"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/mongodatabase"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/thingsqs"
)

type Service interface {
	PushThingActivityToSQS(msg map[string]interface{}) error
	ListAllThingActivities(thingID, limit, page string) (map[string]interface{}, error)
}

type service struct {
	config    *config.Config
	mongodb   *mongodatabase.DBConfig
	dbMaster  *database.Database
	dbReplica *database.Database
	sqsConn   *thingsqs.SQSConn
}

func NewService(repos *model.Repos, conf *config.Config) Service {
	return &service{
		config:    conf,
		mongodb:   repos.MongoDB,
		dbMaster:  repos.MasterDB,
		dbReplica: repos.ReplicaDB,
		sqsConn:   repos.ThingSQS,
	}
}

func (s *service) PushThingActivityToSQS(msg map[string]interface{}) error {
	return pushThingActivityToSQS(s.sqsConn, msg)
}

func (s *service) ListAllThingActivities(id, limit, page string) (map[string]interface{}, error) {
	return listAllThingActivities(s.mongodb, id, limit, page)
}
