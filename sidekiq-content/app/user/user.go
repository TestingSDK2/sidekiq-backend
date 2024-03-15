package user

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"io"
	"log"
	"reflect"
	"time"

	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/email"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/storage"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/cache"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/database"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/helper"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/util"
	"github.com/ProImaging/sidekiq-backend/sidekiq-models/model"
	"github.com/ProImaging/sidekiq-backend/sidekiq-models/model/notification"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func getCacheKey(userID int) string {
	return fmt.Sprintf("user:%d", userID)
}

func getSignupCacheKey(userID int) string {
	return fmt.Sprintf("signupuser:%d", userID)
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func EncodeToString(max int) (string, error) {
	table := [...]byte{'1', '2', '3', '4', '5', '6', '7', '8', '9', '0'}
	b := make([]byte, max)
	n, err := io.ReadAtLeast(rand.Reader, b, max)
	if n != max {
		return "", err
	}
	for i := 0; i < len(b); i++ {
		b[i] = table[int(b[i])%len(table)]
	}
	return string(b), nil
}

func createaccount(db *database.Database, user model.AccountSignup) (map[string]interface{}, error) {
	var fetchstmt string
	var countUser, countAccount *int64
	resData := make(map[string]interface{})

	// Check if email already exists
	fetchstmt = "SELECT COUNT(*) AS COUNT FROM `sidekiq-dev`.Account WHERE email = ?"
	err := db.Conn.Get(&countUser, fetchstmt, user.Email)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch email count on account")
	}
	if (*countUser) != 0 {
		return util.SetResponse(nil, 0, "Account already exists with same email. Please use different email"), nil
	}

	// Check if phone number already exists
	fetchstmt = "SELECT COUNT(*) AS COUNT FROM `sidekiq-dev`.Account WHERE phone = ?"
	err = db.Conn.Get(&countUser, fetchstmt, user.Phone)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch phone count on Account")
	}
	if (*countUser) != 0 {
		return util.SetResponse(nil, 0, "Account already exists with same phone number. Please use different phone number"), nil
	}

	fetchstmt = "SELECT COUNT(*) AS COUNT FROM `sidekiq-dev`.AccountSignup WHERE phone = ? AND email = ?"
	err = db.Conn.Get(&countAccount, fetchstmt, user.Phone, user.Email)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch count on AccountSignup")
	}

	// Account exists in SignupUsers but not Users
	if (*countAccount) != 0 {
		u := &model.AccountSignup{}
		if user.Email != "" {
			fetchstmt = "SELECT id FROM `sidekiq-dev`.AccountSignup WHERE email = ? AND phone = ?"
			err := db.Conn.Get(u, fetchstmt, user.Email, user.Phone)
			if err != nil {
				return nil, errors.Wrap(err, "unable to fetch id")
			}
			resData["id"] = int64(u.ID)
			resData["email"] = user.Email
			resData["phone"] = user.Phone
		}
	} else {
		// Insert in SignupUsers
		stmt := "INSERT INTO `sidekiq-dev`.AccountSignup (email, phone) VALUES(:email, :phone)"
		r, err := db.Conn.NamedExec(stmt, user)
		if err != nil {
			return nil, errors.Wrap(err, "unable to insert user")
		}
		resData["id"], err = r.LastInsertId()
		if err != nil {
			return nil, err
		}
		resData["email"] = user.Email
		resData["phone"] = user.Phone
	}
	return util.SetResponse(resData, 1, "Account created successfully"), nil
}

