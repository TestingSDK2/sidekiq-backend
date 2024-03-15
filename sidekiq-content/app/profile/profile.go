package profile

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/email"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/storage"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/consts"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/database"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/helper"
	"github.com/jmoiron/sqlx"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/pkg/errors"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/mongodatabase"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/util"
	model "github.com/TestingSDK2/sidekiq-backend/sidekiq-models"
	"github.com/sirupsen/logrus"
	"github.com/skip2/go-qrcode"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func paginate(arr []model.ConciseProfile, pageNo, limit int) (ret []model.ConciseProfile) {
	var startIdx, endIdx int
	startIdx = limit * (pageNo - 1)
	endIdx = limit * pageNo

	if len(arr) == limit || len(arr) < limit {
		return arr
	}
	if endIdx < len(arr) {
		ret = arr[startIdx:endIdx]
	} else {
		ret = arr[startIdx:]
	}
	return
}

func paginateExternalProfile(arr []model.ExternalProfile, pageNo, limit int) (ret []model.ExternalProfile) {
	var startIdx, endIdx int
	startIdx = limit * (pageNo - 1)
	endIdx = limit * pageNo

	if len(arr) == limit || len(arr) < limit {
		return arr
	}
	if endIdx < len(arr) {
		ret = arr[startIdx:endIdx]
	} else {
		ret = arr[startIdx:]
	}
	return
}

func getScreenName(mongodb *mongodatabase.DBConfig, mysql *database.Database, profileID, connProfileID int) (string, error) {
	if profileID == 0 || connProfileID == 0 {
		return "", errors.New("profileID or connProfileID cannot be 0")
	}

	var screenName string
	var mysqlErr error
	stmt := "SELECT screenName from `sidekiq-dev`.AccountProfile where id = ?"

	// return the logged in profile's screenName
	if profileID == connProfileID {
		mysqlErr = mysql.Conn.Get(&screenName, stmt, profileID)
		if mysqlErr != nil {
			return "", errors.Wrap(mysqlErr, "unable to fetch screenName")
		}
	} else {
		// check if the connProfileID is in connection of profileID, if yes, fetch screenName from Connection
		dbconn, err := mongodb.New(consts.Connection)
		if err != nil {
			return "", err
		}
		coll, client := dbconn.Collection, dbconn.Client
		defer client.Disconnect(context.TODO())

		filter := bson.M{"profileID": strconv.Itoa(profileID), "connectionID": strconv.Itoa(connProfileID)}
		opts := options.FindOne().SetProjection(
			bson.M{
				"screenName": 1,
			})

		var value map[string]interface{}
		err = coll.FindOne(context.TODO(), filter, opts).Decode(&value)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				mysqlErr = mysql.Conn.Get(&screenName, stmt, connProfileID)
				if mysqlErr != nil {
					return "", errors.Wrap(mysqlErr, "unable to fetch screenName")
				}
			} else {
				return "", errors.Wrap(err, "unable to fetch the screenName")
			}
		}
		screenName = value["screenName"].(string)
	}

	return screenName, nil
}

func getOrgImage(mysql *database.Database, storageService storage.Service, accountID int) (string, error) {
	key := util.GetKeyForOrganizationImage(accountID, "")
	fileName := fmt.Sprintf("%d.png", accountID)
	fileData, err := storageService.GetUserFile(key, fileName)
	if err != nil {
		fmt.Println("unable to fetch profile image", err)
		return "", nil
	}
	if fileData == nil {
		return "", nil
	}
	return fileData.Filename, nil
}

func getOrgImageThumb(mysql *database.Database, storageService storage.Service, accountID int) (model.Thumbnails, error) {
	thumbTypes := []string{"sm", "ic"}
	thumbKey := util.GetKeyForOrganizationImage(accountID, "thumbs")
	thumbfileName := fmt.Sprintf("%d.png", accountID)
	thumbs, err := helper.GetThumbnails(storageService, thumbKey, thumbfileName, thumbTypes)
	if err != nil {
		thumbs = model.Thumbnails{}
	}

	return thumbs, nil
}

func getProfileImage(mysql *database.Database, storageService storage.Service, accountID, profileID int) (string, error) {
	var err error
	if accountID == 0 {
		stmt := `SELECT accountID FROM` + "`sidekiq-dev`.AccountProfile WHERE id = ?"
		err = mysql.Conn.Get(&accountID, stmt, profileID)
		if err != nil {
			return "", err
		}
	}

	key := util.GetKeyForProfileImage(accountID, profileID, "")
	fileName := fmt.Sprintf("%d.png", profileID)
	fileData, err := storageService.GetUserFile(key, fileName)
	if err != nil {
		fmt.Println("unable to fetch profile image", err)
		return "", nil
	}
	if fileData == nil {
		return "", nil
	}
	return fileData.Filename, nil
}

func getProfileImageThumb(mysql *database.Database, storageService storage.Service, accountID, profileID int) (model.Thumbnails, error) {
	var err error
	if accountID == 0 {
		stmt := `SELECT accountID FROM` + "`sidekiq-dev`.AccountProfile WHERE id = ?"
		err = mysql.Conn.Get(&accountID, stmt, profileID)
		if err != nil {
			return model.Thumbnails{}, err
		}
	}

	thumbTypes := []string{"sm", "ic"}
	thumbKey := util.GetKeyForProfileImage(accountID, profileID, "thumbs")
	thumbfileName := fmt.Sprintf("%d.png", profileID)
	thumbs, err := helper.GetThumbnails(storageService, thumbKey, thumbfileName, thumbTypes)
	if err != nil {
		thumbs = model.Thumbnails{}
	}

	return thumbs, nil
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

func getConciseProfile(mysql *database.Database, id int, storageService storage.Service) (*model.ConciseProfile, error) {
	var stmt string
	var err error
	cp := &model.ConciseProfile{}
	if id != 0 {
		stmt = `SELECT id, accountID, shareable, IFNULL(firstName, '') as firstName, IFNULL(lastName, '') as lastName, 
			IFNULL(screenName, '') AS screenName, 
			IFNULL(photo, '') AS photo FROM` + "`sidekiq-dev`.AccountProfile WHERE id = ?"
		err = mysql.Conn.Get(cp, stmt, id)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, sql.ErrNoRows
			} else {
				return nil, errors.Wrap(err, "unable to find basic info")
			}
		}
		cp.Photo, err = getProfileImage(mysql, storageService, cp.UserID, cp.Id)
		if err != nil {
			fmt.Println(cp.Id, err)
			fmt.Println("unable to find profile image for id", cp.Id, err)
		}

		cp.Thumbs, err = getProfileImageThumb(mysql, storageService, cp.UserID, cp.Id)
		if err != nil {
			fmt.Println(cp.Id, err)
			fmt.Println("unable to find profile thumb image for id", cp.Id, err)
		}

	}
	return cp, nil
}

func ListAllOpenProfiles(db *database.Database) (map[string]interface{}, error) {
	stmt := "select id, IFNULL(photo, '') as photo, firstName, lastName, screenName FROM `sidekiq-dev`.AccountProfile WHERE visibility = ? AND searchable = ? AND isActive = ?"
	var profiles []model.ConciseProfile
	err := db.Conn.Select(&profiles, stmt, "Public", 1, 1)
	if err != nil {
		return nil, err
	}
	return util.SetResponse(profiles, 1, "Candidates for co-managers fetched successfully."), nil
}

func ValidateProfileByUser(db *database.Database, profileID int, accountID int) error {
	stmt := "SELECT id FROM `sidekiq-dev`.AccountProfile WHERE id = ? AND (accountID = ? OR managedByID = ?);"
	var discussions []interface{}
	err := db.Conn.Select(&discussions, stmt, profileID, accountID, accountID)
	if err != nil {
		return err
	}
	if len(discussions) == 0 {
		return errors.New("Profile not authorized")
	}
	return nil
}

func getProfilesWithInfoByUserID(storageService storage.Service, db *database.Database, accountID int) (map[string]interface{}, error) {
	stmt := `SELECT
		id, accountID, firstName, lastName, screenName, phone1, email1, gender, visibility, shareable, searchable,
		showConnections, showBoards, showThingsFollowed, approveGroupMemberships, createDate, modifiedDate, isActive, defaultThingsBoard,
		IFNULL(photo, "") as photo,
		IFNULL(bio, "") as bio,
		IFNULL(address1, "") as address1, 
		IFNULL(address2, "") as address2,
		IFNULL(phone2, "") as phone2,
		IFNULL(email2, "") as email2,
		IFNULL(city, "") as city,
		IFNULL(state, "") as state,
		IFNULL(zip, "") as zip,
		IFNULL(country, "") as country,
		IFNULL(timeZone, "") as timeZone,
		IFNULL(notificationsFromTime, "") as notificationsFromTime,
		IFNULL(notificationsToTime, "") as notificationsToTime,
		IFNULL(managedByID, 0) as managedByID,
		IFNULL(tags, "") as tags,
		IFNULL(sharedInfo, "") as sharedInfo,
		IFNULL(deleteDate, CURRENT_TIMESTAMP) as deleteDate,
		IFNULL(birthday, NOW()) as birthday,
		IFNULL(notes, "") as notes

		FROM` + "`sidekiq-dev`.AccountProfile WHERE accountID = ?"

	var profiles []model.Profile
	err := db.Conn.Select(&profiles, stmt, accountID)
	if err != nil {
		return nil, err
	}
	for i := range profiles {
		profiles[i].Photo, err = getProfileImage(db, storageService, profiles[i].AccountID, profiles[i].ID)
		if err != nil {
			profiles[i].Photo = ""
			fmt.Println("unable to fetch profile photo")
		}

		profiles[i].Thumbs, err = getProfileImageThumb(db, storageService, profiles[i].AccountID, profiles[i].ID)
		if err != nil {
			fmt.Println("unable to fetch profile thumb")
		}
	}
	return util.SetResponse(profiles, 1, "All profiles with info fetched."), nil
}

func getProfilesByUserID(db *database.Database, accountID int, storageService storage.Service) (map[string]interface{}, error) {
	var response map[string]interface{}
	data := make(map[string]interface{})

	// get number of profiles allowed for that user
	getAccStmt := "SELECT s.profiles FROM `sidekiq-dev`.Services s JOIN `sidekiq-dev`.Account u ON s.id=u.accountType WHERE u.id=?"
	var numOfProfilesAllowed *int
	err := db.Conn.Get(&numOfProfilesAllowed, getAccStmt, accountID)
	if err != nil {
		response = util.SetResponse(nil, 0, "Error processing the request.")
		return response, err
	}

	// if number of profiles based on account type is determined then fetch those profiles
	stmt := "SELECT id, firstName, lastName, screenName, defaultThingsBoard FROM `sidekiq-dev`.AccountProfile WHERE accountID = ?;"
	profiles := []*model.ConciseProfile{}
	err = db.Conn.Select(&profiles, stmt, accountID)
	if err != nil {
		response = util.SetResponse(nil, 0, "Could not fetch profiles for this user.")
		return response, err
	}
	if len(profiles) == 0 {
		data["profiles"] = profiles
		data["numOfProfiles"] = len(profiles)
		data["numOfProfilesAllowed"] = *numOfProfilesAllowed
		response = util.SetResponse(data, 1, "This user has no profiles. Please create one.")
		return response, nil
	}

	for i := range profiles {
		profiles[i].Photo, err = getProfileImage(db, storageService, accountID, profiles[i].Id)
		if err != nil {
			profiles[i].Photo = ""
			fmt.Println("unable to fetch profile photo")
		}

		profiles[i].Thumbs, err = getProfileImageThumb(db, storageService, accountID, profiles[i].Id)
		if err != nil {
			profiles[i].Thumbs = model.Thumbnails{}
			fmt.Println("unable to fetch profile photo thumb")
		}
	}

	data["profiles"] = profiles
	data["numOfProfiles"] = len(profiles)
	data["numOfProfilesAllowed"] = *numOfProfilesAllowed

	response = util.SetResponse(data, 1, "Profiles fetched successfully.")
	return response, nil
}

func getProfileCountByUserID(db *database.Database, accountID int) (map[string]interface{}, error) {
	var response map[string]interface{}
	data := make(map[string]interface{})

	// get number of profiles allowed for that user
	getAccStmt := "SELECT s.profiles FROM `sidekiq-dev`.Services s JOIN `sidekiq-dev`.Account u ON s.id=u.accountType WHERE u.id=?"
	var numOfProfilesAllowed *int
	err := db.Conn.Get(&numOfProfilesAllowed, getAccStmt, accountID)
	if err != nil {
		response = util.SetResponse(nil, 0, "Error processing the request.")
		return response, err
	}

	// if number of profiles based on account type is determined then fetch those profiles
	stmt := "SELECT COUNT(*) FROM `sidekiq-dev`.AccountProfile WHERE accountID = ?;"
	var profiles int
	err = db.Conn.Get(&profiles, stmt, accountID)
	if err != nil {
		response = util.SetResponse(nil, 0, "Could not fetch profiles for this user.")
		return response, err
	}
	if profiles == 0 {
		data["numOfProfiles"] = profiles
		data["numOfProfilesAllowed"] = *numOfProfilesAllowed
		response = util.SetResponse(data, 1, "This user has no profiles. Please create one.")
		return response, nil
	}

	data["numOfProfiles"] = profiles
	data["numOfProfilesAllowed"] = *numOfProfilesAllowed

	response = util.SetResponse(data, 1, "Profile count fetched successfully")
	return response, nil
}

