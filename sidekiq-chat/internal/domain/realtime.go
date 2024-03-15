package domain

import (
	"context"

	realtimeV1 "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-realtime/v1"
)

type RealtimeUC interface {
	DeliverMessageToGroup(c context.Context, message Message, groupId string, action string) (*realtimeV1.DeliveryResponse, error)
}
