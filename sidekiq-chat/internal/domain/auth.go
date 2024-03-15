package domain

import (
	"context"

	authV1 "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-auth-server/v1"
)

type AuthUC interface {
	ValidateUser(c context.Context, token string, profileId int32, shouldValidateProfile bool) (*authV1.ValidateUserReply, error)
}