func getVerificationCode(db *database.Database, emailService email.Service, userID int, emailID string) (map[string]interface{}, error) {
	code, err := EncodeToString(6)
	if err != nil {
		return nil, errors.Wrap(err, "unable to generate code")
	}
	var user model.AccountSignup
	user.VerificationCode = code
	user.ID = userID
	stmt := "UPDATE `sidekiq-dev`.AccountSignup SET verificationCode = :verificationCode WHERE id = :id;"
	_, err = db.Conn.NamedExec(stmt, user)
	if err != nil {
		return nil, errors.Wrap(err, "unable to update verification code for user")
	}

	// send email
	email := model.Email{}
	email.Sender = "donotreply@otp.sidekiq.com" // don't hardcode, use default.yaml
	email.Receiver = emailID
	email.Subject = "Please Verify Your Email"
	email.HtmlBody = fmt.Sprintf(`<h3>Hey,<br>
		A sign in attempt requires further verification because we did not recognize your Email. 
		To complete the sign in, enter the verification code on the given Email.<br><br>Verification Code: <b>%s</b></h3>`, code)
	email.TextBody = fmt.Sprintf("Hey. A sign in attempt requires further verification because we did not recognize your Email. To complete the sign in, enter the verification code on the given Email. Verification code: %s", code)
	err = emailService.SendEmail(email)
	if err != nil {
		return nil, errors.Wrap(err, "unable to send email for reset password")
	}

	res := make(map[string]interface{})
	resData := make(map[string]interface{})
	resData["code"] = code
	resData["status"] = 1
	resData["message"] = "Verification code sent successfully"
	res["data"] = resData
	return resData, nil
}

func verifyVerificationCode(db *database.Database, userID int, payload map[string]interface{}) (map[string]interface{}, error) {
	user := &model.AccountSignup{}
	stmt := "SELECT id, verificationCode FROM `sidekiq-dev`.AccountSignup WHERE id = ?;"
	err := db.Conn.Get(user, stmt, userID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch verification code for user")
	}
	if payload["verificationCode"].(string) == "" || user.VerificationCode != payload["verificationCode"].(string) {
		return util.SetResponse(nil, 0, "Verification Code provided is wrong. Please try again!"), nil
	}
	stmt = "UPDATE `sidekiq-dev`.AccountSignup SET verificationCode = '' WHERE id = :id;"
	_, err = db.Conn.NamedExec(stmt, user)
	if err != nil {
		return nil, errors.Wrap(err, "unable to update verification code for user")
	}
	return util.SetResponse(nil, 1, "OTP verified successfully."), nil
}

func setAccountInformation(db *database.Database, user model.Account, userID int) (map[string]interface{}, error) {
	res := make(map[string]interface{})
	resData := make(map[string]interface{})

	fetchstmt := "SELECT COUNT(*) AS COUNT FROM `sidekiq-dev`.Account WHERE userName = ?"
	var count *int64
	uname := user.UserName
	err := db.Conn.Get(&count, fetchstmt, uname)
	if err != nil {
		return nil, err
	}
	fmt.Println(err, "	", *count, "	", uname)
	if (*count) != 0 {
		res["status"] = 0
		res["message"] = "Account associated with this username already exists."
		res["data"] = nil
		return res, nil
	}

	fmt.Println("346:")
	signupuser := &model.AccountSignup{}
	stmt := "SELECT id, email, phone FROM `sidekiq-dev`.AccountSignup WHERE id = ?;"
	err = db.Conn.Get(signupuser, stmt, userID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch user")
	}

	user.Email = signupuser.Email
	user.Phone = signupuser.Phone
	user.AccountType = 1
	user.CreateDate = time.Now()
	stmt = "INSERT INTO `sidekiq-dev`.Account (email, accountType, recoveryEmail, phone, firstName, lastName, password, userName, photo) VALUES(:email, :accountType, :recoveryEmail, :phone, :firstName, :lastName, :password, :userName, :photo)"
	r, err := db.Conn.NamedExec(stmt, user)
	if err != nil {
		return nil, errors.Wrap(err, "unable to insert user")
	}
	id, _ := r.LastInsertId()
	stmt = "DELETE FROM `sidekiq-dev`.AccountSignup WHERE id = :id;"
	_, err = db.Conn.NamedExec(stmt, signupuser)
	if err != nil {
		return nil, err
	}

	user.ID = int(id)
	resData["token"] = ""
	res["data"] = map[string]interface{}{
		"id":        user.ID,
		"userName":  user.UserName,
		"firstName": user.FirstName,
		"lastName":  user.LastName,
		"email":     user.Email,
		"phone":     user.Phone,
	}
	res["status"] = 1
	res["message"] = "Your account has been created successfully."

	fmt.Println("newly created account: ", res["data"].(map[string]interface{})["id"].(int))
	return res, nil
}

