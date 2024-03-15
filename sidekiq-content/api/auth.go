package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/ProImaging/sidekiq-backend/sidekiq-content/api/common"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app"
	"github.com/ProImaging/sidekiq-backend/sidekiq-models/model"
	authProtobuf "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-auth-server/v1"
)

func validateUser(config *common.Config, ctx *app.Context, r *http.Request, app *app.App, IsProfileValidate bool) model.AuthResponse {
	token := r.Header.Get(config.AuthCookieName)
	if token == "" {
		c, err := r.Cookie(config.AuthCookieName)
		if c.Value == "" || err != nil {
			return model.AuthResponse{
				User: nil, Profile: 0, ErrCode: http.StatusUnauthorized,
				ErrMsg: "Token is not present", Error: errors.New("token is not present"),
			}
		}
		token = c.Value
	}

	request := authProtobuf.ValidateUserRequest{
		Token:             token,
		ProfileID:         0,
		IsProfileValidate: IsProfileValidate,
	}

	if IsProfileValidate {
		profile := r.Header.Get("Profile")
		var profileID int
		if profile == "" {
			c, err := r.Cookie("Profile")
			if err != nil {
				return model.AuthResponse{
					User: nil, Profile: -1, ErrCode: http.StatusUnauthorized,
					ErrMsg: "Profile is not present", Error: err,
				}
			}
			profile = c.Value
		}
		profileID, _ = strconv.Atoi(profile)
		request.ProfileID = int32(profileID)
	}

	authresp, err := app.Repos.AuthServiceClient.ValidateUser(context.TODO(), &request)
	if err != nil {
		return model.AuthResponse{
			User: nil, Profile: -1, ErrCode: http.StatusUnauthorized,
			ErrMsg: err.Error(), Error: ctx.AuthorizationError(true),
		}
	}

	if authresp.Status != 1 {
		return model.AuthResponse{
			User: nil, Profile: -1, ErrCode: http.StatusUnauthorized,
			ErrMsg: "invalid jwt token", Error: ctx.AuthorizationError(true),
		}
	}

	return model.AuthResponse{User: &model.Account{
		ID:            int(authresp.Data.Id),
		AccountType:   int(authresp.Data.AccountType),
		UserName:      authresp.Data.UserName,
		FirstName:     authresp.Data.FirstName,
		LastName:      authresp.Data.LastName,
		Email:         authresp.Data.Email,
		RecoveryEmail: authresp.Data.RecoveryEmail,
	}, Profile: int(request.ProfileID), ErrCode: 0, ErrMsg: "", Error: nil}
}
