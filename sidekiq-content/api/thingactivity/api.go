package thingactivity

import (
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/api/common"
	// "github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/notification"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/thingactivity"
)

type api struct {
	config *common.Config
	// clientMgr            *notification.ClientManager
	thingActivityService thingactivity.Service
}

func New(conf *common.Config, thingactivity thingactivity.Service) *api {
	return &api{
		config: conf,
		// clientMgr:            clientMgr,
		thingActivityService: thingactivity,
	}
}