func setProfile(db *database.Database, accountID int) (*model.Account, error) {
	// NEW: select the profile based on profile id

	// old
	stmt := "SELECT id, firstName, lastName, email1, password, lastModifiedDate FROM `sidekiq-dev`.Account WHERE id = ?;"
	user := &model.Account{}
	err := db.Conn.Get(user, stmt, accountID)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func addProfile(db *database.Database, profile model.Profile, accountID int) (map[string]interface{}, error) {
	stmt := "SELECT accountType FROM `sidekiq-dev`.Account WHERE id = ?;"
	user := &model.Account{}

	err := db.Conn.Get(user, stmt, accountID)
	if err != nil {
		return nil, err
	}
	var limit int64
	switch user.AccountType {
	case 1:
		limit = 1
	case 2:
		limit = 3
	case 3:
		limit = 100
	}

	var count *int64
	fetchstmt := "SELECT COUNT(*) FROM `sidekiq-dev`.AccountProfile WHERE accountID = ?"
	err = db.Conn.Get(&count, fetchstmt, accountID)
	if err != nil {
		fmt.Println("Error herer", err)
		return nil, err
	}

	fmt.Println((*count), limit)

	if (*count) > limit {
		return nil, consts.ProfileLimitError
	}

	profile.AccountID = accountID

	stmt = `INSERT INTO ` + "`sidekiq-dev`.AccountProfile" +
		` (
				id, accountID, defaultThingsBoard, screenName, 
				firstName, lastName, phone1, phone2, email1, 
				email2, birthday, address1, address2, city, 
				state, zip, country, bio, gender, notes
			) 
			VALUES
			 (
					:id, :accountID, :defaultThingsBoard, :screenName, 
					:firstName, :lastName, :phone1, :phone2, :email1, 
					:email2,:birthday, :address1, :address2, :city, 
					:state, :zip, :country, :bio, :gender, :notes
			)`

	r, err := db.Conn.NamedExec(stmt, profile)
	resData := make(map[string]interface{})
	if err != nil {
		return nil, errors.Wrap(err, "unable to insert profile")
	}
	resData["id"], err = r.LastInsertId()
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch id of inserted profile")
	}

	stmt = "INSERT INTO `sidekiq-dev`.NotificationSettings " +
		`(
			isAllNotifications, isChatMessage, isMention, isInvite, 
			isBoardJoin, isComment, isReaction, profileID) VALUES 
			(
				true, true, true, true, 
				true, true, true, :id
			);`
	_, err = db.Conn.NamedExec(stmt, resData)
	if err != nil {
		return nil, errors.Wrap(err, "unable to insert notification settings")
	}

	stmt = "INSERT INTO `sidekiq-dev`.ShareableSettings " +
		`(
			firstName, lastName, screenName, email, 
			phone, bio, address1, birthday, 
			gender, address2, profileID) VALUES 
			(
				true, true, false, false, 
				false, false, false, false, 
				false, false, :id
			);`
	_, err = db.Conn.NamedExec(stmt, resData)
	if err != nil {
		return nil, errors.Wrap(err, "unable to insert shareable settings")
	}
	return util.SetResponse(resData, 1, "Profile Inserted Successfully"), nil
}

func updateDefaultThingsBoard(db *database.Database, profileID int, defBoardID string) error {
	profile := model.Profile{
		ID:                 profileID,
		DefaultThingsBoard: defBoardID,
	}
	stmt := `UPDATE` + "`sidekiq-dev`.AccountProfile" + `
			SET defaultThingsBoard = :defaultThingsBoard WHERE id = :id`

	_, err := db.Conn.NamedExec(stmt, profile)
	if err != nil {
		return errors.Wrap(err, "error in updating Profiles")
	}

	return nil
}

func editProfile(db *database.Database, profile model.Profile) (map[string]interface{}, error) {
	var count *int64

	fetchstmt := "SELECT COUNT(*) AS COUNT FROM `sidekiq-dev`.AccountProfile WHERE id = ?"
	err := db.Conn.Get(&count, fetchstmt, profile.ID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch count for email")
	}
	if *count == 0 {
		return util.SetResponse(nil, 0, "Profile does not exists. Please create one."), nil
	}
	if profile.ConnectCodeExpiration == "" {
		profile.ConnectCodeExpiration = "1w"
	}
	stmt := `UPDATE` + "`sidekiq-dev`.AccountProfile" + `
			SET
				screenName = :screenName,
				firstName = :firstName,
				lastName = :lastName,
				phone1 = :phone1,
				phone2 = :phone2,
				email1 = :email1,
				email2 = :email2, 
				gender = :gender,
				address1 = :address1, 
				address2 = :address2,
				bio = :bio,
				birthday = :birthday,
				city = :city,
				state = :state,
				zip = :zip,
				country = :country,
				connectCodeExpiration = :connectCodeExpiration
			WHERE
				id = :id`
	_, err = db.Conn.NamedExec(stmt, profile)
	if err != nil {
		return nil, errors.Wrap(err, "error in updating Profiles")
	}

	return util.SetResponse(nil, 1, "Profile Updated Successfully"), nil
}

func deleteProfile(db *database.Database, accountID string, profileIDs []string) (map[string]interface{}, error) {
	deletedAT := time.Now()
	stmt := "UPDATE `sidekiq-dev`.AccountProfile" + `
	SET
	deleteDate = ?
	WHERE id IN (?) and accountID=?
	`
	query, args, err := sqlx.In(stmt, deletedAT, profileIDs, accountID)
	if err != nil {
		return nil, err
	}
	query = db.Conn.Rebind(query)
	result := db.Conn.MustExec(query, args...)
	in, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	fmt.Println("Rows affected", in)
	if in == 0 {
		return util.SetResponse(nil, 1, "profile does not belongs to you"), nil
	}
	return util.SetResponse(nil, 1, "Profiles deleted successfully"), nil
}

func updateProfileSettings(db *database.Database, update model.UpdateProfileSettings, profileID int) (map[string]interface{}, error) {
	var err error
	var stmt string
	if update.UpdateType == "Privacy" {
		update.Profile.ID = profileID

		stmt = `UPDATE ` + "`sidekiq-dev`.AccountProfile" + ` SET 
						managedByID = :managedByID, 
						visibility = :visibility, 
						followMe = :followMe, 
						shareable = :shareable, 
						showConnections = :showConnections, 
						showBoards = :showBoards, 
						notificationsFromTime = :notificationsFromTime, 
						notificationsToTime = :notificationsToTime 
					WHERE id = :id;`

		fmt.Println("update query")
		fmt.Println(stmt)
		fmt.Printf("409: %+v\n", update.Profile)
		_, err = db.Conn.NamedExec(stmt, update.Profile)
	} else if update.UpdateType == "Notification" {
		update.Profile.NotificationSettings.ProfileID = profileID

		stmt = `INSERT INTO ` + "`sidekiq-dev`.NotificationSettings" + `(
						isAllNotifications, isChatMessage, 
						isMention, isInvite, isBoardJoin, 
						isComment, isReaction, profileID
					) 
					VALUES 
						(
							:isAllNotifications, :isChatMessage, 
							:isMention, :isInvite, :isBoardJoin, 
							:isComment, :isReaction, :profileID
						) ON DUPLICATE KEY 
					UPDATE 
						isAllNotifications = :isAllNotifications, 
						isChatMessage = :isChatMessage, 
						isMention = :isMention, 
						isInvite = :isInvite, 
						isBoardJoin = :isBoardJoin, 
						isComment = :isComment, 
						isReaction = :isReaction;`

		_, err = db.Conn.NamedExec(stmt, update.Profile.NotificationSettings)
	} else {
		return util.SetResponse(nil, 0, "Settings type invalid"), nil
	}
	if err != nil {
		return nil, err
	}
	return util.SetResponse(nil, 1, fmt.Sprintf("%s settings updated successfully", update.UpdateType)), nil
}

func getProfileInfo(db *database.Database, profileID int, storageService storage.Service) (map[string]interface{}, error) {
	stmt := `SELECT
		id, accountID, firstName, lastName, screenName, gender, visibility, shareable, searchable, followMe,
		showConnections, showBoards, showThingsFollowed, approveGroupMemberships, createDate, modifiedDate, isActive, defaultThingsBoard,
		IFNULL(screenName, "") as screenName,
		IFNULL(bio, "") as bio,
		IFNULL(email1, "") as email1,
		IFNULL(email2, "") as email2,
		IFNULL(phone1, "") as phone1,
		IFNULL(phone2, "") as phone2,
		IFNULL(address1, "") as address1,
		IFNULL(address2, "") as address2,
		IFNULL(notes, "") as notes,
		IFNULL(city, "") as city,
		IFNULL(state, "") as state,
		IFNULL(zip, "") as zip,
		IFNULL(country, "") as country,
		IFNULL(timeZone, "") as timeZone,
		IFNULL(notificationsFromTime, "") as notificationsFromTime,
		IFNULL(notificationsToTime, "") as notificationsToTime,
		IFNULL(managedByID, 0) as managedByID,
		IFNULL(tags, "") as tags,
		IFNULL(sharedInfo, "") as sharedInfo,
		IFNULL(deleteDate, CURRENT_TIMESTAMP) as deleteDate,
		IFNULL(birthday, "") as birthday,
		IFNULL(connectCodeExpiration, "") as connectCodeExpiration,
		IFNULL(notes, "") as notes

		FROM` + "`sidekiq-dev`.AccountProfile WHERE id = ?"

	profile := model.Profile{}
	err := db.Conn.Get(&profile, stmt, profileID)
	if err != nil {
		return nil, err
	}
	profile.TagsArr = strings.Split(profile.Tags, ",")

	// fetch notification settings for that profile
	notification := model.NotificationSettings{}
	stmt = "SELECT isAllNotifications, isChatMessage, isMention, isInvite, isBoardJoin, isComment, isReaction FROM `sidekiq-dev`.NotificationSettings WHERE profileID = ?"
	err = db.Conn.Get(&notification, stmt, profileID)
	if err == sql.ErrNoRows {
		profile.NotificationSettings = model.NotificationSettings{}
	} else if err != nil {
		return nil, err
	}
	profile.NotificationSettings = notification

	// if co-manager exists
	if profile.ManagedByID != 0 {
		stmt = "SELECT id, firstName, lastName, IFNULL(photo, '') AS photo, email, phone FROM `sidekiq-dev`.Account WHERE id = ?"
		err := db.Conn.Get(&profile.CoManager, stmt, profile.ManagedByID)
		if err == sql.ErrNoRows {
			profile.CoManager = model.ConciseProfile{}
		} else if err != nil {
			return nil, err
		}
	}

	// fetch shareable settings of that profile
	shareableSettings := model.ShareableSettings{}
	shareableStmt := "SELECT firstName, lastName, screenName, email, phone, bio, gender, birthday, address1, address2 FROM `sidekiq-dev`.ShareableSettings WHERE profileID = ?"
	err = db.Conn.Get(&shareableSettings, shareableStmt, profileID)
	if err == sql.ErrNoRows {
		profile.ShareableSettings = model.ShareableSettings{}
	} else if err != nil {
		return nil, err
	}
	profile.ShareableSettings = shareableSettings

	profile.Photo, err = getProfileImage(db, storageService, 0, profileID)
	if err != nil {
		profile.Photo = ""
	}

	profile.Thumbs, err = getProfileImageThumb(db, storageService, 0, profileID)
	if err != nil {
		profile.Thumbs = model.Thumbnails{}
	}

	profile.Thumbs.Original = profile.Photo

	return util.SetResponse(profile, 1, "Profile information fetched successfully."), nil
}

func updateProfileTags(db *database.Database, profile model.Profile, tags []string) error {
	stmt := "SELECT IFNULL(tags, '') as tags FROM `sidekiq-dev`.AccountProfile WHERE id = ?"
	var profileTags string // comma separated string
	err := db.Conn.Get(&profileTags, stmt, profile.ID)
	if err != nil {
		return err
	}

	var profileTagsArr []string

	if len(profileTags) == 0 {
		profile.Tags = strings.Join(tags, ",")
	} else {
		profileTagsArr = strings.Split(profileTags, ",")
		profileTagsArr = append(profileTagsArr, tags...)
		profileTagsArr = util.RemoveArrayDuplicate(profileTagsArr)
		profileTagsStr := strings.Join(profileTagsArr, ",")
		profile.Tags = profileTagsStr
	}

	updateStmt := "UPDATE `sidekiq-dev`.AccountProfile SET tags = :tags WHERE id = :id"
	_, err = db.Conn.NamedExec(updateStmt, profile)
	if err != nil {
		return err
	}
	return nil
}

func fetchTags(db *database.Database, profileID int) (map[string]interface{}, error) {
	stmt := "SELECT tags FROM `sidekiq-dev`.AccountProfile WHERE id = ?"

	var profileTags string
	err := db.Conn.Get(&profileTags, stmt, profileID)
	if err != nil {
		return nil, err
	}

	if len(profileTags) == 0 {
		return util.SetResponse(nil, 1, "Profile has no tags"), nil
	}
	return util.SetResponse(profileTags, 1, "Tags fetched successfully."), nil
}

func fetchProfileConnections(db *mongodatabase.DBConfig, mysql *database.Database, storageService storage.Service, profileID int, limit, page string, searchParameter ...string) (map[string]interface{}, error) {
	profileIDStr := strconv.Itoa(profileID)
	fmt.Println(profileIDStr)
	dbConn, err := db.New(consts.Connection)
	if err != nil {
		return nil, err
	}
	fmt.Println(profileIDStr)
	connCollection, connClient := dbConn.Collection, dbConn.Client
	defer connClient.Disconnect(context.TODO())
	pageNo, err := strconv.Atoi(page)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to int")
	}
	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to int")
	}
	offset := limitInt * (pageNo - 1)

	searchPattern := fmt.Sprintf(".*%s.*", searchParameter[0])
	countPipeline := mongo.Pipeline{
		bson.D{
			{Key: "$match", Value: bson.M{
				"profileID":  profileIDStr,
				"isBlocked":  false,
				"isActive":   true,
				"isArchived": false,
			}},
		},
		bson.D{
			{Key: "$project", Value: bson.M{
				"fullName": bson.M{
					"$concat": bson.A{"$firstName", " ", "$lastName"},
				},
				"firstName":    1,
				"connectionID": 1,
				"lastName":     1,
				"screenName":   1,
				"profileID":    1,
				"relationship": 1,
				"tags":         1,
				"isBlocked":    1,
				"isActive":     1,
				"gender":       1,
			}},
		},
		bson.D{
			{Key: "$match", Value: bson.M{
				"$or": bson.A{
					bson.M{"firstName": primitive.Regex{Pattern: searchPattern, Options: "i"}},
					bson.M{"lastName": primitive.Regex{Pattern: searchPattern, Options: "i"}},
					bson.M{"fullName": primitive.Regex{Pattern: searchPattern, Options: "i"}},
					bson.M{"screenName": primitive.Regex{Pattern: searchPattern, Options: "i"}},
				},
			}},
		},
		bson.D{
			{Key: "$count", Value: "count"},
		},
	}
	pipeline := mongo.Pipeline{
		bson.D{
			{Key: "$match", Value: bson.M{
				"profileID":  profileIDStr,
				"isBlocked":  false,
				"isActive":   true,
				"isArchived": false,
			}},
		},
		bson.D{
			{Key: "$project", Value: bson.M{
				"fullName": bson.M{
					"$concat": bson.A{"$firstName", " ", "$lastName"},
				},
				"firstName":    1,
				"connectionID": 1,
				"lastName":     1,
				"screenName":   1,
				"profileID":    1,
				"relationship": 1,
				"tags":         1,
				"isBlocked":    1,
				"isActive":     1,
				"gender":       1,
			}},
		},
		bson.D{
			{Key: "$match", Value: bson.M{
				"$or": bson.A{
					bson.M{"firstName": primitive.Regex{Pattern: searchPattern, Options: "i"}},
					bson.M{"lastName": primitive.Regex{Pattern: searchPattern, Options: "i"}},
					bson.M{"fullName": primitive.Regex{Pattern: searchPattern, Options: "i"}},
					bson.M{"screenName": primitive.Regex{Pattern: searchPattern, Options: "i"}},
				},
			}},
		},
		bson.D{
			{Key: "$skip", Value: offset},
		},
		bson.D{
			{Key: "$limit", Value: limitInt},
		},
	}
	var result struct {
		Count int `bson:"count"`
	}
	cursor, err := connCollection.Aggregate(context.TODO(), countPipeline)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find count")
	}

	defer cursor.Close(context.TODO())
	if cursor.Next(context.TODO()) {
		err = cursor.Decode(&result)
		if err != nil {
			return nil, errors.Wrap(err, "unable to store in count")
		}
	}
	cursor, err = connCollection.Aggregate(context.Background(), pipeline)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find connections")
	}
	connections := make([]map[string]interface{}, 0)
	err = cursor.All(context.TODO(), &connections)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find profile's connections.")
	}

	for i := range connections {
		profileIDInt, _ := strconv.Atoi(connections[i]["connectionID"].(string))
		connections[i]["photo"], err = getProfileImage(mysql, storageService, 0, profileIDInt)
		if err != nil {
			fmt.Println("error in fetching profile photo")
		}

		connections[i]["thumbs"], err = getProfileImageThumb(mysql, storageService, 0, profileIDInt)
		if err != nil {
			fmt.Println("error in fetching profile thumb photo")
		}

		profileData, err := getSharableDetails(mysql, profileIDInt)
		if err != nil {
			fmt.Println("error in fetching profile details")
			connections[i]["screenName"] = ""
			connections[i]["bio"] = ""
		} else {
			connections[i]["screenName"] = profileData.ScreenName
			connections[i]["bio"] = profileData.Bio
		}

		connections[i]["connectionProfileID"] = connections[i]["connectionID"]
		delete(connections[i], "connectionID")
	}

	if len(connections) == 0 {
		return util.SetPaginationResponse([]int{}, int(result.Count), 1, "No active connections"), nil
	}
	return util.SetPaginationResponse(connections, int(result.Count), 1, "Connections fetched successfully"), nil
}

