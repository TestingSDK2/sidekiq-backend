package storage

import (
	"bytes"
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	_ "unsafe"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/config"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/cache"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/consts"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/database"
	repo "github.com/ProImaging/sidekiq-backend/sidekiq-content/model"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/mongodatabase"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/permissions"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/util"
	"github.com/ProImaging/sidekiq-backend/sidekiq-models/model"
	"github.com/ProImaging/sidekiq-backend/sidekiq-models/model/notification"
	peoplerpc "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const BufferSize = 1024 * 1024

// Service - defines file management
type Service interface {
	GetUserFileList(key string) ([]*model.File, error)
	MoveFile(oldkey string, newKey string) error
	GetUserFile(key string, name string) (*model.File, error)
	GetFileParts(profile int, key, uuid string) ([]*model.FilePart, error)
	GetTotalFileSize(profile int, key, uuid string) (*float64, error)
	UploadUserFile(postOwner, key, fileName string, file *model.File, meta *model.FileUpload,
		payload map[string]interface{}, complete chan *model.File, fmdChan chan map[string]interface{}, isDirect bool) (*model.File, error)
	UploadUserFileInCollection(postOwner, key, fileName string, file *model.File, meta *model.FileUpload,
		payload map[string]interface{}, complete chan *model.File, fmdChan chan map[string]interface{}, isDirect bool) (*model.File, error)
	PushToClients(profile int, file *model.File) error
	AddFile(file map[string]interface{}, profileID int) (map[string]interface{}, error)                                    // repeated
	AddFileInCollection(postOwner, key string, file map[string]interface{}, profileID int) (map[string]interface{}, error) // repeated
	DeletePostMedia(ownerInfo *peoplerpc.ConciseProfileReply, boardID, mediaID, postID string) (map[string]interface{}, error)
	DeleteBoardCover(boardID string) (map[string]interface{}, error)
	DeleteTempMedia(key, fileName string) (map[string]interface{}, error)
	ComputeCloudStorage(prefix string) (map[string]interface{}, error)
	UpdateProfileTags(p model.Profile, tags []string) error
	UpdateBoardThingsTags(profileID int, boardID, thingID string, tags []string) error
	RemoveMediaChunks(profileID int) (map[string]interface{}, error)
	FetchConciseProfile(profileID int) (map[string]int, error)
	UpdateFileById(fileID primitive.ObjectID, payload map[string]interface{}) error
}

type service struct {
	config       *config.Config
	dbMaster     *database.Database
	dbReplica    *database.Database
	mongodb      *mongodatabase.DBConfig
	cache        *cache.Cache
	fileStore    model.FileStorage
	tmpFileStore model.FileStorage
}

// NewtService create new storage service
func NewService(repos *repo.Repos, conf *config.Config) Service {
	svc := &service{
		config:       conf,
		dbMaster:     repos.MasterDB,
		dbReplica:    repos.ReplicaDB,
		mongodb:      repos.MongoDB,
		cache:        repos.Cache,
		fileStore:    repos.Storage,
		tmpFileStore: repos.TmpStorage,
	}
	return svc
}

func (s *service) FetchConciseProfile(id int) (map[string]int, error) {
	stmt := "SELECT id, accountID FROM `sidekiq-dev`.AccountProfile WHERE id = ?"
	var profile struct {
		ID        int `db:"id"`
		AccountID int `db:"accountID"`
	}
	err := s.dbMaster.Conn.Get(&profile, stmt, id)
	if err != nil {
		return nil, err
	}
	data := map[string]int{
		"accountID": profile.AccountID,
		"id":        profile.ID,
	}
	util.PrettyPrint(data)
	return data, nil
}

func (s *service) UpdateProfileTags(profile model.Profile, tags []string) error {
	stmt := "SELECT IFNULL(tags, '') as tags FROM `sidekiq-dev`.AccountProfile WHERE id = ?"
	var profileTags string // comma separated string
	err := s.dbMaster.Conn.Get(&profileTags, stmt, profile.ID)
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
	_, err = s.dbMaster.Conn.NamedExec(updateStmt, profile)
	if err != nil {
		return err
	}
	return nil
}

func (s *service) UpdateBoardThingsTags(profileID int, boardID, thingID string, tags []string) error {
	defer util.Recover()
	dbconn, err := s.mongodb.New(consts.BoardThingsTags)
	if err != nil {
		return errors.Wrap(err, "unable to connect to "+consts.BoardThingsTags)
	}
	btt, bttClient := dbconn.Collection, dbconn.Client
	defer bttClient.Disconnect(context.TODO())

	boardObjID, err := primitive.ObjectIDFromHex(boardID)
	if err != nil {
		return errors.Wrap(err, "unable convert string to ObjectID")
	}

	filter := bson.M{"boardID": boardObjID}

	var thingsTags map[string]interface{}
	err = btt.FindOne(context.TODO(), filter).Decode(&thingsTags)
	if err != nil {
		return errors.Wrap(err, "unable to get thingsTags")
	}

	// update tags of particular thing
	var t []string
	if thingsTags["tags"] != nil {
		if bsonTagsMap, ok := thingsTags["tags"].(map[string]interface{}); ok {
			if bsonTags, ok := bsonTagsMap[thingID].(bson.A); ok {
				for _, bt := range bsonTags {
					t = append(t, bt.(string))
				}
			}
		}
	}

	t = append(t, tags...)
	t = util.RemoveArrayDuplicate(t)

	// modify the object and update in mongo
	thingsTags["tags"].(map[string]interface{})[thingID] = t
	_, err = btt.UpdateOne(context.TODO(), filter, bson.M{"$set": thingsTags})
	if err != nil {
		return errors.Wrap(err, "unable to update BoardThingsTags")
	}

	// modify profile's tags in MySQL
	profile := model.Profile{ID: profileID}
	stmt := "SELECT IFNULL(tags, '') as tags FROM `sidekiq-dev`.AccountProfile WHERE id = ?"
	var profileTags string // comma separated string
	err = s.dbMaster.Conn.Get(&profileTags, stmt, profile.ID)
	if err != nil {
		return errors.Wrap(err, "unable to fetch profile tags from MySQL")
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
	_, err = s.dbMaster.Conn.NamedExec(updateStmt, profile)
	if err != nil {
		return errors.Wrap(err, "unable to update profile tags in MySQL")
	}

	return nil
}

func (s *service) AddFile(payload map[string]interface{}, profileID int) (map[string]interface{}, error) {
	dbconn, err := s.mongodb.New(consts.Board)
	if err != nil {
		return nil, err
	}

	boardCollection, boardClient := dbconn.Collection, dbconn.Client
	defer boardClient.Disconnect(context.TODO())

	profileStr := strconv.Itoa(profileID)
	isValid, err := permissions.CheckValidPermissions(profileStr, s.cache, boardCollection, payload["boardID"].(primitive.ObjectID).Hex(), []string{consts.Owner, consts.Admin, consts.Author}, false)
	if err != nil {
		return nil, err
	}
	var board model.Board
	if !isValid {
		return nil, errors.New("User does not have the access to the board!")
	}
	if payload["collectionID"] == primitive.NilObjectID {
		// find board filter
		filter := bson.M{"_id": payload["postID"]} // not adding isActive bit as if board is deleted, there won't be permission in redis, so would return error from role permissions

		err = boardCollection.FindOne(context.TODO(), filter).Decode(&board)
		if err != nil {
			return nil, errors.Wrap(err, "unable to find board")
		}
	}
	dbconn2, err := s.mongodb.New(consts.File)
	if err != nil {
		return nil, err
	}

	fileCollection, fileClient := dbconn2.Collection, dbconn2.Client
	defer fileClient.Disconnect(context.TODO())
	// get key
	boardOwnerInt, err := strconv.Atoi(board.Owner)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to int")
	}
	// get concise profile
	ownerInfo, err := s.FetchConciseProfile(boardOwnerInt)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch concise profile")
	}
	key := util.GetKeyForBoardMedia(ownerInfo["accountID"], ownerInfo["id"], payload["boardID"].(primitive.ObjectID).Hex(), "")
	url, err := s.fileStore.GetPresignedDownloadURL(key, fmt.Sprintf("%s%s", payload["_id"].(primitive.ObjectID).Hex(), filepath.Ext(payload["fileName"].(string))))
	if err != nil {
		return nil, err
	}

	// url := ""
	payload["createDate"] = time.Now()
	payload["modifiedDate"] = time.Now()
	payload["owner"] = profileStr
	payload["Type"] = consts.FileType
	payload["fileMime"] = mime.TypeByExtension(filepath.Ext(payload["FileExt"].(string)))
	payload["url"] = url
	payload["fileType"] = util.ReturnFileType(payload["FileMime"].(string))

	// adding owner info to payloadvar stmt string
	cp := &model.ConciseProfile{}
	stmt := `SELECT id, accountID, shareable, IFNULL(firstName, '') as firstName, IFNULL(lastName, '') as lastName, 
			IFNULL(screenName, '') AS screenName, 
			IFNULL(photo, '') AS photo FROM` + "`sidekiq-dev`.AccountProfile WHERE id = ?"
	err = s.dbMaster.Conn.Get(cp, stmt, profileID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		} else {
			return nil, errors.Wrap(err, "unable to find basic info")
		}
	}
	// fetching profile image
	var userID int
	stmt = `SELECT accountID FROM` + "`sidekiq-dev`.AccountProfile WHERE id = ?"
	err = s.dbMaster.Conn.Get(&userID, stmt, profileID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find user id for profile image ")
	}

	imageKey := util.GetKeyForProfileImage(userID, profileID, "")
	fileName := fmt.Sprintf("%d.png", profileID)
	fileData, err := s.GetUserFile(imageKey, fileName)
	if err != nil {
		cp.Photo = ""
		fmt.Println("unable to fetch profile picture", err)
	} else {
		cp.Photo = fileData.Filename
	}
	// payload.OwnerInfo = *cp
	// payload.OwnerInfo.Id = profileID
	_, err = fileCollection.InsertOne(context.TODO(), payload)
	if err != nil {
		return nil, errors.Wrap(err, "unable to insert file metadata at mongo")
	}

	if payload["collectionID"].(primitive.ObjectID) != primitive.NilObjectID {
		dbconn, err := s.mongodb.New(consts.Collection)
		if err != nil {
			return nil, err
		}

		collectionCollection, collectionClient := dbconn.Collection, dbconn.Client
		defer collectionClient.Disconnect(context.TODO())

		Things := []model.Things{
			{
				ThingID: payload["_id"].(primitive.ObjectID),
				Type:    payload["fileType"].(string),
				URL:     payload["url"].(string),
			},
		}

		// append thing in things array
		filter := bson.M{"_id": payload["collectionID"].(primitive.ObjectID)}
		update := bson.M{"$addToSet": bson.M{"things": Things[0]}}

		_, err = collectionCollection.UpdateOne(context.TODO(), filter, update)
		if err != nil {
			return nil, errors.Wrap(err, "unable to append things in collection")
		}

	}
	return util.SetResponse(payload, 1, "File meta data inserted successfully."), nil
}

