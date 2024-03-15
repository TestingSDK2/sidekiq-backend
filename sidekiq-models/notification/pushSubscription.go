package notification

import (
	"github.com/SherClockHolmes/webpush-go"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-models/model"
)

// PushSubscription - a push subscriptions
type PushSubscription struct {
	ID             int            `json:"id" db:"id"`
	ProfileID      int            `json:"profileID" db:"profileID"`
	Type           int            `json:"type" db:"type"`
	Endpoint       string         `json:"endpoint" db:"endpoint"`
	Auth           string         `json:"auth" db:"auth"`
	P256dh         string         `json:"p256dh" db:"p256dh"`
	ExpirationTime model.NullTime `json:"expirationTime" db:"expirationTime"`
	CreatedOn      model.NullTime `json:"createDate" db:"createDate"`
}

func (p *PushSubscription) FromWebPush(s *webpush.Subscription) {
	p.Endpoint = s.Endpoint
	p.Auth = s.Keys.Auth
	p.P256dh = s.Keys.P256dh
}

func (p *PushSubscription) ToWebPush() *webpush.Subscription {
	s := &webpush.Subscription{
		Endpoint: p.Endpoint,
		Keys: webpush.Keys{
			Auth:   p.Auth,
			P256dh: p.P256dh,
		},
	}
	return s
}