func getSharableDetails(mysql *database.Database, profileID int) (*model.Profile, error) {
	var profile model.Profile

	shareableStmt := "SELECT firstName, lastName, screenName, email, phone, bio, gender, birthday, address1, address2 FROM `sidekiq-dev`.ShareableSettings WHERE profileID = ?"
	err := mysql.Conn.Get(&profile.ShareableSettings, shareableStmt, profileID)
	if err == sql.ErrNoRows {
		profile.ShareableSettings = model.ShareableSettings{}
	} else if err != nil {
		return nil, err
	}

	if profile.ShareableSettings.Bio {
		Stmt := "SELECT bio FROM `sidekiq-dev`.AccountProfile WHERE id = ?"
		err := mysql.Conn.Get(&profile, Stmt, profileID)
		if err == sql.ErrNoRows {
			profile.Bio = ""
		} else if err != nil {
			return nil, err
		}
	}

	if profile.ShareableSettings.ScreenName {
		Stmt := "SELECT screenName FROM `sidekiq-dev`.AccountProfile WHERE id = ?"
		err := mysql.Conn.Get(&profile, Stmt, profileID)
		if err == sql.ErrNoRows {
			profile.Bio = ""
		} else if err != nil {
			return nil, err
		}
	}

	return &profile, nil
}

func deleteConnection(db *mongodatabase.DBConfig, payload map[string][]string, profileID int) (map[string]interface{}, error) {
	profileIDStr := strconv.Itoa(profileID)
	var filter primitive.M
	dbConn, err := db.New(consts.Connection)
	if err != nil {
		return nil, err
	}
	connCollection, connClient := dbConn.Collection, dbConn.Client
	defer connClient.Disconnect(context.TODO())

	for _, connectionProfileID := range payload["connectionProfileIDs"] {
		filter = bson.M{"profileID": profileIDStr, "connectionID": connectionProfileID}

		_, err = connCollection.DeleteOne(context.TODO(), filter)
		if err != nil {
			return nil, errors.Wrap(err, "unable to delete connection")
		}

		filter = bson.M{"profileID": connectionProfileID, "connectionID": profileIDStr}
		_, err = connCollection.DeleteOne(context.TODO(), filter)
		if err != nil {
			return nil, errors.Wrap(err, "unable to delete connection")
		}
	}

	return util.SetResponse(nil, 1, "Connections removed successfully"), nil
}

func fetchBoardFollowers(db *database.Database, storageService storage.Service, profileID int, limit, page string, searchParameter ...string) (map[string]interface{}, error) {
	var connections []model.FollowersInfo
	var res []model.FollowingInfo

	var err error
	stmt, boardFilter, searchFilter := "", "", ""

	// pagination calculation
	pageInt, err := strconv.Atoi(page)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to int")
	}
	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to int")
	}
	offset := limitInt * (pageInt - 1)

	// adding filters if exists

	// if board ID exists
	if searchParameter[1] != "" {
		boardFilter = fmt.Sprintf("AND (boardID = '%s' OR boardTitle = '%s')", searchParameter[1], searchParameter[1])
	}

	// if any search parameter exists
	if searchParameter[0] != "" {
		searchFilter = `AND
		(
		   CONCAT(firstName, '', lastName) LIKE '%` + searchParameter[0] + `%'
		   OR screenName LIKE '%` + searchParameter[0] + `%'
		)`
	}

	// check count in db
	var count int
	stmt = `SELECT
				COUNT(DISTINCT profileID)
			FROM` +
		"`sidekiq-dev`.AccountProfile as p" + ` 
				INNER JOIN` +
		"`sidekiq-dev`.BoardsFollowed as b" + `
				ON b.profileID = p.id 
			WHERE
				p.id IN 
					(
					SELECT
						profileID 
					FROM` +
		"`sidekiq-dev`.BoardsFollowed" + `
					WHERE
						ownerID = ?  ` + boardFilter + `
					) ` + searchFilter

	err = db.Conn.Get(&count, stmt, profileID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get record's existence")
	}

	if count == 0 {
		return util.SetPaginationResponse(nil, 0, 1, "No board followers found"), nil
	}

	stmt = `SELECT
				b.boardTitle,
				b.boardID,
				p.id as connectionProfileID,
				p.firstName, 
				p.lastName,
				p.screenName 
			FROM` +
		"`sidekiq-dev`.AccountProfile as p" + ` 
				INNER JOIN` +
		"`sidekiq-dev`.BoardsFollowed as b" + `
				ON b.profileID = p.id 
			WHERE
				p.id IN 
					(
					SELECT
						profileID 
					FROM` +
		"`sidekiq-dev`.BoardsFollowed" + `
					WHERE
						ownerID = ?  ` + boardFilter + `
					) ` + searchFilter + ` LIMIT ` + limit + ` OFFSET ` + fmt.Sprintf("%v", offset)

	err = db.Conn.Select(&connections, stmt, profileID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch board followers")
	}
	for i := range connections {
		if len(res) == 0 {
			var temp model.FollowingInfo
			temp.BasicProfileInfo = connections[i].BasicProfileInfo
			temp.BoardDetails = append(temp.BoardDetails, connections[i].BoardInfo)
			res = append(res, temp)
		} else if len(res) > 0 {
			found := false
			for j := range res {
				if res[j].BasicProfileInfo.ID == connections[i].BasicProfileInfo.ID {
					res[j].BoardDetails = append(res[j].BoardDetails, connections[i].BoardInfo)
					found = true
					break
				}
			}
			if !found {
				var temp model.FollowingInfo
				temp.BasicProfileInfo = connections[i].BasicProfileInfo
				temp.BoardDetails = append(temp.BoardDetails, connections[i].BoardInfo)
				res = append(res, temp)
			}
		}
	}

	for i := range res {
		res[i].BasicProfileInfo.Photo, err = getProfileImage(db, storageService, 0, res[i].ID)
		if err != nil {
			fmt.Println("error in fetching profile photo")
		}

		res[i].BasicProfileInfo.Thumbs, err = getProfileImageThumb(db, storageService, 0, res[i].ID)
		if err != nil {
			fmt.Println("error in fetching profile thumbs photo")
		}
	}

	return util.SetPaginationResponse(res, count, 1, "Board followers fetched successfully"), nil
}

func fetchFollowingBoards(db *database.Database, storageService storage.Service, profileID int, limit, page string, searchParameter ...string) (map[string]interface{}, error) {
	var connections []model.FollowersInfo
	var res []model.FollowingInfo

	stmt, boardFilter, searchFilter := "", "", ""

	var err error

	if searchParameter[1] != "" {
		boardFilter = fmt.Sprintf("AND (b.boardID = '%s' OR b.boardTitle = '%s')", searchParameter[1], searchParameter[1])
	}

	// search filter
	if searchParameter[0] != "" {
		searchFilter = `AND
		(
		   CONCAT(firstName, '', lastName) LIKE '%` + searchParameter[0] + `%'
		   OR screenName LIKE '%` + searchParameter[0] + `%'
		)`
	}

	// pagination calculation
	pageInt, err := strconv.Atoi(page)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to int")
	}
	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to int")
	}
	offset := limitInt * (pageInt - 1)

	// check count in db
	var count int
	stmt = `SELECT
				COUNT(DISTINCT  p.id)
			FROM` +
		"`sidekiq-dev`.AccountProfile as p" + `
				INNER JOIN ` +
		"`sidekiq-dev`.BoardsFollowed as b " + boardFilter + `
			WHERE
				p.id IN 
				(
				SELECT
					distinct(ownerID)
			FROM` +
		"`sidekiq-dev`.BoardsFollowed" + `
				WHERE
					profileID = ?
				) AND b.profileID = ? ` + searchFilter

	err = db.Conn.Get(&count, stmt, profileID, profileID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get record's existence")
	}

	if count == 0 {
		return util.SetPaginationResponse(nil, count, 1, "Following boards not found"), nil
	}

	stmt = `SELECT
				b.boardTitle, 
				b.boardID,
				p.id as connectionProfileID,
				p.firstName, 
				p.lastName,
				p.screenName 
			FROM` +
		"`sidekiq-dev`.AccountProfile as p" + `
				INNER JOIN ` +
		"`sidekiq-dev`.BoardsFollowed as b ON b.ownerID = p.id " + boardFilter + `
			WHERE
				p.id IN 
				(
				SELECT
					ownerID
				FROM` +
		"`sidekiq-dev`.BoardsFollowed" + `
				WHERE
					profileID = ?
				) AND b.profileID = ? ` + searchFilter + ` LIMIT ` + limit + ` OFFSET ` + fmt.Sprintf("%v", offset)

	err = db.Conn.Select(&connections, stmt, profileID, profileID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch following boards")
	}

	for i := range connections {
		if len(res) == 0 {
			var temp model.FollowingInfo
			temp.BasicProfileInfo = connections[i].BasicProfileInfo
			temp.BoardDetails = append(temp.BoardDetails, connections[i].BoardInfo)
			res = append(res, temp)
		} else if len(res) > 0 {
			found := false
			for j := range res {
				if res[j].BasicProfileInfo.ID == connections[i].BasicProfileInfo.ID {
					res[j].BoardDetails = append(res[j].BoardDetails, connections[i].BoardInfo)
					found = true
					break
				}
			}
			if !found {
				var temp model.FollowingInfo
				temp.BasicProfileInfo = connections[i].BasicProfileInfo
				temp.BoardDetails = append(temp.BoardDetails, connections[i].BoardInfo)
				res = append(res, temp)
			}
		}
	}

	for i := range res {
		res[i].BasicProfileInfo.Photo, err = getProfileImage(db, storageService, 0, res[i].ID)
		if err != nil {
			fmt.Println("error in fetching profile photo")
		}

		res[i].BasicProfileInfo.Thumbs, err = getProfileImageThumb(db, storageService, 0, res[i].ID)
		if err != nil {
			fmt.Println("error in fetching profile thumb photo")
		}
	}

	return util.SetPaginationResponse(res, count, 1, "Following boards fetched successfully"), nil
}

func fetchConnectionRequests(db *mongodatabase.DBConfig, mysql *database.Database, storageService storage.Service, profileID int, limit, page string) (map[string]interface{}, error) {
	profileIDStr := strconv.Itoa(profileID)
	dbConn, err := db.New(consts.Request)
	if err != nil {
		return nil, err
	}
	collection, client := dbConn.Collection, dbConn.Client
	defer client.Disconnect(context.TODO())

	findOptions := options.Find()
	findOptions.SetSort(bson.M{"createDate": -1})

	var connReq []model.ConnectionRequest
	filter := bson.M{"profileID": profileIDStr}
	cursor, err := collection.Find(context.TODO(), filter, findOptions)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find connections")
	}
	err = cursor.All(context.TODO(), &connReq)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find profile's connections.")
	}
	ll, err := strconv.Atoi(limit)
	if err != nil {
		return nil, errors.Wrap(err, "string to int conversion limit")
	}

	pp, err := strconv.Atoi(page)
	if err != nil {
		return nil, errors.Wrap(err, "string to int conversion page")
	}

	var dd []interface{}
	for _, value := range connReq {
		dd = append(dd, value)
	}

	data := util.PaginateFromArray(dd, pp, ll)

	// Fetch QR from wasabi
	for i := range data {
		// Perform operations on each item as an interface
		if val, ok := data[i].(model.ConnectionRequest); ok {
			profileIDInt, err := strconv.Atoi(val.ProfileID)
			if err != nil {
				return nil, errors.Wrap(err, "unable to convert to int")
			}
			cp, err := getConciseProfile(mysql, profileIDInt, storageService)
			if err != nil {
				return nil, errors.Wrap(err, "unable to fetch concise profile")
			}
			awsKey := util.GetKeyForProfileQR(cp.UserID, profileIDInt)
			fileName := fmt.Sprintf("%s_%s_%s.png", cp.FirstName, cp.LastName, val.Code)
			fileData, err := storageService.GetUserFile(awsKey, fileName)
			if err != nil {
				return nil, errors.Wrap(err, "unable to fetch QR from wasabi")
			}
			val.QR = fileData.Filename
			data[i] = val
		}
	}
	return util.SetPaginationResponse(data, len(connReq), 1, "Connection requests fetched successfully."), nil
}