func (s *service) GetCollectionCoverImage(collectionCollection *mongo.Collection, collectionID, boardID string) (*string, error) {
	coverImage := ""
	filter := bson.M{"_id": collectionID}
	var collection model.Collection
	err := collectionCollection.FindOne(context.TODO(), filter).Decode(&collection)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find collection")
	}
	if len(collection.Things) > 0 {
		dbconn, err := s.mongodb.New(consts.File)
		if err != nil {
			return nil, err
		}
		fileCollection, collectionClient := dbconn.Collection, dbconn.Client
		defer collectionClient.Disconnect(context.TODO())
		var file *model.UploadedFile
		for i := range collection.Things {
			if collection.Things[i].Type == "image" {
				filter := bson.M{"_id": collection.Things[i].ThingID}
				err = fileCollection.FindOne(context.TODO(), filter).Decode(&file)
				if err != nil {
					return nil, errors.Wrap(err, "unable to find collection")
				}
				key := ""
				// key := util.GetKeyForPostCollectionMedia(boardID, collection.PostID.Hex(), collectionID, "")
				fileName := fmt.Sprintf("%s%s", collection.Things[i].ThingID.Hex(), file.FileExt)
				fileData, err := s.GetUserFile(key, fileName)
				if err != nil {
					return nil, errors.Wrap(err, "error getting user file for profile image")
				}
				coverImage = fileData.Location
				break
			}
		}
	}
	return &coverImage, nil
}

