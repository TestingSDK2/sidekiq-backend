package storage

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-models/model"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-people/app/config"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-people/cache"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-people/database"
	repo "github.com/TestingSDK2/sidekiq-backend/sidekiq-people/model"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-people/mongodatabase"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-people/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const BufferSize = 1024 * 1024

type Service interface {
	GetUserFile(key string, name string) (*model.File, error)
	UploadUserFile(postOwner, key, fileName string, file *model.File, meta *model.FileUpload, payload map[string]interface{}, complete chan *model.File, fmdChan chan map[string]interface{}, isDirect bool) (*model.File, error)
	DeleteTempMedia(key, fileName string) (map[string]interface{}, error)
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

			// update profile tags
			// p := model.Profile{ID: profile}
			// s.UpdateProfileTags(p, res["data"].(model.UploadedFile).Tags)

			//GRPC METHOD TO UPDATE TAGS UpdateBoardThingsTags
			// s.UpdateBoardThingsTags(profile,
			// 	res["data"].(model.UploadedFile).BoardID.Hex(),
			// 	res["data"].(model.UploadedFile).Id.Hex(), res["data"].(model.UploadedFile).Tags)
			if err != nil {
				logrus.Error(err)
			}
			fmdChan <- payload
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

func (s *service) DeleteTempMedia(key, fileName string) (map[string]interface{}, error) {
	err := s.fileStore.DeleteFile(key, fileName)
	if err != nil {
		return nil, errors.Wrap(err, "unable to delete temp file from wasabi")
	}
	return util.SetResponse(nil, 1, "Temp file deleted"), nil
}
