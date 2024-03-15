package usecase

import (
	"context"
	"time"

	"github.com/ProImaging/sidekiq-backend/sidekiq-chat/internal/domain"
	authV1 "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-auth-server/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type authUC struct {
	authGrpc       authV1.AuthServiceClient
	contextTimeout time.Duration
}

func NewAuthUC(authGrpc authV1.AuthServiceClient, timeout time.Duration) domain.AuthUC {
	return authUC{
		authGrpc:       authGrpc,
		contextTimeout: timeout,
	}
}

func (aUC authUC) ValidateUser(c context.Context, token string, profileId int32, shouldValidateProfile bool) (*authV1.ValidateUserReply, error) {
	// return &authV1.ValidateUserReply{
	// 	Data: &authV1.Account{
	// 		Id: 454,
	// 	},
	// }, nil
	req := &authV1.ValidateUserRequest{Token: token, ProfileID: profileId, IsProfileValidate: shouldValidateProfile}
	reply, err := aUC.authGrpc.ValidateUser(c, req)
	logrus.Error(err)
	if err != nil {
		logrus.Error(err)
		return reply, err
	}
	if reply == nil {
		return reply, errors.New("unable to authenticate token")
	}
	return reply, nil
}