func (s *service) AddFileInCollection(postOwner, key string, payload map[string]interface{}, profileID int) (map[string]interface{}, error) {
	dbconn, err := s.mongodb.New(consts.File)
	if err != nil {
		return nil, err
	}

	boardCollection, boardClient := dbconn.Collection, dbconn.Client
	defer boardClient.Disconnect(context.TODO())

	profileStr := strconv.Itoa(profileID)
	isValid, err := permissions.CheckValidPermissions(profileStr, s.cache, boardCollection, payload["boardID"].(primitive.ObjectID).Hex(), []string{"owner", "admin", "author"}, false)
	if err != nil {
		return nil, err
	}
	if !isValid {
		return nil, errors.New("User does not have the access to the board!")
	}

	dbconn2, err := s.mongodb.New(consts.File)
	if err != nil {
		return nil, err
	}

	fileCollection, fileClient := dbconn2.Collection, dbconn2.Client
	defer fileClient.Disconnect(context.TODO())

	// getting presigned URL from wasabi
	// key := util.GetKeyForCollectionMedia(payload.BoardID.Hex(), payload.CollectionID.Hex())
	url, err := s.fileStore.GetPresignedDownloadURL(key, fmt.Sprintf("%s%s", payload["_id"].(primitive.ObjectID).Hex(), filepath.Ext(payload["fileName"].(string))))
	if err != nil {
		return nil, err
	}

	// payload.CreateDate = time.Now()
	// payload.ModifiedDate = time.Now()
	// payload.Owner = profileStr
	// payload.Searchable = consts.Public
	// payload.Type = consts.FileType
	// payload.State = consts.Active
	// payload.FileMime = mime.TypeByExtension(filepath.Ext(payload.FileExt))
	// payload.URL = url

	// if payload.FileMime == "application/pdf" {
	// 	payload.FileType = "pdf"
	// } else if strings.Contains(payload.FileMime, "audio/") {
	// 	payload.FileType = "audio"
	// } else if strings.Contains(payload.FileMime, "video/") {
	// 	payload.FileType = "video"
	// } else if strings.Contains(payload.FileMime, "image/") {
	// 	payload.FileType = "image"
	// } else if payload.FileMime == "application/json" {
	// 	payload.FileType = "json"
	// }

	payload["createDate"] = time.Now()
	payload["modifiedDate"] = time.Now()
	payload["owner"] = profileStr
	payload["searchable"] = consts.Public
	payload["type"] = consts.FileType
	payload["state"] = consts.Active
	payload["fileMime"] = mime.TypeByExtension(filepath.Ext(payload["fileExt"].(string)))
	payload["url"] = url

	// Determine payload["fileType"] based on payload["fileMime"]
	if payload["fileMime"].(string) == "application/pdf" {
		payload["fileType"] = "pdf"
	} else if strings.Contains(payload["fileMime"].(string), "audio/") {
		payload["fileType"] = "audio"
	} else if strings.Contains(payload["fileMime"].(string), "video/") {
		payload["fileType"] = "video"
	} else if strings.Contains(payload["fileMime"].(string), "image/") {
		payload["fileType"] = "image"
	} else if payload["fileMime"].(string) == "application/json" {
		payload["fileType"] = "json"
	}

	_, err = fileCollection.InsertOne(context.TODO(), payload)
	if err != nil {
		return nil, errors.Wrap(err, "unable to insert file metadata at mongo")
	}

	dbconn3, err := s.mongodb.New(consts.Collection)
	if err != nil {
		return nil, err
	}

	collectionCollection, collectionClient := dbconn3.Collection, dbconn3.Client
	defer collectionClient.Disconnect(context.TODO())

	Things := []model.Things{
		{
			ThingID: payload["id"].(primitive.ObjectID),
			Type:    payload["fileType"].(string),
			URL:     payload["url"].(string),
		},
	}

	// append thing in things array
	filter := bson.M{"_id": payload["collectionID"].(primitive.ObjectID)}
	update := bson.M{"$addToSet": bson.M{"things": Things[0]}}

	_, err = collectionCollection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		return nil, errors.Wrap(err, "unable to append things in collection")
	}

	// getting collection's latest info
	var result model.Collection
	err = collectionCollection.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find collection")
	}

	return util.SetResponse(result, 1, "File meta data inserted successfully."), nil
}