func getUserFromDB(db *database.Database, userID int) (*model.Account, error) {
	stmt := "SELECT id, accountType, firstName, lastName, email, password, lastModifiedDate FROM `sidekiq-dev`.Account WHERE id = ?;"
	user := &model.Account{}
	err := db.Conn.Get(user, stmt, userID)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func getBasicUserFromDB(db *database.Database, userID int) (*model.AccountSignup, error) {
	stmt := "SELECT id, email FROM `sidekiq-dev`.AccountSignup WHERE id = ?;"
	user := &model.AccountSignup{}
	err := db.Conn.Get(user, stmt, userID)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func getAccountPermissions(db *database.Database, userID int) []*model.AccountPermimssion {
	stmt := "SELECT l.orgID, a.company, l.owner, l.apiAccess FROM `sidekiq-dev`.OrgProfile a LEFT JOIN `sidekiq-dev`.LinkUserToOrg l ON l.orgID = a.id WHERE l.userID = ?;"
	permissions := []*model.AccountPermimssion{}
	db.Conn.Select(&permissions, stmt, userID)
	return permissions
}

func fetchUserForAuthByEmail(cache *cache.Cache, db *database.Database, email string) (*model.Account, error) {
	user := &model.Account{}
	err := db.Conn.Get(user, "SELECT id, email, password FROM `sidekiq-dev`.Account WHERE email = ?;", email)
	if err != nil {
		return nil, errors.New("incorrect email")
	}
	return user, nil
}

func fetchPushSubscriptions(cache *cache.Cache, db *database.Database, userID int) []*notification.PushSubscription {
	stmt := "SELECT id, profileID, type, endpoint, auth, p256dh, expirationTime, createDate FROM `sidekiq-dev`.PushSubscriptions WHERE profileID = ?"
	subscriptions := []*notification.PushSubscription{}
	db.Conn.Select(&subscriptions, stmt, userID)
	return subscriptions
}

func insertPushSubscriptions(db *database.Database, subscription *notification.PushSubscription) (int, error) {
	stmt := "INSERT INTO `sidekiq-dev`.PushSubscriptions (profileID, type, endpoint, auth, p256dh, expirationTime) VALUES(:profileID,:type,:endpoint,:auth,:p256dh,:expirationTime);"
	r, err := db.Conn.NamedExec(stmt, subscription)
	if err != nil {
		return 0, err
	}
	id, err := r.LastInsertId()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to get LastInsertId")
	}
	return int(id), nil
}

func updatePushSubscriptions(db *database.Database, subscription *notification.PushSubscription) (int, error) {
	stmt := "UPDATE `sidekiq-dev`.PushSubscriptions SET auth=:auth, p256dh=:p256dh, expirationTime=:expirationTime WHERE id = :id;"
	_, err := db.Conn.NamedExec(stmt, subscription)
	if err != nil {
		return 0, err
	}
	return subscription.ID, nil
}

func removePushSubscription(db *database.Database, subscription *notification.PushSubscription) error {
	stmt := "DELETE FROM `sidekiq-dev`.PushSubscriptions WHERE profileID = :profileID AND auth = :auth;"
	_, err := db.Conn.NamedExec(stmt, subscription)
	return err
}

func setAccountType(db *database.Database, storageService storage.Service, payload *model.SetAccountType) (map[string]interface{}, error) {

	var countUser *int64

	fetchstmt := "SELECT COUNT(*) AS COUNT FROM `sidekiq-dev`.Account WHERE id = ?"
	err := db.Conn.Get(&countUser, fetchstmt, payload.AccountId)
	if err != nil {
		return util.SetResponse(nil, 0, "Error in processing request"), err
	}

	if (*countUser) != 0 {
		stmt := "UPDATE `sidekiq-dev`.Account SET accountType=:accountType WHERE id = :id"
		_, err := db.Conn.NamedExec(stmt, payload)
		if err != nil {
			return util.SetResponse(nil, 0, "Error in processing request"), errors.Wrap(err, "Error in updating accountType in DB")
		}

		stmt = `SELECT
			id, accountType, createDate, lastModifiedDate, isActive,
			IFNULL(firstName, "") as firstName,
			IFNULL(lastName, "") as lastName,
			IFNULL(userName, "") as userName,
			IFNULL(email, "") as email,
			IFNULL(phone, "") as phone,
			IFNULL(recoveryEmail, "") as recoveryEmail

			FROM` + "`sidekiq-dev`.Account WHERE id = ?"

		user := model.Account{}
		err = db.Conn.Get(&user, stmt, payload.AccountId)
		if err != nil {
			return util.SetResponse(nil, 0, "Error in processing request"), errors.Wrap(err, "unable to fetch user info")
		}

		stmt = `SELECT id, service, description, fee, profiles FROM` + "`sidekiq-dev`.Services WHERE id = ?"

		accountTypedetails := model.AccountTypes{}
		err = db.Conn.Get(&accountTypedetails, stmt, user.AccountType)
		if err != nil {
			return util.SetResponse(nil, 0, "Error in processing request"), errors.Wrap(err, "unable to fetch account type")
		}

		user.Photo, err = getAccountImage(db, storageService, user.ID, 0)
		if err != nil {
			user.Photo = ""
		}

		user.Thumbs, err = getAccountImageThumb(db, storageService, user.ID)
		if err != nil {
			user.Thumbs = model.Thumbnails{}
		}

		res := make(map[string]interface{})
		res["accountDetails"] = user
		res["accountTypeDetails"] = accountTypedetails
		return util.SetResponse(res, 1, "Account type set successfully"), nil
	}

	return util.SetResponse(nil, 0, "User associated with the id not found"), nil
}

func getAccountImage(mysql *database.Database, storageService storage.Service, accountID, profileID int) (string, error) {
	var err error
	if accountID == 0 {
		stmt := `SELECT accountID FROM` + "`sidekiq-dev`.AccountProfile WHERE id = ?"
		err = mysql.Conn.Get(&accountID, stmt, profileID)
		if err != nil {
			return "", err
		}
	}
	key := util.GetKeyForUserImage(accountID, "")
	fileName := fmt.Sprintf("%d.png", accountID)
	fileData, err := storageService.GetUserFile(key, fileName)
	if err != nil {
		return "", err
	}
	// fmt.Println("photo fetched from wasabi-----------------------------------", fileData.Filename)
	return fileData.Filename, nil
}

func getAccountImageThumb(mysql *database.Database, storageService storage.Service, accountID int) (model.Thumbnails, error) {
	thumbTypes := []string{"sm", "ic"}
	thumbKey := util.GetKeyForUserImage(accountID, "thumbs")
	thumbfileName := fmt.Sprintf("%d.png", accountID)
	thumbs, err := helper.GetThumbnails(storageService, thumbKey, thumbfileName, thumbTypes)
	if err != nil {
		thumbs = model.Thumbnails{}
	}

	return thumbs, nil
}

func verifyLink(db *database.Database, emailService email.Service, token string) (map[string]interface{}, error) {
	response := make(map[string]interface{})
	user := []*model.Account{}

	// check if password already saved using the link
	// fetchstmt := "SELECT * FROM `sidekiq-dev`.Account WHERE resetToken = ?"
	fetchstmt := `SELECT
		id, accountType, createDate, lastModifiedDate, isActive,
		IFNULL(firstName, "") as firstName,
		IFNULL(lastName, "") as lastName,
		IFNULL(userName, "") as userName,
		IFNULL(email, "") as email,
		IFNULL(phone, "") as phone,
		IFNULL(recoveryEmail, "") as recoveryEmail,
		IFNULL(resetStatus, "") as resetStatus,
		IFNULL(resetTime, "") as resetTime

		FROM` + "`sidekiq-dev`.Account WHERE resetToken = ?"

	err := db.Conn.Select(&user, fetchstmt, token)
	if err != nil {
		response = util.SetResponse(nil, 0, "Error in processing request")
		return response, err
	}

	if (len(user)) == 1 {
		// password not saved using this link but link may have expired
		uniqueUser := user[0]
		resetStatus := uniqueUser.ResetStatus
		currentTime := time.Now()
		expireTime := currentTime.Add(-time.Minute * 10)

		resetTime, err := time.Parse("2006-01-02 15:04:05", string(uniqueUser.ResetTime))
		if err != nil {
			response = util.SetResponse(nil, 0, "Error in processing request")
			return response, err
		}

		// check if password not set using this token already and link is valid
		if resetStatus && !expireTime.After(resetTime) {
			// return response to frontend
			response = util.SetResponse(nil, 1, "Link Validation Completed successfully")
		} else if resetStatus {
			// generate uuid for sending email
			uuid := uuid.New().String()

			// db store
			var payload struct {
				Email string `json:"email" db:"email"`
				UUID  string `json:"resetToken" db:"resetToken"`
			}
			payload.UUID = uuid
			payload.Email = uniqueUser.Email
			stmt := "UPDATE `sidekiq-dev`.Account SET resetToken=:resetToken, resetTime = now(), resetStatus = true WHERE email = :email;"
			_, err := db.Conn.NamedExec(stmt, payload)
			if err != nil {
				response = util.SetResponse(nil, 0, "Error in processing request")
				return response, err
			}

			// create reset link
			resetPageLink := "https://staging.sidekiq.com/reset-password/" // from frontend
			link := resetPageLink + uuid

			email := model.Email{}
			email.Receiver = uniqueUser.Email
			email.Header = "Sidekiq: Reset password link verfication"
			email.Subject = "Link to reset password"
			email.HtmlBody = fmt.Sprintf(`<h3>Hey, 
				<br>Please <a href="%s">click here</a> to reset your password. 
				The link will automatically expire after 10 minutes</h3>`, link)
			email.TextBody = fmt.Sprintf(`Hey. Please <a href="%s">click here</a> to reset your password. 
				The link will automatically expire after 10 minutes`, link)
			err = emailService.SendEmail(email)
			if err != nil {
				response = util.SetResponse(nil, 0, "Unable to send reset link on your email")
				return response, err
			}
			response = util.SetResponse(nil, 0, "This link has expired. A new link has been sent on your email")
		}
	} else {
		response = util.SetResponse(nil, 0, "You can't reset password with this link.")
	}
	return response, nil
}

func forgotPassword(db *database.Database, emailService email.Service, recipientEmail string) (map[string]interface{}, error) {
	/* Flow -
	   1. Check if account exists in DB from email
	   2. Generate uuid.
	   3. Store uuid in DB and attach in reset link.
	   4. Send reset link on recipient email.
	   5. Return success response with status 1 and appropriate message
	*/

	response := make(map[string]interface{})
	var countUser *int64

	// check if account exists
	if recipientEmail != "" {
		fetchstmt := "SELECT COUNT(*) AS COUNT FROM `sidekiq-dev`.Account WHERE email = ?"
		err := db.Conn.Get(&countUser, fetchstmt, recipientEmail)
		if err != nil {
			response = util.SetResponse(nil, 0, "Error in processing request")
			return response, err
		}
	} else {
		response = util.SetResponse(nil, 0, "Email missing")
	}

	if (*countUser) != 0 {
		// generate uuid for sending email
		uuid := uuid.New().String()

		// db store
		var payload struct {
			Email string `json:"email" db:"email"`
			UUID  string `json:"resetToken" db:"resetToken"`
		}
		payload.UUID = uuid
		payload.Email = recipientEmail
		stmt := "UPDATE `sidekiq-dev`.Account SET resetToken=:resetToken, resetTime = now(), resetStatus = true WHERE email = :email;"
		_, err := db.Conn.NamedExec(stmt, payload)
		if err != nil {
			response = util.SetResponse(nil, 0, "Error in processing request")
			return response, err
		}

		// create reset link based on env value
		resetPageLink := "https://staging.sidekiq.com/reset-password/" // from frontend
		// resetPageLink := "https://sidekiq.com/reset-password/" // from frontend
		link := resetPageLink + uuid

		email := model.Email{}
		email.Sender = "donotreply@otp.sidekiq.com"
		email.Receiver = recipientEmail
		email.Header = "Sidekiq: Forgot password"
		email.Subject = "Forgot Password"
		email.HtmlBody = fmt.Sprintf(`<h3>Hey, 
			<br>Please <a href="%s">click here</a> to reset your password. 
			The link will automatically expire after 10 minutes</h3>`, link)
		email.TextBody = fmt.Sprintf(`Hey. Please <a href="%s">click here</a> to reset your password. The link will automatically expire after 10 minutes`, link)
		err = emailService.SendEmail(email)
		if err != nil {
			response = util.SetResponse(nil, 0, "Unable to send reset link on your email")
			return response, err
		}

		// return response to frontend
		response = util.SetResponse(nil, 1, "Password reset link sent successfully")
	} else {
		response = util.SetResponse(nil, 0, "Account does not exist for this email")
	}
	return response, nil
}

func resetPassword(db *database.Database, emailService email.Service, payload *model.ResetPassword) (map[string]interface{}, error) {
	/* Flow -
	   1. Check if link expired or password already saved once if not save password based on token.
	   2. Set resetStatus to false once token is saved.
	   3. Return success response with status 1 and appropriate message
	*/

	response := make(map[string]interface{})
	user := []*model.Account{}

	// check if password already saved once
	fetchstmt := "SELECT * FROM `sidekiq-dev`.Account WHERE resetToken = ?"
	err := db.Conn.Select(&user, fetchstmt, payload.ResetToken)
	if err != nil {
		response = util.SetResponse(nil, 0, "Error in processing request")
		return response, err
	}

	if (len(user)) == 1 {
		uniqueUser := user[0]
		resetStatus := uniqueUser.ResetStatus
		currentTime := time.Now()
		expireTime := currentTime.Add(-time.Minute * 10)

		resetTime, err := time.Parse(time.RFC3339, string(uniqueUser.ResetTime))
		if err != nil {
			response = util.SetResponse(nil, 0, "Error in processing request")
			return response, err
		}
		// check reset status (if password is updated using this token) or else it is expired.
		if !resetStatus || expireTime.After(resetTime) {
			// generate uuid for sending email
			uuid := uuid.New().String()

			// db store
			var tokenStructure struct {
				Email string `json:"email" db:"email"`
				UUID  string `json:"resetToken" db:"resetToken"`
			}
			tokenStructure.UUID = uuid
			tokenStructure.Email = uniqueUser.Email
			stmt := "UPDATE `sidekiq-dev`.Account SET resetToken=:resetToken, resetTime = now(), resetStatus = true WHERE email = :email;"
			_, err := db.Conn.NamedExec(stmt, tokenStructure)
			if err != nil {
				response = util.SetResponse(nil, 0, "Error in processing request")
				return response, err
			}

			fmt.Println("called: ", 668)

			// create reset link
			resetPageLink := "http://35.170.215.50/reset-password/" // from frontend
			link := resetPageLink + uuid

			email := model.Email{}
			email.Sender = "donotreply@otp.sidekiq.com"
			email.Receiver = payload.ResetToken
			email.Header = "Sidekiq: reset password"
			email.Subject = "Reset Password"
			email.HtmlBody = fmt.Sprintf(`Hey<br>Please <a href="%s">click here</a> to reset your password. The link will automatically expire after 10 minutes`, link)
			email.TextBody = fmt.Sprintf(`Hey. Please <a href="%s">click here</a> to reset your password. The link will automatically expire after 10 minutes`, link)
			err = emailService.SendEmail(email)
			if err != nil {
				response = util.SetResponse(nil, 0, "Unable to send reset link on your email")
				return response, err
			}

			response = util.SetResponse(nil, 0, "This link has expired. A new link has sent on the given email ID")
		} else {
			// save new password in DB based on resetToken and also set resetStatus to false since password should be set only once from this link.
			stmt := "UPDATE `sidekiq-dev`.Account SET password=:password, resetStatus = false, resetToken = '' WHERE resetToken = :resetToken AND resetStatus = true;"
			_, err = db.Conn.NamedExec(stmt, payload)
			if err != nil {
				fmt.Println("Error in update password query")
				response = util.SetResponse(nil, 0, "Error in processing request")
				return response, err
			}
			fmt.Println("New Password set successfully")

			// return response to frontend
			response = util.SetResponse(nil, 1, "Password reset successful.")
		}
	} else {
		response = util.SetResponse(nil, 0, "Invalid Link. You can't reset password with this link.")
	}
	return response, nil
}

func fetchAccounts(db *database.Database) (map[string]interface{}, error) {
	response := make(map[string]interface{})
	accountTypes := []*model.AccountTypes{}
	stmt := "SELECT id, service, description, fee, profiles FROM `sidekiq-dev`.Services WHERE serviceType = 1;"
	err := db.Conn.Select(&accountTypes, stmt)
	fmt.Println(response)
	if err != nil {
		response = util.SetResponse(nil, 0, "Error from backend request")
		return nil, err
	}
	response = util.SetResponse(accountTypes, 1, "Request Successfully completed")
	return response, nil
}

func fetchAccountInformation(db *database.Database, userID int) (map[string]interface{}, error) {
	var stmt string
	var err error

	stmt = `SELECT
	id, accountType, createDate, lastModifiedDate, isActive,
	IFNULL(firstName, "") as firstName,
	IFNULL(lastName, "") as lastName,
	IFNULL(userName, "") as userName,
	IFNULL(email, "") as email,
	IFNULL(phone, "") as phone,
	IFNULL(recoveryEmail, "") as recoveryEmail

	FROM` + "`sidekiq-dev`.Account WHERE id = ?"

	accountInfo := model.AccountInfoResponse{}
	user := model.Account{}
	err = db.Conn.Get(&user, stmt, userID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch user info")
	}
	accountInfo.AccountInformation = user

	fmt.Println(706, accountInfo)

	// get account type
	var serviceType string
	stmt = "SELECT service FROM `sidekiq-dev`.Services WHERE id = ?"
	err = db.Conn.Get(&serviceType, stmt, user.AccountType)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch user's service type")
	}
	accountInfo.ServiceType = cases.Lower(language.English).String(serviceType)

	// fetch organization account if account type is 3
	orgInfo := &model.Organization{}
	if user.AccountType == 3 {
		stmt := `
		SELECT 
			accountID,
			IFNULL(organizationName, ' ') as organizationName,
			IFNULL(website, ' ') as website,
			IFNULL(registrationNumber, ' ') as registrationNumber,
			IFNULL(email, ' ') as email,
			IFNULL(bio, ' ') as bio,
			IFNULL(city, ' ') as city,
			IFNULL(state, ' ') as state,
			IFNULL(zip, ' ') as zip,
			IFNULL(country, ' ') as country,
			IFNULL(phone, ' ') as phone,
			IFNULL(address1, ' ') as address1,
			IFNULL(address2, ' ') as address2,
			IFNULL(photo, ' ') as photo,
			IFNULL(abv, ' ') as abv,
			IFNULL(mission, ' ') as mission
			FROM` + "`sidekiq-dev`.OrgProfile" +
			` WHERE 
			accountID = ?`

		err = db.Conn.Get(orgInfo, stmt, userID)
		if err == sql.ErrNoRows {
			log.Println("empty data for organization for userID", userID)
			accountInfo.OrganizationInformation = nil
			return util.SetResponse(accountInfo, 1, "Information retrieved"), nil
		} else if err != nil && err != sql.ErrNoRows {
			return nil, errors.Wrap(err, "unable to fetch user's organization info")
		} else if err == nil {
			accountInfo.OrganizationInformation = orgInfo
			return util.SetResponse(accountInfo, 1, "Information retrieved"), nil
		}
	} else {
		accountInfo.OrganizationInformation = nil
		return util.SetResponse(accountInfo, 1, "Information retrieved"), nil
	}

	return util.SetResponse(accountInfo, 1, "Information retrieved"), nil
}

