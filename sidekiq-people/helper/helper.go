package helper

import (
	"database/sql"
	"fmt"

	"github.com/ProImaging/sidekiq-backend/sidekiq-models/model"
	"github.com/ProImaging/sidekiq-backend/sidekiq-people/app/storage"
	"github.com/ProImaging/sidekiq-backend/sidekiq-people/database"
	"github.com/ProImaging/sidekiq-backend/sidekiq-people/util"

	"github.com/pkg/errors"
)

func GetAccountImage(mysql *database.Database, storageService storage.Service, accountID, profileID int) (string, error) {
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
	return fileData.Filename, nil
}

func GetAccountImageThumb(mysql *database.Database, storageService storage.Service, accountID int) (model.Thumbnails, error) {
	thumbTypes := []string{"sm", "ic"}
	thumbKey := util.GetKeyForUserImage(accountID, "thumbs")
	thumbfileName := fmt.Sprintf("%d.png", accountID)
	thumbs, err := GetThumbnails(storageService, thumbKey, thumbfileName, thumbTypes)
	if err != nil {
		thumbs = model.Thumbnails{}
	}

	return thumbs, nil
}

func GetProfileImage(mysql *database.Database, storageService storage.Service, accountID, profileID int) (string, error) {
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

func GetProfileImageThumb(mysql *database.Database, storageService storage.Service, accountID, profileID int) (model.Thumbnails, error) {
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
	thumbs, err := GetThumbnails(storageService, thumbKey, thumbfileName, thumbTypes)
	if err != nil {
		thumbs = model.Thumbnails{}
	}

	return thumbs, nil
}

func GetConciseProfile(mysql *database.Database, id int, storageService storage.Service) (*model.ConciseProfile, error) {
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
		cp.Photo, err = GetProfileImage(mysql, storageService, cp.UserID, cp.Id)
		if err != nil {
			fmt.Println(cp.Id, err)
			fmt.Println("unable to find profile image for id", cp.Id, err)
		}

		cp.Thumbs, err = GetProfileImageThumb(mysql, storageService, cp.UserID, cp.Id)
		if err != nil {
			fmt.Println(cp.Id, err)
			fmt.Println("unable to find profile thumb image for id", cp.Id, err)
		}

	}
	return cp, nil
}

func GetOrgImage(mysql *database.Database, storageService storage.Service, accountID int) (string, error) {
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

func GetOrgImageThumb(mysql *database.Database, storageService storage.Service, accountID int) (model.Thumbnails, error) {
	thumbTypes := []string{"sm", "ic"}
	thumbKey := util.GetKeyForOrganizationImage(accountID, "thumbs")
	thumbfileName := fmt.Sprintf("%d.png", accountID)
	thumbs, err := GetThumbnails(storageService, thumbKey, thumbfileName, thumbTypes)
	if err != nil {
		thumbs = model.Thumbnails{}
	}

	return thumbs, nil
}
