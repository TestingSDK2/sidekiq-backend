package activity

import (
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/mongodatabase"
	"github.com/ProImaging/sidekiq-backend/sidekiq-models/model"
)

func logThingActivity(db *mongodatabase.DBConfig, activity model.ThingActivity) error {
	// add sqs code
	return nil
}

func listAllThingActivities(db *mongodatabase.DBConfig, thingID string) (map[string]interface{}, error) {
	return nil, nil
}