func fetchArchivedConnections(db *mongodatabase.DBConfig, mysql *database.Database, storageService storage.Service, profileID int, limit, page string, searchParameter ...string) (map[string]interface{}, error) {
	profileIDStr := strconv.Itoa(profileID)
	dbConn, err := db.New("Connection")
	if err != nil {
		return nil, err
	}

	connCollection, connClient := dbConn.Collection, dbConn.Client
	defer connClient.Disconnect(context.TODO())

	pageInt, err := strconv.Atoi(page)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to int")
	}
	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to int")
	}

	offset := limitInt * (pageInt - 1)

	searchPattern := fmt.Sprintf(".*%s.*", searchParameter[0])

	countPipeline := mongo.Pipeline{
		bson.D{
			{Key: "$match", Value: bson.D{
				{Key: "$and", Value: bson.A{
					bson.D{{Key: "profileID", Value: profileIDStr}},
					bson.M{"isArchived": true},
				}},
			}},
		},
		bson.D{
			{Key: "$project", Value: bson.D{
				{Key: "fullName", Value: bson.D{
					{Key: "$concat", Value: bson.A{"$firstName", " ", "$lastName"}},
				}},
			}},
		},
		bson.D{
			{Key: "$match", Value: bson.D{
				{Key: "$or", Value: bson.A{
					bson.D{{Key: "firstName", Value: primitive.Regex{Pattern: searchPattern, Options: "i"}}},
					bson.D{{Key: "lastName", Value: primitive.Regex{Pattern: searchPattern, Options: "i"}}},
					bson.D{{Key: "fullName", Value: primitive.Regex{Pattern: searchPattern, Options: "i"}}},
					bson.D{{Key: "screenName", Value: primitive.Regex{Pattern: searchPattern, Options: "i"}}},
				}},
			}},
		},
		bson.D{
			{Key: "$count", Value: "count"},
		},
	}

	pipeline := mongo.Pipeline{
		bson.D{
			{Key: "$match", Value: bson.M{
				"profileID":  profileIDStr,
				"isArchived": true,
			}},
		},
		bson.D{
			{Key: "$project", Value: bson.M{
				"fullName": bson.M{
					"$concat": bson.A{"$firstName", " ", "$lastName"},
				},
				"firstName":    1,
				"connectionID": 1,
				"lastName":     1,
				"screenName":   1,
			}},
		},
		bson.D{
			{Key: "$match", Value: bson.M{
				"$or": bson.A{
					bson.M{"firstName": primitive.Regex{Pattern: searchPattern, Options: "i"}},
					bson.M{"lastName": primitive.Regex{Pattern: searchPattern, Options: "i"}},
					bson.M{"fullName": primitive.Regex{Pattern: searchPattern, Options: "i"}},
					bson.M{"screenName": primitive.Regex{Pattern: searchPattern, Options: "i"}},
				},
			}},
		},
		bson.D{
			{Key: "$skip", Value: offset},
		},
		bson.D{
			{Key: "$limit", Value: limitInt},
		},
	}

	var result struct {
		Count int `bson:"count"`
	}

	cursor, err := connCollection.Aggregate(context.TODO(), countPipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	if cursor.Next(context.TODO()) {
		err = cursor.Decode(&result)
		if err != nil {
			return nil, err
		}
	}

	cursor, err = connCollection.Aggregate(context.Background(), pipeline)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find archived connections")
	}

	connections := []model.BoardMemberRole{}

	err = cursor.All(context.TODO(), &connections)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find archived connections.")
	}

	for i := range connections {
		profileIDInt, _ := strconv.Atoi(connections[i].ProfileID)
		connections[i].Photo, err = getProfileImage(mysql, storageService, 0, profileIDInt)
		if err != nil {
			fmt.Println("error in fetching profile photo")
		}

		connections[i].Thumbs, err = getProfileImageThumb(mysql, storageService, 0, profileIDInt)
		if err != nil {
			fmt.Println("error in fetching profile thumbs photo")
		}
	}

	return util.SetPaginationResponse(connections, int(result.Count), 1, "Archived fetched successfully"), nil
}

func fetchBlockedConnections(db *mongodatabase.DBConfig, mysql *database.Database, storageService storage.Service, profileID int, limit, page string, searchParameter ...string) (map[string]interface{}, error) {
	profileIDStr := strconv.Itoa(profileID)
	dbConn, err := db.New("Connection")
	if err != nil {
		return nil, err
	}

	connCollection, connClient := dbConn.Collection, dbConn.Client
	defer connClient.Disconnect(context.TODO())

	pageInt, err := strconv.Atoi(page)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to int")
	}
	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to int")
	}

	offset := limitInt * (pageInt - 1)

	searchPattern := fmt.Sprintf(".*%s.*", searchParameter[0])

	countPipeline := mongo.Pipeline{
		bson.D{
			{Key: "$match", Value: bson.D{
				{Key: "$and", Value: bson.A{
					bson.D{{Key: "profileID", Value: profileIDStr}},
					bson.M{"isBlocked": true},
				}},
			}},
		},
		bson.D{
			{Key: "$project", Value: bson.D{
				{Key: "fullName", Value: bson.D{
					{Key: "$concat", Value: bson.A{"$firstName", " ", "$lastName"}},
				}},
			}},
		},
		bson.D{
			{Key: "$match", Value: bson.D{
				{Key: "$or", Value: bson.A{
					bson.D{{Key: "firstName", Value: primitive.Regex{Pattern: searchPattern, Options: "i"}}},
					bson.D{{Key: "lastName", Value: primitive.Regex{Pattern: searchPattern, Options: "i"}}},
					bson.D{{Key: "fullName", Value: primitive.Regex{Pattern: searchPattern, Options: "i"}}},
					bson.D{{Key: "screenName", Value: primitive.Regex{Pattern: searchPattern, Options: "i"}}},
				}},
			}},
		},
		bson.D{
			{Key: "$count", Value: "count"},
		},
	}

	pipeline := mongo.Pipeline{
		bson.D{
			{Key: "$match", Value: bson.M{
				"profileID": profileIDStr,
				"isBlocked": true,
			}},
		},
		bson.D{
			{Key: "$project", Value: bson.M{
				"fullName": bson.M{
					"$concat": bson.A{"$firstName", " ", "$lastName"},
				},
				"firstName":    1,
				"connectionID": 1,
				"lastName":     1,
				"screenName":   1,
			}},
		},
		bson.D{
			{Key: "$match", Value: bson.M{
				"$or": bson.A{
					bson.M{"firstName": primitive.Regex{Pattern: searchPattern, Options: "i"}},
					bson.M{"lastName": primitive.Regex{Pattern: searchPattern, Options: "i"}},
					bson.M{"fullName": primitive.Regex{Pattern: searchPattern, Options: "i"}},
					bson.M{"screenName": primitive.Regex{Pattern: searchPattern, Options: "i"}},
				},
			}},
		},
		bson.D{
			{Key: "$skip", Value: offset},
		},
		bson.D{
			{Key: "$limit", Value: limitInt},
		},
	}

	var result struct {
		Count int `bson:"count"`
	}

	cursor, err := connCollection.Aggregate(context.TODO(), countPipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	if cursor.Next(context.TODO()) {
		err = cursor.Decode(&result)
		if err != nil {
			return nil, err
		}
	}

	cursor, err = connCollection.Aggregate(context.Background(), pipeline)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find archived connections")
	}

	connections := []model.BoardMemberRole{}

	err = cursor.All(context.TODO(), &connections)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find archived connections.")
	}

	for i := range connections {
		profileIDInt, _ := strconv.Atoi(connections[i].ProfileID)
		connections[i].Photo, err = getProfileImage(mysql, storageService, 0, profileIDInt)
		if err != nil {
			fmt.Println("error in fetching profile photo")
		}

		connections[i].Thumbs, err = getProfileImageThumb(mysql, storageService, 0, profileIDInt)
		if err != nil {
			fmt.Println("error in fetching profile thumb photo")
		}
	}

	return util.SetPaginationResponse(connections, int(result.Count), 1, "Profile blocked connections fetched successfully"), nil
}

func sendCoManagerRequest(mysql *database.Database, db *mongodatabase.DBConfig,
	emailService email.Service, profileID int, connReq map[string]interface{},
) (map[string]interface{}, error) {
	var err error
	connReq["profileID"] = strconv.Itoa(profileID)

	dbconn, err := db.New(consts.Request)
	if err != nil {
		return nil, errors.Wrap(err, "unable to establish connection with ConnectionRequest.")
	}
	conn, client := dbconn.Collection, dbconn.Client
	defer client.Disconnect(context.TODO())

	// Get the document
	var connObj model.ConnectionRequest
	reqObjID, _ := primitive.ObjectIDFromHex(connReq["_id"].(string))
	err = conn.FindOne(context.TODO(), bson.M{"_id": reqObjID}).Decode(&connObj)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find the connection request")
	}

	// useless
	// connObj.AssigneeID = connReq["assigneeID"].(string)
	// _, err = conn.UpdateOne(context.TODO(), bson.M{"_id": reqObjID}, bson.M{"$set": connObj})
	// if err != nil {
	// 	return nil, errors.Wrap(err, "unable to update connection request")
	// }

	// send email
	email := model.Email{
		Receiver: connReq["email1"].(string),
		Header:   "Sidekiq: Connection request",
		Subject:  "Connection request from sidekiq",
		TextBody: "Please use one of the following ways to connect",
		HtmlBody: fmt.Sprintf("<h2>%s</h2><br><img src='%s' width='200' height='200' alt='No QR' /><br>", connObj.Code, connObj.QR),
	}
	err = emailService.SendEmail(email)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to send email. Please enter a valid email")
	}

	return util.SetResponse(nil, 1, "Request sent"), nil
}

func acceptCoManagerRequest(mysql *database.Database, db *mongodatabase.DBConfig,
	storageService storage.Service, accountID, profileID int, code string,
) (map[string]interface{}, error) {
	var stmt string
	dbconn, err := db.New(consts.Request)
	if err != nil {
		return nil, err
	}

	connReqColl, connReqClient := dbconn.Collection, dbconn.Client
	defer connReqClient.Disconnect(context.TODO())

	// check if the code has expired or not
	var connReq model.ConnectionRequest
	err = connReqColl.FindOne(context.TODO(), bson.M{"code": code}).Decode(&connReq)
	if err != nil {
		return util.SetResponse(nil, 0, "The code has either expired or incorrect."), nil
	}

	if connReq.AssigneeID == "" {
		return util.SetResponse(nil, 0, "Only co-manager request code will be accepted."), nil
	}

	var senderUserID int

	assigneeProfileIDInt, err := strconv.Atoi(connReq.AssigneeID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to int")
	}
	senderProfileIDInt, err := strconv.Atoi(connReq.ProfileID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to int")
	}

	// get request sender's accountID
	stmt = "SELECT accountID FROM `sidekiq-dev`.AccountProfile WHERE id = ?"
	err = mysql.Conn.Get(&senderUserID, stmt, senderProfileIDInt)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find accountID")
	}

	// check if self connecting
	if profileID == assigneeProfileIDInt || accountID == senderUserID {
		return util.SetResponse(nil, 0, "You cannot self assign"), nil
	}

	// accountID of profileID should set in managedByID of connReq.ProfileID
	stmt = "UPDATE `sidekiq-dev`.AccountProfile SET managedByID =:managedByID where id =:id"

	profileIDInt, err := strconv.Atoi(connReq.AssigneeID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to int")
	}

	p := model.Profile{ID: profileIDInt, ManagedByID: accountID}
	_, err = mysql.Conn.NamedExec(stmt, p)
	if err != nil {
		return nil, errors.Wrap(err, "unable to update co-manager")
	}

	// If organization then create staff profile for co-manager
	var accountType int
	stmt = "SELECT accountType FROM `sidekiq-dev`.Account WHERE id = ?"
	err = mysql.Conn.Get(&accountType, stmt, senderUserID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find accountType")
	}
	fmt.Println("Account type", accountType)
	if accountType == 3 {
		// fetch profileID
		var profileOwnerID int
		stmt = "SELECT id FROM `sidekiq-dev`.AccountProfile WHERE managedByID = ?"
		err = mysql.Conn.Get(&profileOwnerID, stmt, accountID)
		if err != nil {
			return nil, errors.Wrap(err, "err from inserting in staff profile\nunable to find profileID")
		}
		// insert in staff profile
		parameters := map[string]interface{}{
			"id":        accountID,
			"accountID": senderUserID,
		}
		fmt.Println("These are parameters", parameters)
		stmt = "INSERT IGNORE INTO `sidekiq-dev`.OrgStaff" + `
					(accountID,managedByID,photo, firstName, lastName)
				SELECT
				:accountID,id as managedByID,photo, firstName, lastName  
				FROM` + "`sidekiq-dev`.Account" + ` WHERE id = :id;`
		_, err := mysql.Conn.NamedExec(stmt, parameters)
		if err != nil {
			return nil, errors.Wrap(err, "unable to insert in staff profile")
		}

	}

	// remove request from the collection
	_, err = connReqColl.DeleteOne(context.TODO(), bson.M{"code": code})
	if err != nil {
		return nil, errors.Wrap(err, "unable to delete the code.")
	}
	// delete QR from wasabi
	cp, _ := getConciseProfile(mysql, senderProfileIDInt, storageService)
	key := util.GetKeyForProfileQR(senderUserID, senderProfileIDInt)
	fileName := fmt.Sprintf("%s_%s_%s.png", cp.FirstName, cp.LastName, connReq.Code)
	_, err = storageService.DeleteTempMedia(key, fileName)
	if err != nil {
		return nil, errors.Wrap(err, "unable to delete QR code from wasabi")
	}
	return util.SetResponse(nil, 1, "co-manager set successfully."), nil
}

func fetchProfilesWithCoManager(storageService storage.Service, db *database.Database, mongo *mongodatabase.DBConfig, profileID int, profiles []model.Profile, search, page, limit string) (map[string]interface{}, error) {
	var err error

	dbconn, err := mongo.New(consts.Request)
	if err != nil {
		return nil, err
	}
	connReqColl, connReqClient := dbconn.Collection, dbconn.Client
	defer connReqClient.Disconnect(context.TODO())

	var filteredResults, profilesWithComanger, finalResp []model.ProfileWithCoManager

	for _, p := range profiles {
		var reqInfo model.ConnectionRequest
		pwc := model.ProfileWithCoManager{}
		pwc.ScreenName = p.ScreenName
		pwc.ID = p.ID
		pwc.Name = fmt.Sprintf("%s %s", p.FirstName, p.LastName)
		pwc.Photo = p.Photo
		pwc.Thumbs = p.Thumbs

		filter := bson.M{"profileID": strconv.Itoa(profileID), "assigneeID": strconv.Itoa(p.ID)}

		// check if code is sent or not
		count, err := connReqColl.CountDocuments(context.TODO(), filter)
		if err != nil {
			return nil, errors.Wrap(err, "unable to find count")
		}

		if count != 0 {
			err = connReqColl.FindOne(context.TODO(), filter).Decode(&reqInfo)
			if err != nil {
				return nil, err
			}
			pwc.RequestInfo = &reqInfo
		} else {
			pwc.RequestInfo = nil
		}

		if p.ManagedByID != 0 { // if managed by some user
			stmt := `SELECT IFNULL(photo, "") as comanagerPhoto, CONCAT(firstName, " ", lastName) AS comanagerName FROM ` + "`sidekiq-dev`.Account WHERE id = ?"
			err := db.Conn.Get(&pwc, stmt, p.ManagedByID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					pwc.ManagedByID = 0
					continue
				}
				return nil, errors.Wrap(err, "unable to find co-manager")
			}
			pwc.ManagedByID = p.ManagedByID
		}
		profilesWithComanger = append(profilesWithComanger, pwc)
	}

	// search
	if search != "" {
		for _, pwc := range profilesWithComanger {
			// match search in profile name or co-manager name
			if fuzzy.Match(search, pwc.Name) || fuzzy.Match(search, pwc.CoManagerName) ||
				fuzzy.MatchFold(search, pwc.Name) || fuzzy.MatchFold(search, pwc.CoManagerName) {
				filteredResults = append(filteredResults, pwc)
			}
		}
		if len(filteredResults) == 0 {
			return util.SetPaginationResponse([]string{}, 0, 1, "No profiles with co-managers found"), nil
		}
	} else {
		filteredResults = profilesWithComanger
	}

	// pagination
	var pageNo int
	pageNo, _ = strconv.Atoi(page)
	limitInt, _ := strconv.Atoi(limit)
	var data []interface{}
	for _, d := range filteredResults {
		data = append(data, d)
	}

	subset := util.PaginateFromArray(data, pageNo, limitInt)

	for _, d := range subset {
		tmp := d.(model.ProfileWithCoManager)
		// tmp.ComanagerPhoto, err = getAccountImage(db, storageService, tmp.ManagedByID, tmp.ID)
		thumb, err := getAccountImageThumb(db, storageService, tmp.ManagedByID)
		if err != nil {
			tmp.ComanagerPhoto = ""
			fmt.Println("unable to fetch comnager profile photo")
		} else {
			tmp.ComanagerPhoto = thumb.Icon
		}

		finalResp = append(finalResp, tmp)
	}

	return util.SetPaginationResponse(finalResp, len(profilesWithComanger), 1, "Profiles with co-managers fetched successfully"), nil
}