func fetchAccountServices(db *database.Database, user model.Account) (map[string]interface{}, error) {
	accServiceInfo := model.AccountService{}
	stmt := `SELECT 
		s.id,
		s.service, 
		s.description, 
		s.recurring,
		IFNULL(s.image, '') as image, 
		s.fee, s.profiles 
		FROM` + "`sidekiq-dev`.Services s JOIN" + "`sidekiq-dev`.Account u ON s.id=u.accountType WHERE u.id=?"
	err := db.Conn.Get(&accServiceInfo, stmt, user.ID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find account service")
	}

	switch accServiceInfo.Recurring {
	case 2:
		accServiceInfo.ExpiryDate = user.CreateDate.AddDate(0, 1, 0)
	case 3:
		accServiceInfo.ExpiryDate = user.CreateDate.AddDate(1, 0, 0)
	}

	return util.SetResponse(accServiceInfo, 1, "Account service information fetched successfully."), nil
}

func contains(db *database.Database, tableName, dbField, value string) bool {
	fetchstmt := fmt.Sprintf("SELECT COUNT(*) AS COUNT FROM `sidekiq-dev`.%s WHERE %s = ?", tableName, dbField)
	var count int
	err := db.Conn.Get(&count, fetchstmt, value)
	if err != nil {
		fmt.Println("error in count query")
		return true
	}
	if count == 0 {
		return false
	}
	return true
}