func (s *service) GetUserFileList(key string) ([]*model.File, error) {
	return s.fileStore.GetFiles(key)
}

func (s *service) GetUserFile(key string, name string) (*model.File, error) {
	url, err := s.fileStore.GetPresignedDownloadURL(key, name)
	if err != nil {
		return nil, err
	} else if url != "" {
		return &model.File{
			Name:     name,
			Filename: url,
		}, nil
	}
	return s.fileStore.GetFile(key, name)
}

func (s *service) MoveFile(oldkey string, newKey string) error {
	err := s.fileStore.MoveFile(oldkey, newKey)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) GetFileParts(profile int, key, uuid string) ([]*model.FilePart, error) {
	return getFileParts(s.dbMaster, key, uuid)
}

func (s *service) GetTotalFileSize(profile int, key, uuid string) (*float64, error) {
	return getTotalFileSize(s.dbMaster, key, uuid)
}

func (s *service) UploadUserFile(postOwner, key, fileName string, file *model.File, meta *model.FileUpload,
	payload map[string]interface{}, complete chan *model.File,
	fmdChan chan map[string]interface{}, isDirect bool,
) (*model.File, error) {
	if isDirect {
		return s.fileStore.StoreFile(key, fileName, file)
	}

	totalBytes := []byte{}
	buffer := make([]byte, BufferSize)
	hash := md5.New()
	for {
		fmt.Println("--------------FORLOOPCALLER-----------------")
		bytesread, err := file.Reader.Read(buffer)
		if err != nil {
			if err != io.EOF {
				return nil, errors.Wrap(err, "Error Reading in File")
			}
			break
		}
		chunk := buffer[:bytesread]
		totalBytes = append(totalBytes, chunk...)
		hash.Write(chunk)
	}
	hashInBytes := hash.Sum(nil)
	etag := hex.EncodeToString(hashInBytes[:16])

	// file.ETag = "2bd7e1b2e89b226a112ac6e6fa3fe0f1" // don't hardcode

	if etag != file.ETag {
		fmdChan <- nil
		return nil, errors.New("Calculated md5 hash doesn't match the one provided")
	}

	file.Reader = bytes.NewReader(totalBytes)

	// single part file upload. Only for user, profile images and qr?
	if meta.TotalSize == int64(len(totalBytes)) {
		fmdChan <- nil
		return s.fileStore.StoreFile(key, fileName, file)
	}

	numDigits := 1
	fileSize := meta.TotalSize

	for fileSize != 0 {
		fileSize /= 10
		numDigits += 1
	}

	// multipart file upload
	part := &model.FilePart{
		Name:  fmt.Sprintf("%0*d_%s", numDigits, meta.Start, meta.Name),
		Size:  len(totalBytes),
		ETag:  file.ETag,
		Start: meta.Start,
	}
	file.Name = part.Name

	storedFile, err := s.tmpFileStore.StoreFile(key, fileName, file) // stores to local storage
	if err != nil {
		return nil, errors.Wrap(err, "Error storing file")
	}

	idInt, err := strconv.Atoi(postOwner)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to int")
	}
	_, err = insertFilePart(s.dbMaster, part, meta, key, idInt)
	if err != nil {
		return nil, errors.Wrap(err, "Error saving part to database")
	}

	parts, err := getFileParts(s.dbMaster, key, meta.UUID)
	if err != nil {
		return nil, errors.Wrap(err, "Error fetching file parts")
	}

	var totaSize int64 = 0
	for _, part := range parts {
		totaSize += int64(part.Size)
	}

	var res map[string]interface{}
	errChan := make(chan error)
	if totaSize == meta.TotalSize {
		go func(errChan chan<- error) {
			defer util.RecoverGoroutinePanic(nil)
			fmt.Println("----------------------------------------------------------------------------------------big bro called")
			result, err := uploadParts(key, fileName, s.tmpFileStore, s.fileStore, meta, parts)
			if err != nil {
				logrus.Error(errors.Wrap(err, "Error uploading file parts"))
				complete <- nil
				errChan <- errors.Wrap(err, "Error uploading file parts")
			}

			err = cleanupTempParts(s.tmpFileStore, s.dbMaster, meta, parts, key)
			if err != nil {
				logrus.Error(err)
			}
			payload["fileName"] = result.Name
			payload["fileSize"] = util.ConvertBytesToHumanReadable(result.Size)
			payload["storageLocation"] = result.Filename
			profile, _ := strconv.Atoi(postOwner)
			res, err = s.AddFile(payload, profile)
			if err != nil {
				logrus.Error(err)
			}

			// update profile tags
			// p := model.Profile{ID: profile}
			// s.UpdateProfileTags(p, res["data"].(model.UploadedFile).Tags)
			s.UpdateBoardThingsTags(profile,
				res["data"].(model.UploadedFile).BoardID.Hex(),
				res["data"].(model.UploadedFile).Id.Hex(), res["data"].(model.UploadedFile).Tags)
			if err != nil {
				logrus.Error(err)
			}
			fmdChan <- res
			complete <- result
			errChan <- nil
		}(errChan)
	} else {
		fmt.Println("------------------------------------------------------inside else")
		fmdChan <- nil
		complete <- nil
		go func() {
			defer util.RecoverGoroutinePanic(nil)
			errChan <- nil
		}()
	}
	if err := <-errChan; err != nil {
		return nil, errors.Wrap(err, "error from go routine")
	}
	fmt.Println("---------------273")
	return storedFile, nil
}

