package accountapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/ProImaging/sidekiq-backend/sidekiq-models/model"
	"github.com/ProImaging/sidekiq-backend/sidekiq-people/app"
	"github.com/ProImaging/sidekiq-backend/sidekiq-people/helper"
	"github.com/ProImaging/sidekiq-backend/sidekiq-people/util"
	authProtobuf "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-auth-server/v1"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Pong Api
func (a *api) Pong(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	time.Sleep(5 * time.Minute)
	json.NewEncoder(w).Encode(util.SetResponse(nil, 1, "pong"))
	return nil
}

// GetAccounts
func (a *api) FetchAccounts(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	res, err := a.App.AccountService.FetchAccounts()
	if err != nil {
		return err
	}
	json.NewEncoder(w).Encode(res)
	return nil
}

// FetchAccountByID - fetch account by ID
func (a *api) FetchAccountByID(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	user, err := a.App.AccountService.FetchAccount(ctx.User.ID, false)
	if err != nil {
		return err
	}
	user.Token = ctx.User.Token
	user.WriteToJSON(w)
	return nil
}

func (a *api) FetchContacts(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	contacts, err := a.App.AccountService.FetchContacts(ctx.User.ID)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(contacts)
	return nil
}

func (a *api) CreateAccount(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	var payload model.AccountSignup
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return err
	}
	res, err := a.App.AccountService.CreateAccount(payload)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(res)
	return nil
}

func (a *api) GetVerificationCode(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	var payload model.RegistrationUser
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to decode payload json")
	}

	if payload.Type == "email" {
		res, err := a.App.AccountService.GetVerificationCode(payload.ID, payload.Email)
		if err != nil {
			return err
		}
		json.NewEncoder(w).Encode(res)
	}
	return nil
}

func (a *api) VerifyVerificationCode(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	var payload map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return err
	}
	res, err := a.App.AccountService.VerifyVerificationCode(int(payload["id"].(float64)), payload)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(res)
	return nil
}

func (a *api) VerifyLink(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	var payload struct {
		Token string `json:"token" db:"token"`
	}

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return err
	}

	res, err := a.App.AccountService.VerifyLink(payload.Token)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(res)
	return nil
}

// ForgotPassword - Here we send reset password link on the recipient email
func (a *api) ForgotPassword(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	var payload struct {
		Email string `json:"email" db:"email"`
	}

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return err
	}

	res, err := a.App.AccountService.ForgotPassword(payload.Email)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(res)
	return nil
}

// ResetPassword
func (a *api) ResetPassword(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	var payload *model.ResetPassword

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return err
	}

	res, err := a.App.AccountService.ResetPassword(payload)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(res)
	return nil
}

// SetAccountType
func (a *api) SetAccountType(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	var payload *model.SetAccountType
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return err
	}

	payload.AccountId = strconv.Itoa(ctx.User.ID)

	res, err := a.App.AccountService.SetAccountType(payload)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(res)
	return nil
}

func (a *api) FetchAccountServices(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	var err error

	accInfo, err := a.App.AccountService.FetchAccountInformation(ctx.User.ID)
	if err != nil {
		return err
	}

	res, err := a.App.AccountService.FetchAccountServices(accInfo["data"].(model.Account))
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(res)
	return nil
}

func (a *api) VerifyPin(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return errors.Wrap(err, "unable to parse input")
	}

	res, err := a.App.AccountService.VerifyPin(payload)
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(res)
	return nil
}