func setOrganizationInfo(db *database.Database, payload *model.Organization) (map[string]interface{}, error) {
	var accountType int
	stmt := "SELECT accountType FROM `sidekiq-dev`.Account where id = ?"
	err := db.Conn.Get(&accountType, stmt, payload.AccountID)
	if err != nil {
		return nil, err
	}
	if accountType != 3 {
		return util.SetResponse(nil, 1, "Account type should be of organization"), nil
	}
	fetchstmt := "SELECT COUNT(*) AS COUNT FROM `sidekiq-dev`.OrgProfile WHERE accountID = ?"
	var count int

	err = db.Conn.Get(&count, fetchstmt, payload.AccountID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch count")
	}
	if count > 0 {
		return util.SetResponse(nil, 0, "Organization info already exists"), nil
	}

	// check if org creds already exists
	if contains(db, "OrgProfile", "email", payload.Email) {
		return util.SetResponse(nil, 0, "Email already exists. Please use another email"), nil
	} else if contains(db, "OrgProfile", "registrationNumber", payload.RegistrationNumber) {
		return util.SetResponse(nil, 0, "Registration Number already exists. Please use another Registration Number"), nil
	} else if contains(db, "OrgProfile", "website", payload.Website) {
		return util.SetResponse(nil, 0, "Website address already exists. Please use another Website address"), nil
	} else if contains(db, "OrgProfile", "organizationName", payload.OrganizationName) {
		return util.SetResponse(nil, 0, "Organization name already exists. Please use another Organization name"), nil
	}

	stmt = `INSERT INTO` +
		"`sidekiq-dev`.OrgProfile" + `(
			accountID,
			organizationName,
			website,
			registrationNumber,
			email,
			phone,
			address1,
			address2,
			city,
			state,
			zip,
			country,
			bio
		)
		VALUES
		(
			:accountID,
			:organizationName,
			:website,
			:registrationNumber,
			:email,
			:phone,
			:address1,
			:address2,
			:city,
			:state,
			:zip,
			:country,
			:bio
		);`
	_, err = db.Conn.NamedExec(stmt, payload)
	if err != nil {
		return nil, errors.Wrap(err, "unable to perform insert")
	}
	return util.SetResponse(nil, 1, "Organization informartion set successfully"), nil
}