func fetchExternalProfiles(storageService storage.Service, db *database.Database, accountID int, search, page, limit string) (map[string]interface{}, error) {
	var searchFilter string
	if search != "" {
		searchFilter = ` AND
		(
			CONCAT(ap.firstName, ' ', ap.lastName) LIKE '%` + search + `%'
			OR ap.screenName LIKE '%` + search + `%'
			OR ap.firstName LIKE '%` + search + `%'
			OR ap.lastName LIKE '%` + search + `%'
		)`
	}

	managingProfiles := []model.ExternalProfile{}
	stmt := "SELECT ap.id,  ap.accountID,ap.screenName, ap.firstName, ap.lastName,ac.accountType,ac.firstName as accountFirstName,ac.lastName as accountLastName, IFNULL(ap.photo, ' ') as photo FROM `sidekiq-dev`.AccountProfile as ap INNER JOIN `sidekiq-dev`.Account as ac WHERE ap.managedByID = ? and ac.id = ap.AccountID" + searchFilter
	err := db.Conn.Select(&managingProfiles, stmt, accountID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find external profile.")
	}

	if len(managingProfiles) == 0 {
		return util.SetPaginationResponse([]string{}, 0, 1, "You are not managing any profiles."), nil
	}

	checkMap := make(map[int]string)
	for index := range managingProfiles {
		managingProfiles[index].OwnerDetails = model.OwnerDetails{}
		if managingProfiles[index].AccountType == 3 {
			name, ok := checkMap[managingProfiles[index].AccountID]
			if !ok {
				stmt = "select organizationName from `sidekiq-dev`.OrgProfile where accountID = ?"
				db.Conn.QueryRow(stmt, managingProfiles[index].AccountID).Scan(&managingProfiles[index].OwnerDetails.Name)
				checkMap[managingProfiles[index].AccountID] = managingProfiles[index].OwnerDetails.Name
				managingProfiles[index].IsOrganization = true
			} else {
				managingProfiles[index].OwnerDetails.Name = name
				managingProfiles[index].IsOrganization = true
			}
		} else {
			managingProfiles[index].IsPersonal = true
			managingProfiles[index].OwnerDetails.Name = fmt.Sprint(managingProfiles[index].AccountFirstName, " ", managingProfiles[index].AccountLastName)
		}

		managingProfiles[index].OwnerDetails.Photo, err = getProfileImage(db, storageService, managingProfiles[index].AccountID, managingProfiles[index].Id)
		if err != nil {
			managingProfiles[index].Photo = ""
			fmt.Println("unable to fetch profile photo")
		}

		managingProfiles[index].OwnerDetails.Thumbs, err = getProfileImageThumb(db, storageService, managingProfiles[index].AccountID, managingProfiles[index].Id)
		if err != nil {
			managingProfiles[index].Thumbs = model.Thumbnails{}
			fmt.Println("unable to fetch profile photo")
		}
	}

	// pagination
	var pageNo int
	pageNo, _ = strconv.Atoi(page)
	limitInt, _ := strconv.Atoi(limit)
	subset := paginateExternalProfile(managingProfiles, pageNo, limitInt)
	for i := range subset {
		if subset[i].IsPersonal {
			subset[i].Photo, err = getProfileImage(db, storageService, subset[i].AccountID, subset[i].Id)
			if err != nil {
				subset[i].Photo = ""
				fmt.Println("unable to fetch profile photo")
			}

			subset[i].Thumbs, err = getProfileImageThumb(db, storageService, subset[i].AccountID, subset[i].Id)
			if err != nil {
				subset[i].Thumbs = model.Thumbnails{}
				fmt.Println("unable to fetch profile thumb photo")
			}
		} else {
			subset[i].Photo, err = getOrgImage(db, storageService, subset[i].AccountID)
			if err != nil {
				subset[i].Photo = ""
				fmt.Println("unable to fetch org photo")
			}

			subset[i].Thumbs, err = getOrgImageThumb(db, storageService, subset[i].AccountID)
			if err != nil {
				subset[i].Photo = ""
				fmt.Println("unable to fetch org thumb photo")
			}
		}

	}
	return util.SetPaginationResponse(subset, len(managingProfiles), 1, "External profiles fetched successfully."), nil
}

func leaveProfile(db *database.Database, profileToLeave int) (map[string]interface{}, error) {
	profile := model.Profile{
		ID:          profileToLeave,
		ManagedByID: int(model.NullInt64{}.Int64),
	}
	stmt := "UPDATE `sidekiq-dev`.AccountProfile SET managedByID = :managedByID WHERE id = :id"
	_, err := db.Conn.NamedExec(stmt, profile)
	if err != nil {
		return nil, err
	}

	return util.SetResponse(nil, 1, "Profile left successfully."), nil
}

func fetchProfileInfoBasedOffShareableSettings(db *database.Database, profileID string) (*model.Profile, error) {
	stmt := `
        SELECT p.firstName, p.lastName, 
            CASE 
                WHEN s.bio = true AND p.bio IS NOT NULL
                THEN p.bio 
                ELSE '' 
            END as bio,
            CASE
                WHEN s.email = true AND p.email1 IS NOT NULL 
                THEN p.email1 
                ELSE '' 
            END as email1,
            CASE
                WHEN s.screenName = true AND p.screenName IS NOT NULL 
                THEN p.screenName 
                ELSE '' 
            END as screenName,
            CASE
                WHEN s.phone = true AND p.phone1 IS NOT NULL 
                THEN p.phone1 
                ELSE '' 
            END as phone1,
            CASE
                WHEN s.gender = true AND p.gender IS NOT NULL 
                THEN p.gender 
                ELSE 0
            END as gender,
            CASE
                WHEN s.birthday = true AND p.birthday IS NOT NULL 
                THEN p.birthday 
                ELSE ''
            END as birthday,
            CASE
                WHEN s.address1 = true AND p.address1 IS NOT NULL 
                THEN p.address1 
                ELSE ''
            END as address1,
            CASE
                WHEN s.address2 = true AND p.address2 IS NOT NULL 
                THEN p.address2 
                ELSE ''
            END as address2
        FROM ` + "`sidekiq-dev`.AccountProfile as p " +
		"JOIN `sidekiq-dev`.ShareableSettings as s on p.id=s.profileID AND p.id = ?"

	var p model.Profile
	err := db.Conn.Get(&p, stmt, profileID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find shareable info")
	}

	return &p, nil
}

func updateShareableSettings(db *database.Database, mongo *mongodatabase.DBConfig, profileID int,
	shareableSettings model.ShareableSettings) (map[string]interface{}, error) {
	updateStmt := "UPDATE `sidekiq-dev`.ShareableSettings SET " +
		`
		firstName =:firstName, 
		lastName =:lastName, 
		screenName =:screenName, 
		email =:email, 
		phone =:phone, 
		bio =:bio, 
		address1 =:address1, 
		address2 =:address2, 
		birthday =:birthday, 
		gender =:gender

		WHERE profileID = :profileID`

	shareableSettings.ProfileID = profileID
	fmt.Println(1792, shareableSettings)
	_, err := db.Conn.NamedExec(updateStmt, shareableSettings)
	if err != nil {
		return nil, errors.Wrap(err, "unable to update shareable-settings")
	}

	profile, err := fetchProfileInfoBasedOffShareableSettings(db, strconv.Itoa(profileID))
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch profile info based shareable off settings.")
	}

	dbconn, err := mongo.New(consts.Connection)
	if err != nil {
		return nil, errors.Wrap(err, "unable to establish connection with Connection.")
	}
	conn, client := dbconn.Collection, dbconn.Client
	defer client.Disconnect(context.TODO())

	var existingConns []model.Connection
	filter := bson.M{"connectionID": strconv.Itoa(profileID)}

	cur, err := conn.Find(context.TODO(), filter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch existing connections.")
	}

	err = cur.All(context.TODO(), &existingConns)
	if err != nil {
		return nil, errors.Wrap(err, "unable to unpack into existing connections.")
	}

	for _, ec := range existingConns {
		if ec.ScreenName == "" && profile.ScreenName != "" {
			ec.ScreenName = profile.ScreenName
		}
		if ec.Bio == "" && profile.Bio != "" {
			ec.Bio = profile.Bio
		}
		if ec.Birthday == "" && profile.Birthday != "" {
			ec.Birthday = profile.Birthday
		}
		if ec.Address1 == "" && profile.Address1 != "" {
			ec.Address1 = profile.Address1
		}
		if ec.City == "" && profile.City != "" {
			ec.City = profile.City
		}
		if ec.Country == "" && profile.Country != "" {
			ec.Country = profile.Country
		}
		if ec.Email1 == "" && profile.Email1 != "" {
			ec.Email1 = profile.Email1
		}
		if ec.Gender == 0 && profile.Gender != 0 {
			ec.Gender = profile.Gender
		}
		if ec.Zip == "" && profile.Zip != "" {
			ec.Zip = profile.Zip
		}

		// save to mongo
		_, err = conn.UpdateOne(context.TODO(),
			bson.M{"connectionID": ec.ConnectionProfileID, "profileID": ec.ProfileID},
			bson.M{"$set": ec})
		if err != nil {
			return nil, errors.Wrap(err, "unable to update Connection record")
		}
	}

	return util.SetResponse(nil, 1, "Shareable settings updated successfully."), nil
}

func generateCode(db *mongodatabase.DBConfig, mysql *database.Database, storageService storage.Service, accountID,
	CallerprofileID int, payload model.ConnectionRequest,
) (map[string]interface{}, error) {
	var err error
	// this temp profile ID is toggling the actual profile ID in case of
	// generating code on behalf of someone
	var tempProfileID int
	payload.ID = primitive.NewObjectID()

	if payload.ProfileID != "" {
		profileInt, err := strconv.Atoi(payload.ProfileID)
		if err != nil {
			return nil, errors.Wrap(err, "str to int conversion failed on profile id code generate")
		}
		tempProfileID = profileInt
	} else {
		payload.ProfileID = strconv.Itoa(CallerprofileID)
		tempProfileID = CallerprofileID
	}

	payload.Code = util.Get8DigitCode()
	var expTime time.Time

	var codeExpirationTime string

	stmt := "SELECT connectCodeExpiration from `sidekiq-dev`.AccountProfile where id = ?"
	err = mysql.Conn.Get(&codeExpirationTime, stmt, tempProfileID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find accountID")
	}
	payload.Duration = codeExpirationTime
	payload.CreateDate = time.Now()

	switch codeExpirationTime {
	case "1d":
		expTime = payload.CreateDate.AddDate(0, 0, 1)
	case "1h":
		expTime = payload.CreateDate.Add(time.Hour * 1)
	case "1w":
		expTime = payload.CreateDate.AddDate(0, 0, 7)
	case "1m":
		expTime = payload.CreateDate.AddDate(0, 1, 0)
	}
	payload.ExpiryDate = expTime

	cp, _ := getConciseProfile(mysql, tempProfileID, storageService)
	fileName := fmt.Sprintf("%s_%s_%s.png", cp.FirstName, cp.LastName, payload.Code)
	localQrPath := fmt.Sprintf("./%s", fileName)

	// generate QR CODE
	err = qrcode.WriteFile(payload.Code, qrcode.Medium, 256, localQrPath)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create QR code")
	}

	qrFile, err := os.Open(localQrPath)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read QR code")
	}
	defer qrFile.Close()

	qrFileStat, _ := os.Stat(localQrPath)
	awsKey := util.GetKeyForProfileQR(cp.UserID, tempProfileID)
	fullPath := fmt.Sprintf("%s%s", awsKey, fileName)

	var qrFileReader io.Reader = qrFile

	f := &model.File{
		Name:   fullPath,
		Type:   "image/png",
		Size:   qrFileStat.Size(),
		ETag:   "ljksdfajklfj2l3kj4klfksjfd4llkj",
		Reader: qrFileReader,
	}

	_, err = storageService.UploadUserFile("", awsKey, fileName, f, nil, nil, nil, nil, true)
	if err != nil {
		return nil, errors.Wrap(err, "unable to upload QR")
	}

	res, err := storageService.GetUserFile(awsKey, fileName)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get presigned URL")
	}

	payload.QR = res.Filename

	err = os.Remove(localQrPath)
	if err != nil {
		return nil, errors.Wrap(err, "unable to delete qr locally.")
	}

	dbconn, err := db.New(consts.Request)
	if err != nil {
		return nil, errors.Wrap(err, "unable to establish connection with Request.")
	}
	conn, client := dbconn.Collection, dbconn.Client
	defer client.Disconnect(context.TODO())

	_, err = conn.InsertOne(context.TODO(), payload)
	if err != nil {
		return nil, errors.Wrap(err, "unable to insert connection request at mongo.")
	}

	return util.SetResponse(payload, 1, "Code generated successfully"), nil
}