func (s *service) UploadUserFileInCollection(postOwner, key, fileName string, file *model.File, meta *model.FileUpload,
	payload map[string]interface{}, complete chan *model.File,
	fmdChan chan map[string]interface{}, isDirect bool,
) (*model.File, error) {
	fmt.Println("---------------509")
	if isDirect {
		return s.fileStore.StoreFile(key, fileName, file)
	}

	totalBytes := []byte{}
	buffer := make([]byte, BufferSize)
	hash := md5.New()
	for {
		fmt.Println("--------------FORLOOPCALLER-----------------")
		bytesread, err := file.Reader.Read(buffer)
		if err != nil {
			if err != io.EOF {
				return nil, errors.Wrap(err, "Error Reading in File")
			}
			break
		}
		chunk := buffer[:bytesread]
		totalBytes = append(totalBytes, chunk...)
		hash.Write(chunk)
	}
	hashInBytes := hash.Sum(nil)
	etag := hex.EncodeToString(hashInBytes[:16])

	// file.ETag = "2bd7e1b2e89b226a112ac6e6fa3fe0f1" // don't hardcode

	if etag != file.ETag {
		// etag not provided in headers, this channel was waiting for some data to listen
		fmdChan <- nil
		fmt.Println("Calculated md5 hash doesn't match the one provided")
		return nil, errors.New("Calculated md5 hash doesn't match the one provided")
	}

	file.Reader = bytes.NewReader(totalBytes)

	// single part file upload. Only for user, profile images and qr?
	if meta.TotalSize == int64(len(totalBytes)) {
		fmdChan <- nil
		return s.fileStore.StoreFile(key, fileName, file)
	}

	numDigits := 1
	fileSize := meta.TotalSize

	for fileSize != 0 {
		fileSize /= 10
		numDigits += 1
	}

	// multipart file upload
	part := &model.FilePart{
		Name:  fmt.Sprintf("%0*d_%s", numDigits, meta.Start, meta.Name),
		Size:  len(totalBytes),
		ETag:  file.ETag,
		Start: meta.Start,
	}
	file.Name = part.Name

	storedFile, err := s.tmpFileStore.StoreFile(key, fileName, file) // stores to local storage
	if err != nil {
		return nil, errors.Wrap(err, "Error storing file")
	}

	idInt, err := strconv.Atoi(postOwner)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to int")
	}
	_, err = insertFilePart(s.dbMaster, part, meta, key, idInt)
	if err != nil {
		return nil, errors.Wrap(err, "Error saving part to database")
	}

	parts, err := getFileParts(s.dbMaster, key, meta.UUID)
	if err != nil {
		return nil, errors.Wrap(err, "Error fetching file parts")
	}

	var totaSize int64 = 0
	for _, part := range parts {
		totaSize += int64(part.Size)
	}

	var res map[string]interface{}
	errChan := make(chan error)
	if totaSize == meta.TotalSize {
		go func(errChan chan<- error) {
			defer util.RecoverGoroutinePanic(nil)
			fmt.Println("----------------------------------------------------------------------------------------big bro called")
			result, err := uploadParts(key, fileName, s.tmpFileStore, s.fileStore, meta, parts)
			if err != nil {
				logrus.Error(errors.Wrap(err, "Error uploading file parts"))
				complete <- nil
				errChan <- errors.Wrap(err, "Error uploading file parts")
			}

			err = cleanupTempParts(s.tmpFileStore, s.dbMaster, meta, parts, key)
			if err != nil {
				logrus.Error(err)
			}

			payload["fileName"] = result.Name
			payload["fileSize"] = util.ConvertBytesToHumanReadable(result.Size)
			payload["storageLocation"] = result.Filename

			profile, _ := strconv.Atoi(postOwner)
			fmt.Println("reached here: 608 :", profile)

			res, err = s.AddFile(payload, profile)
			if err != nil {
				logrus.Error(err)
			}
			fmdChan <- res
			complete <- result
			errChan <- nil
		}(errChan)
	} else {
		fmt.Println("------------------------------------------------------inside else")
		fmdChan <- nil
		complete <- nil
		go func() {
			defer util.RecoverGoroutinePanic(nil)
			errChan <- nil
		}()
	}
	if err := <-errChan; err != nil {
		return nil, errors.Wrap(err, "error from go routine")
	}
	fmt.Println("---------------630")
	return storedFile, nil
}

