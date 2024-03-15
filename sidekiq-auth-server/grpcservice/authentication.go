package grpcservice

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-auth-server/api/common"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-auth-server/app"
	authProtobuf "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-auth-server/v1"
	acProtobuf "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"
	"github.com/golang-jwt/jwt"

	"github.com/sirupsen/logrus"
)

type AuthServer struct {
	authProtobuf.AuthServiceServer
	App    *app.App
	Config *common.Config
}

func (s *AuthServer) ValidateUser(ctx context.Context, req *authProtobuf.ValidateUserRequest) (*authProtobuf.ValidateUserReply, error) {

	if req.Token == "" {
		return nil, errors.New("token is not present")
	}

	claims, err := s.App.JwtService.FetchJWTToken(req.Token)
	if err != nil {
		if errors.Is(err, jwt.ErrSignatureInvalid) {
			logrus.Errorf("err jwt signature is invalid: ", err.Error())
			return nil, errors.New("invalid jwt token")
		}

		logrus.Errorf("error fetching JWT token: %s", err.Error())
		return nil, fmt.Errorf("error fetching JWT token: %s", err.Error())
	}

	if claims.UserID <= 0 {
		logrus.Error("error userID is < 0")
		return nil, errors.New("invalid jwt token: UserID is invalid")
	}

	// Fetch account details concurrently
	var wg sync.WaitGroup
	var accountDetails *acProtobuf.AccountDetailReply
	var accountErr error

	wg.Add(1)
	go func() {
		defer wg.Done()
		cr := acProtobuf.AccountDetailRequest{
			AccountId: int32(claims.UserID),
		}
		accountDetails, accountErr = s.fetchAccountDetails(ctx, &cr)
	}()

	// Validate profile if necessary
	var profileErr error
	if req.IsProfileValidate {
		if req.ProfileID <= 0 {
			logrus.Error("profileID is < 0")
			return nil, errors.New("profile is not present")
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			vr := acProtobuf.ValidateProfileRequest{
				ProfileId: int32(req.ProfileID),
				AccountId: int32(claims.UserID),
			}
			profileErr = s.validateProfile(ctx, &vr)
		}()
	}

	// Wait for goroutines to finish
	wg.Wait()

	// Handle errors
	if accountErr != nil {
		logrus.Errorf("error fetching account details: %s", accountErr.Error())
		return nil, fmt.Errorf("error fetching account details: %s", accountErr.Error())
	}
	if profileErr != nil {
		logrus.Errorf("error validating profile: %s", profileErr.Error())
		return nil, fmt.Errorf("error validating profile: %s", profileErr.Error())
	}

	return &authProtobuf.ValidateUserReply{
		Data: &authProtobuf.Account{
			Id:            accountDetails.Data.ID,
			AccountType:   accountDetails.Data.AccountType,
			UserName:      accountDetails.Data.UserName,
			FirstName:     accountDetails.Data.FirstName,
			LastName:      accountDetails.Data.LastName,
			Email:         accountDetails.Data.Email,
			RecoveryEmail: accountDetails.Data.RecoveryEmail,
		},
		Status:  1,
		Message: "User verified.",
	}, nil
}

func (s *AuthServer) fetchAccountDetails(ctx context.Context, req *acProtobuf.AccountDetailRequest) (*acProtobuf.AccountDetailReply, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	res, err := s.App.Repos.AccountServiceClient.GetAccountDetails(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("error fetching account details from AccountServiceClient: %s", err.Error())
	}
	return res, nil
}

func (s *AuthServer) validateProfile(ctx context.Context, req *acProtobuf.ValidateProfileRequest) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	res, err := s.App.Repos.AccountServiceClient.ValidateProfile(ctx, req)
	if err != nil {
		logrus.Errorf("error validating profile from AccountServiceClient: %s", err)
		return fmt.Errorf("error validating profile from AccountServiceClient: %s", err.Error())
	}
	if res.Status != 1 {
		logrus.Error("profile is invalid")
		return errors.New("profile is invalid")
	}
	return nil
}

func (s *AuthServer) CreateJWTToken(ctx context.Context, req *authProtobuf.CreateJWTTokenRequest) (*authProtobuf.CreateJWTTokenReply, error) {
	jwtToken, err := s.App.JwtService.CreateJWTToken(int(req.AccountID), s.Config.TokenExpiration, s.App.Config.JWTKey)
	if err != nil {
		logrus.Println("Error while creating jwt token ", err)
		return nil, err
	}
	return &authProtobuf.CreateJWTTokenReply{
		Token:  jwtToken.Value,
		Status: 1,
	}, nil
}