func deleteCode(db *mongodatabase.DBConfig, mysql *database.Database, storageService storage.Service,
	accountID int, payload map[string]interface{}) (map[string]interface{}, error) {
	var err error

	dbconn, err := db.New(consts.Request)
	if err != nil {
		return nil, errors.Wrap(err, "unable to establish connection with Request.")
	}

	conn, client := dbconn.Collection, dbconn.Client
	defer client.Disconnect(context.TODO())

	var idsToDelete []string
	var reqs []model.ConnectionRequest

	data, ok := payload["codes"].([]interface{})
	if ok {
		for _, val := range data {
			idsToDelete = append(idsToDelete, val.(string))
		}
	} else {
		return nil, errors.Wrap(err, "invalid data mapping")
	}

	filter := bson.M{"code": bson.M{"$in": idsToDelete}}

	// find requests from mongo
	cur, err := conn.Find(context.TODO(), filter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find all the requests")
	}
	err = cur.All(context.TODO(), &reqs)
	if err != nil {
		return nil, errors.Wrap(err, "unable to decode cursor")
	}

	// delete QR codes from wasabi
	for idx := range reqs {
		// get basic info
		senderProfileID, _ := strconv.Atoi(reqs[idx].ProfileID)
		cp, err := getConciseProfile(mysql, senderProfileID, storageService)
		if err != nil {
			return nil, errors.Wrap(err, "unable to find basic info")
		}
		fileName := fmt.Sprintf("%s_%s_%s.png", cp.FirstName, cp.LastName, reqs[idx].Code)
		key := util.GetKeyForProfileQR(accountID, senderProfileID)
		_, err = storageService.DeleteTempMedia(key, fileName)
		if err != nil {
			return nil, errors.Wrap(err, "unable to delete QR code from wasabi")
		}
	}

	// delete ids from mongo
	ret, err := conn.DeleteMany(context.TODO(), filter)
	if err != nil {
		return nil, errors.Wrap(err, "unable to delete multiple QR codes")
	}

	msg := "%s deleted successfully"
	if ret.DeletedCount == 1 {
		msg = fmt.Sprintf(msg, "Code")
	} else {
		msg = fmt.Sprintf(msg, "Codes")
	}
	return util.SetResponse(nil, 1, msg), nil
}

func sendConnectionRequest(mysql *database.Database, db *mongodatabase.DBConfig, emailService email.Service,
	storageService storage.Service, profileID int, connReq map[string]interface{},
) (map[string]interface{}, error) {
	var err error
	dbconn, err := db.New(consts.Request)
	if err != nil {
		return nil, errors.Wrap(err, "unable to establish connection with Request.")
	}

	conn, client := dbconn.Collection, dbconn.Client
	defer client.Disconnect(context.TODO())

	connID, _ := primitive.ObjectIDFromHex(connReq["_id"].(string))

	// get the code based off id
	req := model.ConnectionRequest{}
	err = conn.FindOne(context.TODO(), bson.M{"_id": connID}).Decode(&req)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find the code.")
	}

	// save receiver's email in the connection request
	_, err = conn.UpdateOne(context.TODO(), bson.M{"_id": connID}, bson.M{"$set": bson.M{"email1": connReq["email1"].(string)}})
	if err != nil {
		return nil, errors.Wrap(err, "unable to save email in Request.")
	}

	idInt, _ := strconv.Atoi(req.ProfileID)
	cp, err := getConciseProfile(mysql, idInt, storageService)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get basic info")
	}

	email := model.Email{
		Receiver: connReq["email1"].(string),
		Header:   "Connection request from sidekiq",
		Subject:  fmt.Sprintf("%s %s has sent you a connection request", cp.FirstName, cp.LastName),
		TextBody: "",
		HtmlBody: fmt.Sprintf("<p style='font-size:20px'>Enter the code or scan the QR using sidekiq scanner to connect</p><br><p style='font-size:30px'>%s</p><br><img style='border:5px solid black' src='%s' width='200' height='200' alt='No QR' /><br>", req.Code, req.QR),
	}

	err = emailService.SendEmail(email)
	if err != nil {
		return nil, errors.Wrap(err, "unable to send email")
	}

	return util.SetResponse(nil, 1, "Email sent successfully."), nil
}

func acceptConnectionRequest(mysql *database.Database, db *mongodatabase.DBConfig, profileID int, code string) (map[string]interface{}, error) {
	dbconn, err := db.New(consts.Request)
	if err != nil {
		return nil, err
	}
	connReqColl, connReqClient := dbconn.Collection, dbconn.Client
	defer connReqClient.Disconnect(context.TODO())

	// check if the code has expired or not
	var connReq model.ConnectionRequest
	err = connReqColl.FindOne(context.TODO(), bson.M{"code": code}).Decode(&connReq)
	if err != nil {
		return util.SetResponse(nil, 0, "The code has either expired or incorrect."), nil
	}

	dbconn2, err := db.New(consts.Connection)
	if err != nil {
		return nil, err
	}
	connColl, connClient := dbconn2.Collection, dbconn2.Client
	defer connClient.Disconnect(context.TODO())

	// fetch the basic and shareable info of the receiver (profileID)
	// fetch the basic and shareable info of the sender (connReq.ProfileID)

	// check if the connection already exists or not
	c1, _ := connColl.CountDocuments(context.TODO(), bson.M{"profileID": strconv.Itoa(profileID), "connectionID": connReq.ProfileID})
	c2, _ := connColl.CountDocuments(context.TODO(), bson.M{"connectionID": connReq.ProfileID, "profileID": strconv.Itoa(profileID)})

	if int(c1) > 0 && int(c2) > 0 {
		return util.SetResponse(nil, 0, "Connection already exists"), nil
	}

	records := []interface{}{
		model.Connection{
			ProfileID:           strconv.Itoa(profileID),
			ConnectionProfileID: connReq.ProfileID,
		},
		model.Connection{
			ProfileID:           connReq.ProfileID,
			ConnectionProfileID: strconv.Itoa(profileID),
		},
	}

	// check if self connecting
	if strconv.Itoa(profileID) == connReq.ProfileID {
		return util.SetResponse(nil, 0, "Cannot establish connection with yourself."), nil
	}

	// if not exists, insert two records in Connection collection
	// save basic. This would be for both the entries
	for _, r := range records {
		record := r.(model.Connection)
		var p *model.Profile
		p, err := fetchProfileInfoBasedOffShareableSettings(mysql, record.ConnectionProfileID)
		if err != nil {
			return nil, errors.Wrap(err, "unable to find shareable info")
		}

		// map profile date into connection
		record.ID = primitive.NewObjectID()
		record.FirstName = p.FirstName
		record.LastName = p.LastName
		record.ScreenName = p.ScreenName
		record.Bio = p.Bio
		record.Birthday = p.Birthday
		record.Address1 = p.Address1
		record.Address2 = p.Address2
		record.Email1 = p.Email1
		record.Gender = p.Gender
		record.IsBlocked = false
		record.IsActive = true
		record.IsArchived = false

		_, err = connColl.InsertOne(context.TODO(), record)
		if err != nil {
			return nil, errors.Wrap(err, "unable to insert record at Connection.")
		}
	}

	// fetch the sender's connection information
	var senderConnection model.Connection
	err = connColl.FindOne(context.TODO(), bson.M{"connectionID": connReq.ProfileID, "profileID": strconv.Itoa(profileID)}).Decode(&senderConnection)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch sender's connection info.")
	}

	// remove request from the collection
	_, err = connReqColl.DeleteOne(context.TODO(), bson.M{"code": code})
	if err != nil {
		return nil, errors.Wrap(err, "unable to delete the code.")
	}

	return util.SetResponse(senderConnection, 1, "Sender's info fetched successfully"), nil
}

func fetchManagingProfiles(db *database.Database, profileID int) (map[string]interface{}, error) {
	managingProfiles := []model.ConciseProfile{}

	stmt := `SELECT id, firstName, lastName, photo FROM` + "`sidekiq-dev`.AccountProfile WHERE managedByID = ?"
	err := db.Conn.Select(&managingProfiles, stmt, profileID)
	if err != nil {
		return nil, err
	}
	return util.SetResponse(managingProfiles, 1, "Managing profiles fetched successfully."), nil
}

func moveConnection(db *mongodatabase.DBConfig, payload map[string]interface{}, profileID int) (map[string]interface{}, error) {
	var connectionIDs []string
	var action bool
	var statusType string

	for _, record := range payload {
		switch reflect.TypeOf(record).Kind() {
		case reflect.Slice:
			s := reflect.ValueOf(record)

			for i := 0; i < s.Len(); i++ {
				var v interface{} = s.Index(i).Interface()
				connectionIDs = append(connectionIDs, v.(string))
			}
		}
		if rec, ok := record.(bool); ok {
			action = rec
		}
		if rec, ok := record.(string); ok {
			statusType = rec
		}
	}

	if len(connectionIDs) == 0 {
		return util.SetResponse(nil, 0, "ConnectionIDs not found"), nil
	}

	dbConn, err := db.New("Connection")
	if err != nil {
		return nil, err
	}
	connCollection, connClient := dbConn.Collection, dbConn.Client
	fmt.Println(connCollection)
	defer connClient.Disconnect(context.TODO())

	var filter primitive.M
	var update primitive.M

	profileIDStr := strconv.Itoa(profileID)

	for _, connectionProfileID := range connectionIDs {

		filter = bson.M{"$and": bson.A{
			bson.M{"profileID": profileIDStr},
			bson.M{"connectionID": connectionProfileID},
		}}

		switch statusType {
		case "archive":
			update = bson.M{"$set": bson.M{"isArchived": action}}
			if !action {
				// set isActive to true  and isBlocked = false
				update["$set"].(bson.M)["isActive"] = true
				update["$set"].(bson.M)["isBlocked"] = false
			} else {
				// set isActive and isBlocked to false
				update["$set"].(bson.M)["isActive"] = false
				update["$set"].(bson.M)["isBlocked"] = false
			}
		case "blocked":
			update = bson.M{"$set": bson.M{"isBlocked": action}}
			if !action {
				// set isActive to true  and isArchived = false
				update["$set"].(bson.M)["isActive"] = true
				update["$set"].(bson.M)["isArchived"] = false
			} else {
				// set isActive to false  and isArchived = false
				update["$set"].(bson.M)["isActive"] = false
				update["$set"].(bson.M)["isArchived"] = false
			}
		case "active":
			update = bson.M{"$set": bson.M{"isActive": action}}
			if !action {
				update["$set"].(bson.M)["isArchived"] = true
				update["$set"].(bson.M)["isBlocked"] = false
			} else {
				update["$set"].(bson.M)["isArchived"] = false
				update["$set"].(bson.M)["isBlocked"] = false
			}
		default:
			return util.SetResponse(nil, 0, "Request type invalid"), nil
		}

		fmt.Println("update: ", update)
		_, err = connCollection.UpdateOne(context.TODO(), filter, update)
		if err != nil {
			return util.SetResponse(nil, 0, "unable to perform update"), nil
		}
	}

	return util.SetResponse(nil, 1, "Connections moved successfully"), nil
}

func getOrgStaff(storageService storage.Service, db *database.Database, profileID int, accountID, limit, page, searchParameter string) (map[string]interface{}, error) {
	accountIDInt, err := strconv.Atoi(accountID)
	if err != nil {
		return nil, err
	}
	// pagination calculation
	pageInt, err := strconv.Atoi(page)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to int")
	}
	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to int")
	}
	offset := limitInt * (pageInt - 1)

	// if any search parameter exists
	var searchFilter string
	if searchParameter != "" {
		searchFilter = `AND
		(
		   CONCAT(firstName, ' ', lastName) LIKE '%` + searchParameter + `%'
		)`
	}

	// check count in db
	var count int
	stmt := `SELECT
				COUNT(DISTINCT id)
				userName
			FROM ` + "`sidekiq-dev`.Account " +
		`WHERE
			id IN (
				SELECT
				managedByID
				FROM ` + "`sidekiq-dev`.AccountProfile " +
		`WHERE
				accountID = ?
			) ` + searchFilter

	err = db.Conn.Get(&count, stmt, accountIDInt)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get record's existence")
	}

	if count == 0 {
		return util.SetPaginationResponse(nil, 0, 1, "No staff profiles found"), nil
	}

	var staffProfiles []model.ConciseProfile
	// stmt = `SELECT
	// 			id,
	// 			firstName,
	// 			lastName,
	// 			photo,
	// 			userName
	// 		FROM ` + "`sidekiq-dev`.Account " +
	// 	`WHERE
	// 		id IN (
	// 			SELECT
	// 			managedByID
	// 			FROM ` + "`sidekiq-dev`.AccountProfile " +
	// 	`WHERE
	// 			accountID = ?
	// 		) ` + searchFilter + ` LIMIT ` + limit + ` OFFSET ` + fmt.Sprintf("%v", offset)

	stmt = `SELECT
			id,
			firstName,
			lastName,
			photo
		FROM ` + "`sidekiq-dev`.Account " +
		`WHERE
		id IN (
			SELECT
			managedByID
			FROM ` + "`sidekiq-dev`.AccountProfile " +
		`WHERE
			accountID = ?
		) ` + searchFilter + ` LIMIT ` + limit + ` OFFSET ` + fmt.Sprintf("%v", offset)

	err = db.Conn.Select(&staffProfiles, stmt, accountID)
	if err != nil {
		if err == sql.ErrNoRows {
			return util.SetResponse(nil, 0, "No staff profiles found"), nil
		}
		return nil, err
	}
	for i := range staffProfiles {
		staffProfiles[i].Photo, err = getAccountImage(db, storageService, staffProfiles[i].Id, staffProfiles[i].Id)
		if err != nil {
			staffProfiles[i].Photo = ""
			fmt.Println("unable to fetch account photo")
		}

		staffProfiles[i].Thumbs, err = getAccountImageThumb(db, storageService, staffProfiles[i].Id)
		if err != nil {
			staffProfiles[i].Thumbs = model.Thumbnails{}
			fmt.Println("unable to fetch account photo")
		}
	}

	return util.SetPaginationResponse(staffProfiles, 1, 1, "Staff profiles fetched successfully"), nil
}