func (s *service) PushToClients(profile int, file *model.File) error {
	subs, err := getPushSubscriptionsByProfile(s.dbMaster, profile)
	fmt.Println(strings.Repeat("*", 20))
	fmt.Println("subs: ", subs)
	if err != nil {
		return err
	}

	pushErrors := map[int]error{}
	for _, sub := range subs {
		subscription := sub.ToWebPush()
		notificationData := &notification.Message{
			Type:    "file",
			Content: file.ToJSON(),
		}
		fmt.Println("Push notf subscriber: ", sub.ProfileID)
		fmt.Println("notification data: ", notificationData.ToJSON())
		resp, err := webpush.SendNotification([]byte(notificationData.ToJSON()), subscription, &webpush.Options{
			Subscriber:      fmt.Sprintf("%d", sub.ProfileID),
			VAPIDPublicKey:  s.config.VapidPublicKey,
			VAPIDPrivateKey: s.config.VapidPrivateKey,
			TTL:             30,
		})
		fmt.Println(strings.Repeat("#", 45))
		fmt.Println("push notf subscription: ", subscription)
		fmt.Println("push notification response: ", resp)
		if err != nil {
			fmt.Println("push notf error: ", err)
			pushErrors[sub.ProfileID] = err
		}
		defer resp.Body.Close()
	}
	if len(pushErrors) > 0 {
		fmt.Println("push errors: ", pushErrors)
		return errors.New(fmt.Sprintf("Failed to send %d notifications", len(pushErrors)))
	}
	return nil
}

