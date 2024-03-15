package grpcservice

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ProImaging/sidekiq-backend/sidekiq-models/model"
	"github.com/ProImaging/sidekiq-backend/sidekiq-people/app"
	acProtobuf "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AccountServer struct {
	acProtobuf.AccountServiceServer
	App *app.App
}

// AuthAccount mehtod is verify account credentials.
func (s *AccountServer) AuthAccount(ctx context.Context, reqdata *acProtobuf.CredentialsRequest) (*acProtobuf.AccountReply, error) {
	req := model.Credentials{}
	req.Email = reqdata.Email
	req.Password = reqdata.Password
	req.UserName = reqdata.UserName

	accountData, err := s.App.AccountService.AuthAccount(&req)
	if err != nil {
		logrus.Errorf("error while auth account: %s", err.Error())
		return nil, err
	}

	return &acProtobuf.AccountReply{
		Id:    int32(accountData.ID),
		Email: accountData.Email,
	}, nil
}

// GetAccountDetails is get account details based on accountID
func (s *AccountServer) GetAccountDetails(ctx context.Context, reqdata *acProtobuf.AccountDetailRequest) (*acProtobuf.AccountDetailReply, error) {
	accountData, err := s.App.AccountService.FetchAccount(int(reqdata.AccountId), true)
	if err != nil {
		logrus.Errorf("error while fetch account: %s", err.Error())
		return nil, err
	}

	return &acProtobuf.AccountDetailReply{
		Data: &acProtobuf.Account{
			ID:          int32(accountData.ID),
			AccountType: int32(accountData.AccountType),
			UserName:    accountData.UserName,
			FirstName:   accountData.FirstName,
			LastName:    accountData.LastName,
			Photo:       accountData.Photo,
			Thumbs: &acProtobuf.Thumbnails{
				Medium:   accountData.Thumbs.Medium,
				Small:    accountData.Thumbs.Small,
				Large:    accountData.Thumbs.Large,
				Icon:     accountData.Thumbs.Icon,
				Original: accountData.Thumbs.Original,
			},
			Email:            accountData.Email,
			RecoveryEmail:    accountData.Email,
			Phone:            accountData.Phone,
			Password:         accountData.Password,
			CreateDate:       convertToProtoTimestamp(accountData.CreateDate),
			LastModifiedDate: convertToProtoTimestamp(accountData.LastModifiedDate),
			Token:            accountData.Token,
			IsActive:         accountData.IsActive,
			ResetToken:       accountData.ResetToken,
			ResetStatus:      accountData.ResetStatus,
		},
	}, nil
}

func convertToProtoTimestamp(t time.Time) *timestamppb.Timestamp {
	return timestamppb.New(t)
}

// ValidateProfile is validate profile with account.
func (s *AccountServer) ValidateProfile(ctx context.Context, reqdata *acProtobuf.ValidateProfileRequest) (*acProtobuf.GenericReply, error) {
	err := s.App.ProfileService.ValidateProfile(int(reqdata.ProfileId), int(reqdata.AccountId))
	if err != nil {
		logrus.Errorf("error while validate profile: %s", err.Error())
		return nil, err
	}

	return &acProtobuf.GenericReply{
		Data:    nil,
		Status:  1,
		Message: "Profile details are verified",
	}, nil
}

func (s *AccountServer) GetConciseProfile(ctx context.Context, reqdata *acProtobuf.ConciseProfileRequest) (*acProtobuf.ConciseProfileReply, error) {

	profiledata, err := s.App.ProfileService.GetConciseProfile(int(reqdata.ProfileId))
	if err != nil {
		logrus.Errorf("error while get profile details: %s", err.Error())
		return nil, err
	}

	return &acProtobuf.ConciseProfileReply{
		Id:                 int32(profiledata.Id),
		AccountID:          int32(profiledata.UserID),
		FirstName:          profiledata.FirstName,
		LastName:           profiledata.LastName,
		Photo:              profiledata.Photo,
		ScreenName:         profiledata.ScreenName,
		UserName:           profiledata.UserName,
		Email:              profiledata.Email,
		Phone:              profiledata.Phone,
		Type:               profiledata.Type,
		Shareable:          profiledata.Shareable,
		DefaultThingsBoard: profiledata.DefaultThingsBoard,
		Thumbs: &acProtobuf.Thumbnails{
			Small:    profiledata.Thumbs.Small,
			Medium:   profiledata.Thumbs.Medium,
			Large:    profiledata.Thumbs.Large,
			Icon:     profiledata.Thumbs.Icon,
			Original: profiledata.Thumbs.Original,
		},
	}, nil
}

func convertDataToBytes(data interface{}) ([]byte, error) {
	return json.Marshal(data)
}