func getOrgInfo(db *database.Database, accountID int, storageService storage.Service) (map[string]interface{}, error) {
	var accountType int
	stmt := "SELECT accountType FROM `sidekiq-dev`.Account where id = ?"
	err := db.Conn.Get(&accountType, stmt, accountID)
	if err != nil {
		return nil, err
	}
	if accountType != 3 {
		return util.SetResponse(nil, 1, "Account type invalid"), nil
	}
	var Info model.Organization
	stmt = `
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
	err = db.Conn.Get(&Info, stmt, accountID)
	if err == sql.ErrNoRows {
		return nil, errors.Wrap(err, "no data found for the given parameters")
	} else if err != nil {
		return nil, errors.Wrap(err, "unable to fetch org info")
	}

	Info.Photo, err = getOrgImage(db, storageService, accountID)
	if err != nil {
		Info.Photo = ""
		logrus.Println("Organization image not get ", err)
	}

	Info.Thumbs, err = getOrgImageThumb(db, storageService, accountID)
	if err != nil {
		Info.Thumbs = model.Thumbnails{}
		Info.Thumbs.Original = Info.Photo
		logrus.Println("Organization thumb image not get ", err)
	} else {
		Info.Thumbs.Original = Info.Photo
	}

	return util.SetResponse(Info, 1, "Organization information fetched successfully"), nil
}

func contains(db *database.Database, tableName, dbField, value string, accountID int) bool {
	fetchstmt := fmt.Sprintf("SELECT COUNT(*) AS COUNT FROM `sidekiq-dev`.%s WHERE %s = ? AND accountID != ?", tableName, dbField)
	var count int
	err := db.Conn.Get(&count, fetchstmt, value, accountID)
	if err != nil {
		fmt.Println("error in count query")
		return true
	}
	fmt.Println("Count", count)
	if count == 0 {
		return false
	}
	return true
}

func updateOrgInfo(db *database.Database, payload model.Organization) (map[string]interface{}, error) {
	var accountType int
	stmt := "SELECT accountType FROM `sidekiq-dev`.Account where id = ?"
	err := db.Conn.Get(&accountType, stmt, payload.AccountID)
	if err != nil {
		return nil, err
	}
	if accountType != 3 {
		return util.SetResponse(nil, 1, "Account type should be of organization"), nil
	}

	// check if org creds already exists
	if contains(db, "OrgProfile", "email", payload.Email, payload.AccountID) {
		return util.SetResponse(nil, 0, "Email already exists. Please use another email"), nil
	} else if contains(db, "OrgProfile", "registrationNumber", payload.RegistrationNumber, payload.AccountID) {
		return util.SetResponse(nil, 0, "Registration Number already exists. Please use another Registration Number"), nil
	} else if contains(db, "OrgProfile", "website", payload.Website, payload.AccountID) {
		return util.SetResponse(nil, 0, "Website address already exists. Please use another Website address"), nil
	} else if contains(db, "OrgProfile", "organizationName", payload.OrganizationName, payload.AccountID) {
		return util.SetResponse(nil, 0, "Organization name already exists. Please use another Organization name"), nil
	}

	stmt = "UPDATE `sidekiq-dev`.OrgProfile" +
		` SET
				organizationName = :organizationName,
				website = :website,
				registrationNumber = :registrationNumber,
				email = :email,
				phone = :phone,
				address1 = :address1,
				address2 = :address2,
				city = :city,
				state = :state,
				zip = :zip,
				country = :country,
				bio = :bio
			WHERE 
				accountID = :accountID
			`
	_, err = db.Conn.NamedExec(stmt, payload)
	if err != nil {
		return nil, errors.Wrap(err, "unable to update org info")
	}
	return util.SetResponse(nil, 1, "Organization information updated successfully"), nil
}

func fetchMembershipDetails(db *database.Database, id string) (map[string]interface{}, error) {
	accountID, err := strconv.Atoi(id)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to int")
	}
	data := model.MembershipInfo{}
	var accountType *int

	stmt := "SELECT accountType FROM `sidekiq-dev`.Account where id = ?"
	err = db.Conn.Get(&accountType, stmt, accountID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch account type")
	}

	stmt = "SELECT description, fee, profiles FROM `sidekiq-dev`.Services where id = ?"
	err = db.Conn.Get(&data.AccountInfo, stmt, accountType)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch from services")
	}

	stmt = "SELECT description FROM `sidekiq-dev`.Policies where type = 'cancellation'"
	err = db.Conn.Get(&data.CancellationPolicy, stmt)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch from policies")
	}

	stmt = "SELECT IFNULL(endDate, '') FROM `sidekiq-dev`.Accountervices where accountID = ?"
	err = db.Conn.Get(&data.ExpirationDate, stmt, accountID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch from user-services")
	}

	return util.SetResponse(data, 1, "Account membership details fetched successfully"), nil
}

func fetchBoards(db *database.Database, profileID int, sectionType string) (map[string]interface{}, error) {
	var boards []struct {
		BoardTitle string `json:"boardTitle" db:"boardTitle"`
		BoardID    string `json:"boardID" db:"boardID"`
	}
	var boardStmt string
	var err error

	if sectionType == "following" {
		boardStmt = "SELECT DISTINCT boardTitle, boardID FROM `sidekiq-dev`.BoardsFollowed WHERE profileID = ?"
	} else if sectionType == "followed" {
		boardStmt = "SELECT DISTINCT boardTitle, boardID FROM `sidekiq-dev`.BoardsFollowed WHERE ownerID = ?"
	} else {
		return util.SetResponse(nil, 0, "Invalid parameter in requested URL"), nil
	}

	err = db.Conn.Select(&boards, boardStmt, profileID)
	if err != nil {
		return nil, err
	}

	return util.SetResponse(boards, 1, cases.Title(language.English).String(strings.ToLower(sectionType))+" boards fetched successfully"), nil
}

func fetchStaffProfile(db *database.Database, accountID, comanagerID int, storageService storage.Service) (map[string]interface{}, error) {
	var stmt string
	fmt.Println("accountID", accountID)
	fmt.Println("comanagerID", comanagerID)
	profile := model.OrgStaff{}

	stmt = `
        SELECT
            u.firstName,
            u.lastName,
            IFNULL(s.photo, "") as photo,
            IFNULL(s.bio, "") as bio,
            IFNULL(s.address1, "") as address1,
            IFNULL(s.city, "") as city,
            IFNULL(s.nickName, "") as nickName,
            IFNULL(s.bio, "") as bio,
            IFNULL(s.phone1, "") as phone1,
            IFNULL(s.phone2, "") as phone2,
            IFNULL(s.email1, "") as email1,
            IFNULL(s.emergencyEmail, "") as emergencyEmail,
            IFNULL(s.address2,"") as address2, 
            IFNULL(s.state, "") as state,
            IFNULL(s.zip, "") as zip,
            IFNULL(s.country, "") as country,
            IFNULL(s.gender, 0) as gender,
            IFNULL(s.emergencyContact, "") as emergencyContact,
            IFNULL(s.emergencyContactPhone, "") as emergencyContactPhone,
            IFNULL(s.emergencyContactID, "") as emergencyContactID,
            IFNULL(s.notes, "") as notes,
            IFNULL(s.startDate, "") as startDate,
            IFNULL(s.endDate, "") as endDate,
            IFNULL(s.jobTitle, "") as jobTitle,
            IFNULL(s.skills, "") as skills,
            IFNULL(s.interests, "") as interests,
            IFNULL(s.reportsToID, "") as reportsToID
        FROM ` + "`sidekiq-dev`.OrgStaff s" +
		` JOIN ` + "`sidekiq-dev`.AccountProfile p ON s.managedByID = p.accountID" +
		` JOIN ` + "`sidekiq-dev`.Account u ON p.accountID = u.id" +
		` WHERE s.accountID = ? AND p.accountID = ?`

	err := db.Conn.Get(&profile, stmt, accountID, comanagerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return util.SetResponse(nil, 0, "staff profile info not found"), nil
		}
		return nil, err
	}

	profile.Photo, err = getAccountImage(db, storageService, comanagerID, 0)
	if err != nil {
		logrus.Println("unable to get account image")
		profile.Photo = ""
	}

	profile.Thumbs, err = getAccountImageThumb(db, storageService, comanagerID)
	if err != nil {
		profile.Thumbs = model.Thumbnails{}
		logrus.Println("unable to get account thumb image")
	}

	return util.SetResponse(profile, 1, "Staff profile info fetched successfully"), nil
}

func updateStaffProfile(db *database.Database, accountID, comanagerID int, payload model.OrgStaff) (map[string]interface{}, error) {
	stmt := "UPDATE `sidekiq-dev`.OrgStaff s JOIN `sidekiq-dev`.AccountProfile p ON p.accountID = s.managedByID" + `
        SET 
            s.bio = :bio,
            s.address1 = :address1,
			s.address2 = :address2,
            s.city = :city,
            s.nickName = :nickName,
            s.phone1 = :phone1,
            s.phone2 = :phone2,
            s.email1 = :email1,
            s.emergencyEmail = :emergencyEmail,
            s.state = :state,
            s.zip = :zip,
            s.country = :country,
            s.gender = :gender,
            s.emergencyContact = :emergencyContact,
            s.emergencyContactPhone = :emergencyContactPhone,
            s.emergencyContactID = :emergencyContactID,
            s.notes = :notes,
            s.startDate = :startDate,
            s.endDate = :endDate,
            s.jobTitle = :jobTitle,
            s.skills = :skills,
            s.interests = :interests,
            s.reportsToID = :reportsToID 
        WHERE 
        s.accountID = :orgID AND p.accountID = :id`

	payload.OrgID = accountID
	payload.Id = comanagerID

	_, err := db.Conn.NamedExec(stmt, payload)
	if err != nil {
		return nil, err
	}

	return util.SetResponse(nil, 1, "Staff profile updated successfully"), nil
}

func fetchOrganizationProfiles(db *database.Database, profileID int, accountID string) (map[string]interface{}, error) {
	var err error
	var profiles []model.OrganizationProfiles
	var memberProfile []model.BasicProfileInfo

	accountIDInt, _ := strconv.Atoi(accountID)
	stmt := "SELECT id, firstName, lastName, IFNULL(managedByID, 0) as managedByID FROM `sidekiq-dev`.AccountProfile WHERE accountID = ?"
	err = db.Conn.Select(&memberProfile, stmt, accountID)
	if err == sql.ErrNoRows {
		memberProfile = nil
	} else if err != nil {
		return nil, err
	}

	if memberProfile == nil {
		return util.SetResponse(nil, 1, "Organization has no profiles"), nil
	}

	for i := range memberProfile {
		var temp model.OrganizationProfiles
		temp.ProfileInfo.ID = memberProfile[i].ID
		temp.ProfileInfo.FirstName = memberProfile[i].FirstName
		temp.ProfileInfo.LastName = memberProfile[i].LastName
		temp.ProfileInfo.Photo = memberProfile[i].Photo
		temp.ProfileInfo.ManagedByID = memberProfile[i].ManagedByID
		profiles = append(profiles, temp)
	}

	var coMangagerProfile []model.BasicProfileInfo
	stmt = "SELECT id, firstName, lastName, photo FROM `sidekiq-dev`.Account WHERE id IN (SELECT managedByID FROM `sidekiq-dev`.AccountProfile WHERE accountID = ?)"
	err = db.Conn.Select(&coMangagerProfile, stmt, accountIDInt)
	if err == sql.ErrNoRows {
		coMangagerProfile = nil
	} else if err != nil {
		return nil, err
	}

	if coMangagerProfile == nil {
		return util.SetResponse(profiles, 1, "Organization profiles fetched successfully"), nil
	}

	for i := range memberProfile {
		id := memberProfile[i].ManagedByID
		for j := range coMangagerProfile {
			if id == coMangagerProfile[j].ID {
				profiles[i].ComanagerInfo = coMangagerProfile[j]
			}
		}
	}

	return util.SetResponse(profiles, 1, "Organization profiles fetched successfully"), nil
}

func fetchMemberProfileInfo(db *database.Database, profileID int) (map[string]interface{}, error) {
	// profiles := make(model.OrganizationProfiles, 0)
	var profile model.OrganizationProfiles

	var memberProfile model.BasicProfileInfo
	stmt := "SELECT id, CONCAT(firstName,' ',lastName) as fullName, IFNULL(managedByID, 0) as managedByID FROM `sidekiq-dev`.AccountProfile WHERE id = ?"
	err := db.Conn.Select(&memberProfile, stmt, profileID)
	if err == sql.ErrNoRows {
		return util.SetResponse(nil, 1, "Profile info not found"), nil
	} else if err != nil {
		return nil, err
	}

	var coMangagerProfile []model.Profile
	stmt = "SELECT id, CONCAT(firstName,' ',lastName) as fullName, email1, phone1, address1, photo, screenName FROM `sidekiq-dev`.AccountProfile WHERE id IN (SELECT managedByID FROM `sidekiq-dev`.AccountProfile WHERE id = ?)"
	err = db.Conn.Select(&coMangagerProfile, stmt, profileID)
	if err == sql.ErrNoRows {
		coMangagerProfile = nil
	} else if err != nil {
		return nil, err
	}

	if coMangagerProfile == nil {
		return util.SetResponse(profile, 1, "Organization profiles fetched successfully"), nil
	}

	return util.SetResponse(profile, 1, "Profile info fetched successfully"), nil
}

func fetchProfileBoardsView(storageService storage.Service, sql *database.Database, db *mongodatabase.DBConfig, profileID, myID string) (map[string]interface{}, error) {
	var res []*model.Board
	var result struct {
		Count int `bson:"count"`
	}
	// profileID = targetProfile
	// checking if myID/Caller is connection of profileID or not
	if profileID != myID {
		dbConn, err := db.New(consts.Connection)
		if err != nil {
			return nil, err
		}
		connCollection, connClient := dbConn.Collection, dbConn.Client
		defer connClient.Disconnect(context.TODO())
		countPipeline := mongo.Pipeline{
			bson.D{
				{Key: "$match", Value: bson.M{
					"profileID":    myID,
					"isBlocked":    false,
					"isArchived":   false,
					"connectionID": profileID,
				}},
			},
			bson.D{
				{Key: "$count", Value: "count"},
			},
		}
		cursor, err := connCollection.Aggregate(context.TODO(), countPipeline)
		if err != nil {
			return nil, errors.Wrap(err, "unable to find count")
		}
		defer cursor.Close(context.TODO())
		if cursor.Next(context.TODO()) {
			err = cursor.Decode(&result)
			if err != nil {
				return nil, errors.Wrap(err, "unable to store in count")
			}
		}
	}
	dbconn, err := db.New(consts.Board)
	if err != nil {
		return nil, err
	}
	collection, client := dbconn.Collection, dbconn.Client
	defer client.Disconnect(context.TODO())

	filter := bson.M{}
	// Caller want to see his/her boards
	if profileID == myID || result.Count > 0 {
		filter = bson.M{
			"$and": bson.A{
				bson.M{"owner": profileID},
				bson.M{"state": consts.Active},
			},
		}
	}
	// Caller not a connection but a board member
	// if profileID != myID && result.Count <= 0 {
	// 	filter = bson.M{
	// 		"$and": bson.A{
	// 			bson.M{"$or": bson.A{
	// 				bson.M{"viewers": myID},
	// 				bson.M{"subscribers": myID},
	// 				bson.M{"admins": myID},
	// 				bson.M{"authors": myID},
	// 			}},
	// 			bson.M{"owner": profileID},
	// 			bson.M{"visible": consts.Public},
	// 			bson.M{"state": consts.Active},
	// 		},
	// 	}
	// }

	// if profileID != myID && result.Count > 0 {
	// 	filter = bson.M{
	// 		"$and": bson.A{
	// 			bson.M{"owner": profileID},
	// 			bson.M{"visible": consts.Public},
	// 			bson.M{"state": consts.Active},
	// 		},
	// 	}
	// }

	if profileID != myID && result.Count <= 0 {
		filter = bson.M{
			"$and": bson.A{
				bson.M{"owner": profileID},
				bson.M{"visible": consts.Public},
				bson.M{"state": consts.Active},
			},
		}
	}

	findOptions := options.Find()
	findOptions.SetSort(bson.M{"createDate": -1})

	cursor, err := collection.Find(context.TODO(), filter, findOptions)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find boards")
	}
	defer cursor.Close(context.TODO())

	err = cursor.All(context.TODO(), &res)
	if err != nil {
		return nil, err
	}

	for key, value := range res {

		// fetching profile image
		profileInt, err := strconv.Atoi(value.Owner)
		if err != nil {
			fmt.Println("unable to convert profile id to int", value.Owner, err)
		}
		photo, err := getProfileImage(sql, storageService, 0, profileInt)
		if err != nil {
			fmt.Println("unable to find profile image for profileID", value.Owner, err)
		}
		res[key].OwnerInfo.Photo = photo
	}
	fmt.Println(len(res))
	return util.SetResponse(res, 1, "Profile boards view fetched successfully"), nil
}

func fetchProfileView(storageService storage.Service, sql *database.Database, db *mongodatabase.DBConfig, profileID, myID string) (map[string]interface{}, error) {
	var res model.ProfileView
	var result struct {
		Count int `bson:"count"`
	}

	if profileID != myID {
		dbConn, err := db.New(consts.Connection)
		if err != nil {
			return nil, err
		}
		connCollection, connClient := dbConn.Collection, dbConn.Client
		defer connClient.Disconnect(context.TODO())
		countPipeline := mongo.Pipeline{
			bson.D{
				{Key: "$match", Value: bson.M{
					"profileID":    myID,
					"isBlocked":    false,
					"isActive":     false,
					"connectionID": profileID,
				}},
			},
			bson.D{
				{Key: "$count", Value: "count"},
			},
		}

		cursor, err := connCollection.Aggregate(context.TODO(), countPipeline)
		if err != nil {
			return nil, errors.Wrap(err, "unable to find count")
		}
		defer cursor.Close(context.TODO())
		if cursor.Next(context.TODO()) {
			err = cursor.Decode(&result)
			if err != nil {
				return nil, errors.Wrap(err, "unable to store in count")
			}
		}
		res.IsConnected = result.Count > 0
	}
	// fetch profile info
	profileIDInt, err := strconv.Atoi(profileID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to int")
	}
	stmt := `
		SELECT p.firstName, p.lastName,p.visibility,p.showBoards,p.shareable,
			CASE 
					WHEN  s.bio = true THEN IFNULL(p.bio, '') ELSE '' 
			END as bio,
			CASE
					WHEN  s.email = true THEN  IFNULL(p.email1, '') ELSE ''  
			END as email1,
			CASE
					WHEN  s.screenName = true THEN  IFNULL(p.screenName, '') ELSE '' 
			END as screenName,
			CASE
					WHEN  s.gender = true THEN  IFNULL(p.gender, 0) ELSE 0
			END as gender,
			CASE
					WHEN  s.birthday = true THEN  IFNULL(p.birthday, '') ELSE '' 
			END as birthday,
			CASE
					WHEN  s.address1 = true THEN  IFNULL(p.address1, '') ELSE '' 
			END as address1,
			CASE
					WHEN  s.address2 = true THEN  IFNULL(p.address2, '') ELSE '' 
			END as address2,
			CASE
					WHEN  s.phone = true THEN  IFNULL(p.phone1, '') ELSE '' 
			END as phone1

			FROM ` + "`sidekiq-dev`.AccountProfile as p " +
		"JOIN `sidekiq-dev`.ShareableSettings as s on p.id =s.profileID AND p.id = ?"

	err = sql.Conn.Get(&res, stmt, profileIDInt)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch shareable settings fields")
	}
	if res.Birthday == "0000-00-00" {
		res.Birthday = ""
	}
	// the caller is seeing his/her profile
	if profileID == myID {
		res.IsPrivate = false
	}
	// caller is a connection of profileID
	if res.Visibility == "Member" && result.Count > 0 {
		res.IsPrivate = false
	}
	// caller is a connection of profileID
	if res.Visibility == "Member" && result.Count <= 0 {
		res = model.ProfileView{
			FirstName: res.FirstName,
			LastName:  res.LastName,
			IsPrivate: true,
		}
	}
	// caller is a connection of profileID
	if res.Visibility == "Private" && result.Count > 0 {
		res.IsPrivate = true
	}
	// caller is a not connection of profileID
	if res.Visibility == "Private" && profileID != myID && result.Count <= 0 {
		res = model.ProfileView{
			FirstName: res.FirstName,
			LastName:  res.LastName,
			IsPrivate: true,
		}
	}
	// fetching profile image
	profileInt, err := strconv.Atoi(profileID)
	if err != nil {
		fmt.Println("unable to convert profile id to int", profileID, err)
	}
	photo, err := getProfileImage(sql, storageService, 0, profileInt)
	if err != nil {
		fmt.Println("unable to find profile image for profileID", profileID, err)
	}
	res.Photo = photo
	return util.SetResponse(res, 1, "Profile about fetched successfully"), nil
}

func addConnectionDetails(mysql *database.Database, db *mongodatabase.DBConfig, payload model.Connection, storageService storage.Service) (map[string]interface{}, error) {
	dbconn, err := db.New(consts.Connection)
	if err != nil {
		return nil, errors.Wrap(err, "error in connecting to mongo")
	}
	coll, client := dbconn.Collection, dbconn.Client
	defer client.Disconnect(context.TODO())

	// var result model.Connection
	filter := bson.M{"profileID": payload.ProfileID,
		"connectionID": payload.ConnectionProfileID}

	var connObj model.Connection
	err = coll.FindOne(context.TODO(), filter).Decode(&connObj)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find document")
	}

	payload.IsActive = connObj.IsActive
	payload.IsArchived = connObj.IsArchived
	payload.IsBlocked = connObj.IsBlocked
	payload.CreateDate = connObj.CreateDate

	mapdata, err := util.ToMap(payload)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert map")
	}

	delete(mapdata, "_id")

	_, err = coll.UpdateOne(context.TODO(), filter, bson.M{"$set": mapdata})
	if err != nil {
		return nil, errors.Wrap(err, "unable to update mongo collection")
	}

	var data map[string]interface{}
	err = coll.FindOne(context.TODO(), filter).Decode(&data)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find collection")
	}

	profileIDInt, _ := strconv.Atoi(data["connectionID"].(string))
	data["photo"], err = getProfileImage(mysql, storageService, 0, profileIDInt)
	if err != nil {
		fmt.Println("error in fetching profile photo")
	}

	return util.SetResponse(data, 1, "Connection info updated successfully"), nil
}

func getConnectionDetails(db *mongodatabase.DBConfig, callerProfileID, connectionProfileID string) (map[string]interface{}, error) {
	var response model.Connection
	dbconn, err := db.New(consts.Connection)
	if err != nil {
		return nil, errors.Wrap(err, "error in connecting to connection collection")
	}
	conn, client := dbconn.Collection, dbconn.Client
	defer client.Disconnect(context.TODO())

	filter := bson.M{"profileID": callerProfileID,
		"connectionID": connectionProfileID}

	err = conn.FindOne(context.TODO(), filter).Decode(&response)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return util.SetResponse(nil, 1, "profile id not a connection"), nil
		}
		return nil, errors.Wrap(err, "unable to get connection details")
	}
	return util.SetResponse(response, 1, "Connection info fetched successfully"), nil
}

func fetchDefaultBoardID(sql *database.Database, profileID int) (string, error) {
	var defaultBoardID string
	stmt := "SELECT defualtThingsBoard from `sidekiq-dev`.AccountProfile WHERE id = ?"
	err := sql.Conn.Get(defaultBoardID, stmt, profileID)
	if err != nil {
		return "", errors.Wrap(err, "unable to fetch defaultBoardID")
	}

	return defaultBoardID, nil
}

func updateProfileTagsNew(db *mongodatabase.DBConfig, mySql *database.Database, profileID string) error {
	profileIDInt, _ := strconv.Atoi(profileID)
	var profileTags []string

	// fetch all tags from mongo collections where owner is profileID
	errCh := make(chan error)
	// filter := bson.D{{Key: "owner", Value: profileID}}
	filter := bson.M{"$and": bson.A{
		bson.M{"owner": profileID},
		bson.M{"state": "ACTIVE"},
	}}
	var mx sync.Mutex

	// fetch all Notes tags and append to tags
	go func(errChan chan<- error) {
		defer util.RecoverGoroutinePanic(errChan)
		var notes []*model.Note
		noteConn, err := db.New(consts.Note)
		if err != nil {
			fmt.Println("unable to connect note")
			errCh <- errors.Wrap(err, "unable to connect note")
			return
		}
		noteCollection, noteClient := noteConn.Collection, noteConn.Client
		defer noteClient.Disconnect(context.TODO())
		curr, err := noteCollection.Find(context.TODO(), filter)
		if err != nil {
			fmt.Println("unable to fetch note tags")
			errCh <- errors.Wrap(err, "unable to fetch note tags")
			return
		}
		err = curr.All(context.TODO(), &notes)
		if err != nil {
			fmt.Println("error while note")
			errCh <- errors.Wrap(err, "failure in variable mapping")
			return
		}
		for i := range notes {
			mx.Lock()
			profileTags = append(profileTags, notes[i].Tags...)
			mx.Unlock()
		}
		errCh <- nil
	}(errCh)

	// fetch all Tasks tags and append to tags
	go func(errChan chan<- error) {
		defer util.RecoverGoroutinePanic(errChan)
		var tasks []*model.Task
		taskConn, err := db.New(consts.Task)
		if err != nil {
			fmt.Println("unable to connect task")
			errCh <- errors.Wrap(err, "unable to connect task")
			return
		}
		taskCollection, taskClient := taskConn.Collection, taskConn.Client
		defer taskClient.Disconnect(context.TODO())
		curr, err := taskCollection.Find(context.TODO(), filter)
		if err != nil {
			fmt.Println("unable to fetch task tags")
			errCh <- errors.Wrap(err, "unable to fetch task tags")
			return
		}
		err = curr.All(context.TODO(), &tasks)
		if err != nil {
			fmt.Println("error while task")
			errCh <- errors.Wrap(err, "failure in variable mapping")
			return
		}
		// for i := range tasks {
		// 	mx.Lock()
		// 	profileTags = append(profileTags, tasks[i].Tags...)
		// 	mx.Unlock()
		// }
		errCh <- nil
	}(errCh)

	// fetch all Files tags and append to tags
	go func(errChan chan<- error) {
		defer util.RecoverGoroutinePanic(errChan)
		var files []*model.UploadedFile
		fileConn, err := db.New(consts.File)
		if err != nil {
			fmt.Println("unable to connect file")
			errCh <- errors.Wrap(err, "unable to connect file")
			return
		}
		fileCollection, fileClient := fileConn.Collection, fileConn.Client
		defer fileClient.Disconnect(context.TODO())
		curr, err := fileCollection.Find(context.TODO(), filter)
		if err != nil {
			fmt.Println("error while file")
			fmt.Println("unable to fetch files tags")
			errCh <- errors.Wrap(err, "unable to fetch files tags")
			return
		}
		err = curr.All(context.TODO(), &files)
		if err != nil {
			errCh <- errors.Wrap(err, "failure in variable mapping")
			return
		}
		for i := range files {
			mx.Lock()
			profileTags = append(profileTags, files[i].Tags...)
			mx.Unlock()
		}
		errCh <- nil
	}(errCh)

	// fetch all Collection tags and append to tags
	go func(errChan chan<- error) {
		defer util.RecoverGoroutinePanic(errChan)
		var col []*model.Collection
		colConn, err := db.New(consts.Collection)
		if err != nil {
			fmt.Println("unable to connect collection")
			errCh <- errors.Wrap(err, "unable to connect collection")
			return
		}
		colCollection, colClient := colConn.Collection, colConn.Client
		defer colClient.Disconnect(context.TODO())
		curr, err := colCollection.Find(context.TODO(), filter)
		if err != nil {
			fmt.Println("unable to fetch collection tags")
			errCh <- errors.Wrap(err, "unable to fetch collection tags")
			return
		}
		err = curr.All(context.TODO(), &col)
		if err != nil {
			fmt.Println("error while Collection")
			errCh <- errors.Wrap(err, "failure in variable mapping")
			return
		}
		for i := range col {
			mx.Lock()
			profileTags = append(profileTags, col[i].Tags...)
			mx.Unlock()
		}
		errCh <- nil
	}(errCh)

	// fetch all Board tags and append to tags
	go func(errChan chan<- error) {
		defer util.RecoverGoroutinePanic(errChan)
		var boards []*model.Board
		boardConn, err := db.New(consts.Board)
		if err != nil {
			fmt.Println("unable to connect board")
			errCh <- errors.Wrap(err, "unable to connect board")
			return
		}
		boardCollection, boardClient := boardConn.Collection, boardConn.Client
		defer boardClient.Disconnect(context.TODO())
		curr, err := boardCollection.Find(context.TODO(), filter)
		if err != nil {
			fmt.Println("unable to fetch note")
			errCh <- errors.Wrap(err, "unable to fetch board tags")
			return
		}
		err = curr.All(context.TODO(), &boards)
		if err != nil {
			fmt.Println("error while Board")
			errCh <- errors.Wrap(err, "failure in variable mapping")
			return
		}
		for i := range boards {
			mx.Lock()
			profileTags = append(profileTags, boards[i].Tags...)
			mx.Unlock()
		}
		errCh <- nil
	}(errCh)

	for i := 0; i < 5; i++ {
		if err := <-errCh; err != nil {
			fmt.Printf("error occurred from go routine%v", err)
			return err
		}
	}
	// update in mysql
	profileTags = util.RemoveArrayDuplicate(profileTags)
	profileTagsStr := strings.Join(profileTags, ",")
	p := model.Profile{ID: profileIDInt, Tags: profileTagsStr}
	updateStmt := "UPDATE `sidekiq-dev`.AccountProfile SET tags = :tags WHERE id = :id"
	_, err := mySql.Conn.NamedExec(updateStmt, p)
	if err != nil {
		return errors.Wrap(err, "unable to perform update query in MySQL")
	}
	return nil
}

func getOwnerInfoUsingProfileIDs(mysql *database.Database, profileIDs []string) ([]model.ConciseProfile, error) {
	ownerInfo := []model.ConciseProfile{}

	stmt := `SELECT id,firstName, lastName,
			IFNULL(screenName, '') AS screenName,
			IFNULL(photo, '') AS photo FROM` + "`sidekiq-dev`.AccountProfile WHERE id IN (?)"

	query, args, err := sqlx.In(stmt, profileIDs)
	if err != nil {
		return nil, err
	}
	query = mysql.Conn.Rebind(query) // sqlx.In returns queries with the `?` bindvar, rebind it here for matching the database in used (e.g. postgre, oracle etc, can skip it if you use mysql)
	rows, err := mysql.Conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var alb model.ConciseProfile

		if err := rows.Scan(&alb.Id, &alb.FirstName, &alb.LastName, &alb.ScreenName,
			&alb.Photo); err != nil {
			return nil, err
		}
		ownerInfo = append(ownerInfo, alb)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return ownerInfo, nil
}