func (s *service) DeletePostMedia(ownerInfo *peoplerpc.ConciseProfileReply, boardID, mediaID, postID string) (map[string]interface{}, error) {
	dbconn, err := s.mongodb.New(consts.File)
	if err != nil {
		return nil, errors.Wrap(err, "unable to establish connection with File")
	}

	fileCollection, fileClient := dbconn.Collection, dbconn.Client
	defer fileClient.Disconnect(context.TODO())

	mediaObjID, _ := primitive.ObjectIDFromHex(mediaID)
	postObjId, _ := primitive.ObjectIDFromHex(postID)

	var ext map[string]interface{}
	opts := options.FindOne().SetProjection(
		bson.M{"fileExt": 1},
	)
	filter := bson.M{"_id": mediaObjID, "postID": postObjId}
	err = fileCollection.FindOne(context.TODO(), filter, opts).Decode(&ext)
	if err != nil {
		return nil, errors.Wrap(err, "unable to find the file meta data")
	}

	key := util.GetKeyForBoardPostMedia(int(ownerInfo.AccountID), int(ownerInfo.Id), boardID, postID, "")
	fileName := fmt.Sprintf("%s%s", mediaID, ext["fileExt"].(string))
	wasabiErr := s.fileStore.DeleteFile(key, fileName)

	// delete metadata from mongo
	_, err = fileCollection.DeleteOne(context.TODO(), filter)
	if wasabiErr != nil && err != nil {
		return util.SetResponse(nil, 0, "unable to delete file"), nil
	}
	return util.SetResponse(nil, 0, "File deleted successfully"), nil
}

