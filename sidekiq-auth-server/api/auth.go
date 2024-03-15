package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/ProImaging/sidekiq-backend/sidekiq-auth-server/app"
	"github.com/ProImaging/sidekiq-backend/sidekiq-auth-server/util"
	"github.com/ProImaging/sidekiq-backend/sidekiq-models/model"
	acProtobuf "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"
)

func (a *API) AuthUser(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	creds := model.CredentialsFromJReader(r.Body)
	if creds == nil {
		return &app.ValidationError{Message: "Invalid Credentials"}
	}
	res := make(map[string]interface{})
	resData := make(map[string]interface{})

	accountReply, err := a.App.Repos.AccountServiceClient.AuthAccount(context.Background(), &acProtobuf.CredentialsRequest{
		Email:    creds.Email,
		UserName: creds.UserName,
		Password: creds.Password,
	})

	if err != nil {
		if err.Error() == "rpc error: code = Unknown desc = incorrect email" {
			json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "The email you've entered is incorrect"))
			return nil
		}

		if err.Error() == "rpc error: code = Unknown desc = incorrect password" {
			json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "The password you've entered is incorrect"))
			return nil
		}

		return err
	}

	jwtToken, err := a.App.JwtService.CreateJWTToken(int(accountReply.Id), a.Config.TokenExpiration, a.App.Config.JWTKey)
	if err != nil {
		json.NewEncoder(w).Encode(util.SetResponse(nil, 0, "Login not successful."))
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:    a.Config.AuthCookieName,
		Value:   jwtToken.Value,
		Expires: jwtToken.ExpiresAt,
	})

	w.WriteHeader(http.StatusOK)
	res["message"] = "Login successful."
	res["status"] = 1

	resData["id"] = strconv.Itoa(int(accountReply.Id))
	resData["email"] = accountReply.Email
	resData["token"] = jwtToken.Value
	resData["Expires"] = jwtToken.ExpiresAt.Format(time.RFC3339)

	res["data"] = resData
	json.NewEncoder(w).Encode(res)
	return nil
}
