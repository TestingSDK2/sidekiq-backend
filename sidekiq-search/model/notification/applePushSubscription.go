package notification

import (
	"github.com/ProImaging/sidekiq-backend/sidekiq-models/model"
)

// ApplePushSubscription - a push subscriptions
type ApplePushSubscription struct {
	ID             int            `json:"id" db:"id"`
	UserID         int            `json:"profileID" db:"profileID"`
	Type           int            `json:"type" db:"type"`
	DeviceToken    string         `json:"deviceToken" db:"deviceToken"`
	ExpirationTime model.NullTime `json:"expirationTime" db:"expirationTime"`
	CreatedOn      model.NullTime `json:"createDate" db:"createDate"`
}
