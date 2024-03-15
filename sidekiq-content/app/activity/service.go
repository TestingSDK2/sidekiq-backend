package activity

import (
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/config"
	repo "github.com/TestingSDK2/sidekiq-backend/sidekiq-content/model"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/mongodatabase"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-models/model"
)

type Service interface {
	ListAllThingActivities(thingID string) (map[string]interface{}, error)
	LogThingActivity(activity model.ThingActivity) error
}

type service struct {
	config  *config.Config
	mongodb *mongodatabase.DBConfig
}

// NewService - creates new File service
func NewService(repos *repo.Repos, conf *config.Config) Service {
	return &service{
		config:  conf,
		mongodb: repos.MongoDB,
	}
}

func (s *service) ListAllThingActivities(thingID string) (map[string]interface{}, error) {
	return listAllThingActivities(s.mongodb, thingID)
}
func (s *service) LogThingActivity(activity model.ThingActivity) error {
	return logThingActivity(s.mongodb, activity)
}