func (s *service) DeleteBoardCover(boardID string) (map[string]interface{}, error) {
	var err error
	dbconn, err := s.mongodb.New(consts.Board)
	if err != nil {
		return nil, errors.Wrap(err, "unable to establish connection with File")
	}

	boardCollection, boardClient := dbconn.Collection, dbconn.Client
	defer boardClient.Disconnect(context.TODO())

	boardObjID, err := primitive.ObjectIDFromHex(boardID)
	if err != nil {
		return nil, errors.Wrap(err, "unable convert string to ObjectID")
	}

	var board model.Board
	err = boardCollection.FindOne(context.TODO(), bson.M{"_id": boardObjID}).Decode(&board)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch board")
	}

	boardOwnerInt, err := strconv.Atoi(board.Owner)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert string to int")
	}
	ownerInfo, err := s.FetchConciseProfile(boardOwnerInt)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch concise profile")
	}

	// delete og board cover from wasabi
	key := util.GetKeyForBoardCover(ownerInfo["accountID"], ownerInfo["id"], boardID, "")
	fileName := fmt.Sprintf("%s.png", boardID)
	fmt.Println(key)
	err = s.fileStore.DeleteFile(key, fileName)
	if err != nil {
		return nil, errors.Wrap(err, "unable to delete board cover")
	}

	// delete board cover thumbs
	thumbs := []string{"ic", "sm", "lg", "md"}
	goRoutines := 0
	errChan := make(chan error)
	for _, th := range thumbs {
		goRoutines += 1
		go func(th string, errChan chan<- error) {
			k := util.GetKeyForBoardCover(ownerInfo["accountID"], ownerInfo["id"], boardID, th)
			err = s.fileStore.DeleteFile(k, fileName)
			if err != nil {
				errChan <- errors.Wrap(err, "unable to delete board cover thumb")
			}
			errChan <- nil
		}(th, errChan)
	}

	for goRoutines != 0 {
		if err := <-errChan; err != nil {
			return nil, errors.Wrap(err, "error from go routine")
		}
		goRoutines--
	}

	return util.SetResponse(nil, 1, "Board cover removed."), nil
}

func (s *service) DeleteTempMedia(key, fileName string) (map[string]interface{}, error) {
	err := s.fileStore.DeleteFile(key, fileName)
	if err != nil {
		return nil, errors.Wrap(err, "unable to delete temp file from wasabi")
	}
	return util.SetResponse(nil, 1, "Temp file deleted"), nil
}

func (s *service) ComputeCloudStorage(prefix string) (map[string]interface{}, error) {
	// totalSpace := model.FileStorageSpace{}
	// Computing total storage at user level

	userStorage, err := s.fileStore.GetUserStorage(prefix)
	if err != nil {
		return nil, errors.Wrap(err, "error computing user storage")
	}

	return util.SetResponse(userStorage, 1, "user storage computed successfully"), nil
}

func (s *service) RemoveMediaChunks(profileID int) (map[string]interface{}, error) {
	stmt := "DELETE FROM `sidekiq-dev`.FileParts WHERE profileID = ?"
	_, err := s.dbMaster.Conn.Exec(stmt, profileID)
	if err != nil {
		return nil, errors.Wrap(err, "unable to delete file parts based of profileID")
	}
	return util.SetResponse(nil, 1, "Media chunks removed successfully"), nil
}

func (s *service) UpdateFileById(fileID primitive.ObjectID, payload map[string]interface{}) error {
	dbconn, err := s.mongodb.New(consts.File)
	if err != nil {
		return err
	}

	colCollection, colClient := dbconn.Collection, dbconn.Client
	defer colClient.Disconnect(context.TODO())

	_, err = colCollection.UpdateOne(context.TODO(), bson.M{"_id": fileID}, bson.M{"$set": payload})
	if err != nil {
		return err
	}

	return nil
}