func updateAccountInfo(db *database.Database, payload model.Account) (map[string]interface{}, error) {
	fetchstmt := "SELECT COUNT(*) AS COUNT FROM `sidekiq-dev`.Account WHERE email = ? AND id != ?"
	var count *int64
	err := db.Conn.Get(&count, fetchstmt, payload.Email, payload.ID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch count for email")
	}

	if (*count) != 0 {
		return util.SetResponse(nil, 0, "Please use another email. This email is already associated with another account"), nil
	}

	fetchstmt = "SELECT COUNT(*) AS COUNT FROM `sidekiq-dev`.Account WHERE userName = ? AND id != ?"
	err = db.Conn.Get(&count, fetchstmt, payload.UserName, payload.ID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch count for username")
	}

	if (*count) != 0 {
		return util.SetResponse(nil, 0, "Please use another userName. This userName is already associated with another account"), nil
	}

	payload.LastModifiedDate = time.Now()
	stmt := "UPDATE `sidekiq-dev`.Account" +
		` SET
				userName = :userName,
				firstName = :firstName,
				lastName = :lastName,
				photo = :photo,
				email = :email,
				recoveryEmail = :recoveryEmail,
				phone = :phone,
				lastModifiedDate = :lastModifiedDate
			WHERE 
				id = :id
			`
	_, err = db.Conn.NamedExec(stmt, payload)
	if err != nil {
		return nil, errors.Wrap(err, "unable to update account info")
	}
	return util.SetResponse(nil, 1, "Account information updated successfully"), nil
}

func verifyPin(db *database.Database, payload map[string]interface{}) (map[string]interface{}, error) {
	errRes := util.SetResponse(nil, 0, "please enter the correct pin")
	if reflect.TypeOf(payload["pin"]).Name() == "string" {
		return errRes, nil
	}

	var p int
	stmt := "SELECT PIN from `sidekiq-dev`.Pin WHERE id = 1"
	err := db.Conn.Get(&p, stmt)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find pin")
	}

	if p != int(payload["pin"].(float64)) {
		return errRes, nil
	}
	return util.SetResponse(nil, 1, "pin verified successfully"), nil
}