func (a *api) SetAccountInformation(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	r.ParseMultipartForm(12 << 20)
	err := r.ParseForm()
	if err != nil {
		return errors.Wrap(err, "error parsing form")
	}

	var payload *model.Account
	err = json.Unmarshal([]byte(r.FormValue("accountInfo")), &payload)
	if err != nil {
		return errors.Wrap(err, "error parsing user metadata")
	}

	res, err := a.App.AccountService.SetAccountInformation(*payload, payload.ID)
	if err != nil {
		return errors.Wrap(err, "error in setting account information")
	}

	if res["status"] == 0 {
		json.NewEncoder(w).Encode(res)
		return nil
	}

	payload.ID = res["data"].(map[string]interface{})["id"].(int)

	request := authProtobuf.CreateJWTTokenRequest{
		AccountID: int32(payload.ID),
	}

	tokenReply, err := a.App.Repos.AuthServiceClient.CreateJWTToken(context.TODO(), &request)
	if err != nil {
		return errors.Wrap(err, "error in creating JWT token")
	}

	if tokenReply.Status != 1 {
		return errors.New("error in creating JWT token")
	}

	if res["status"].(int) == 1 {
		res["data"].(map[string]interface{})["token"] = tokenReply.Token
	}

	errctx := a.App.AccountService.DeleteSignupCachedUser(payload.ID)
	if errctx != nil {
		return errors.Wrap(errctx, "error in deleting sign-up cache")
	}

	json.NewEncoder(w).Encode(res)
	return nil
}

func (a *api) FetchAccountInformation(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	res, err := a.App.AccountService.FetchAccountInformation(ctx.User.ID)
	if err != nil {
		json.NewEncoder(w).Encode(res)
		return nil
	}

	key := util.GetKeyForUserImage(ctx.User.ID, "")
	fileName := fmt.Sprintf("%d.png", ctx.User.ID)
	fileData, err := a.App.StorageService.GetUserFile(key, fileName)
	if err == nil {
		if accdetails, ok := res["data"].(model.AccountInfoResponse); ok {
			accdetails.AccountInformation.Photo = fileData.Filename

			thumbKey := util.GetKeyForUserImage(ctx.User.ID, "thumbs")
			thumbfileName := fmt.Sprintf("%v.png", ctx.User.ID)
			thumbs, err := helper.GetThumbnails(a.App.StorageService, thumbKey, thumbfileName, []string{})
			if err != nil {
				thumbs = model.Thumbnails{}
			}

			thumbs.Original = accdetails.AccountInformation.Photo
			accdetails.AccountInformation.Thumbs = thumbs

			if accdetails.OrganizationInformation != nil {

				// Fetch Organization Image
				key := util.GetKeyForOrganizationImage(ctx.User.ID, "")
				fileName := fmt.Sprintf("%v.png", ctx.User.ID)
				fileData, err := a.App.StorageService.GetUserFile(key, fileName)
				if err == nil {
					accdetails.OrganizationInformation.Photo = fileData.Filename
				} else {
					logrus.Error(err, "error in fetching organization image")
				}

				thumbKey := util.GetKeyForOrganizationImage(ctx.User.ID, "thumbs")
				thumbfileName := fmt.Sprintf("%v.png", ctx.User.ID)
				accdetails.OrganizationInformation.Thumbs, err = helper.GetThumbnails(a.App.StorageService, thumbKey, thumbfileName, []string{})
				if err != nil {
					accdetails.OrganizationInformation.Thumbs = model.Thumbnails{}
				}

				accdetails.OrganizationInformation.Thumbs.Original = accdetails.OrganizationInformation.Photo
			}
			res["data"] = accdetails
		}
		json.NewEncoder(w).Encode(res)
		return nil
	} else {
		if accdetails, ok := res["data"].(model.AccountInfoResponse); ok {
			accdetails.AccountInformation.Photo = ""
			if accdetails.OrganizationInformation != nil {

				// Fetch Organization Image
				key := util.GetKeyForOrganizationImage(ctx.User.ID, "")
				fileName := fmt.Sprintf("%v.png", ctx.User.ID)
				fileData, err := a.App.StorageService.GetUserFile(key, fileName)
				if err == nil {
					accdetails.OrganizationInformation.Photo = fileData.Filename
				} else {
					logrus.Error(err, "error in fetching organization image")
				}

				thumbKey := util.GetKeyForOrganizationImage(ctx.User.ID, "thumbs")
				thumbfileName := fmt.Sprintf("%v.png", ctx.User.ID)
				accdetails.OrganizationInformation.Thumbs, err = helper.GetThumbnails(a.App.StorageService, thumbKey, thumbfileName, []string{})
				if err != nil {
					accdetails.OrganizationInformation.Thumbs = model.Thumbnails{}
				}

				accdetails.OrganizationInformation.Thumbs.Original = accdetails.OrganizationInformation.Photo
			}
			res["data"] = accdetails
		}
		json.NewEncoder(w).Encode(res)
		log.Println("error in fetching profile image", err)
		err = nil
	}
	return err
}

func (a *api) UpdateAccountInfo(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	r.ParseMultipartForm(12 << 20)
	err := r.ParseForm()
	if err != nil {
		return errors.Wrap(err, "error parsing form")
	}
	accountData := r.FormValue("accountInfo")
	var payload model.Account
	err = json.Unmarshal([]byte(accountData), &payload)
	if err != nil {
		return errors.Wrap(err, "unable to unmarshall accountData")
	}
	payload.ID = ctx.User.ID
	res, err := a.App.AccountService.UpdateAccountInfo(payload)
	if err != nil {
		return errors.Wrap(err, "unable to update account info")
	}
	if res["status"] == 0 {
		json.NewEncoder(w).Encode(res)
		return nil
	}

	res, err = a.App.AccountService.FetchAccountInformation(ctx.User.ID)
	if err != nil {
		return errors.Wrap(err, "unable to fetch account information")
	}

	key := util.GetKeyForUserImage(ctx.User.ID, "")
	fileName := fmt.Sprintf("%d.png", ctx.User.ID)
	fileData, err := a.App.StorageService.GetUserFile(key, fileName)
	if err == nil {
		if accdetails, ok := res["data"].(model.AccountInfoResponse); ok {
			accdetails.AccountInformation.Photo = fileData.Filename

			thumbKey := util.GetKeyForUserImage(ctx.User.ID, "thumbs")
			thumbfileName := fmt.Sprintf("%v.png", ctx.User.ID)
			thumbs, err := helper.GetThumbnails(a.App.StorageService, thumbKey, thumbfileName, []string{})
			if err != nil {
				thumbs = model.Thumbnails{}
			}

			thumbs.Original = accdetails.AccountInformation.Photo
			accdetails.AccountInformation.Thumbs = thumbs

			if accdetails.OrganizationInformation != nil {

				// Fetch Organization Image
				key := util.GetKeyForOrganizationImage(ctx.User.ID, "")
				fileName := fmt.Sprintf("%v.png", ctx.User.ID)
				fileData, err := a.App.StorageService.GetUserFile(key, fileName)
				if err == nil {
					accdetails.OrganizationInformation.Photo = fileData.Filename
				} else {
					logrus.Error(err, "error in fetching organization image")
				}

				thumbKey := util.GetKeyForOrganizationImage(ctx.User.ID, "thumbs")
				thumbfileName := fmt.Sprintf("%v.png", ctx.User.ID)
				thumbs, err := helper.GetThumbnails(a.App.StorageService, thumbKey, thumbfileName, []string{})
				if err != nil {
					thumbs = model.Thumbnails{}
				}

				thumbs.Original = accdetails.OrganizationInformation.Photo
				accdetails.OrganizationInformation.Thumbs = thumbs

			}
			res["data"] = accdetails
		}

	} else {
		if accdetails, ok := res["data"].(model.AccountInfoResponse); ok {
			accdetails.AccountInformation.Photo = ""
			if accdetails.OrganizationInformation != nil {

				// Fetch Organization Image
				key := util.GetKeyForOrganizationImage(ctx.User.ID, "")
				fileName := fmt.Sprintf("%v.png", ctx.User.ID)
				fileData, err := a.App.StorageService.GetUserFile(key, fileName)
				if err == nil {
					accdetails.OrganizationInformation.Photo = fileData.Filename
				} else {
					logrus.Error(err, "error in fetching organization image")
				}

				thumbKey := util.GetKeyForOrganizationImage(ctx.User.ID, "thumbs")
				thumbfileName := fmt.Sprintf("%v.png", ctx.User.ID)
				thumbs, err := helper.GetThumbnails(a.App.StorageService, thumbKey, thumbfileName, []string{})
				if err != nil {
					thumbs = model.Thumbnails{}
				}

				thumbs.Original = accdetails.OrganizationInformation.Photo
				accdetails.OrganizationInformation.Thumbs = thumbs
			}
			res["data"] = accdetails
		}
	}
	json.NewEncoder(w).Encode(res)
	return nil
}
