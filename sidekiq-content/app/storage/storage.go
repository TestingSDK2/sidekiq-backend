package storage

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/database"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/util"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-models/model"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-models/model/notification"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func deleteFilePart(db *database.Database, filePart *model.FilePart, key string) (int, error) {
	stmt := "DELETE FROM `sidekiq-dev`.FileParts WHERE name = ? AND awsKey = ?"
	_, err := db.Conn.Exec(stmt, filePart.Name, key)
	if err != nil {
		return 0, errors.Wrap(err, "unable to delete file part from database")
	}
	return 1, nil
}

func insertFilePart(db *database.Database, filePart *model.FilePart, meta *model.FileUpload, key string, profileID int) (int, error) {
	stmt := "INSERT INTO `sidekiq-dev`.FileParts (profileID, awsKey, name, type, uuid, etag, start, size, totalSize) VALUES(:profileID,:awsKey,:name,:type,:uuid,:etag,:start,:size,:totalSize);"

	val := map[string]interface{}{
		"profileID": profileID,
		"awsKey":    key,
		"uuid":      meta.UUID,
		"type":      meta.Type,
		"name":      filePart.Name,
		"etag":      filePart.ETag,
		"start":     filePart.Start,
		"size":      filePart.Size,
		"totalSize": meta.TotalSize,
	}

	r, err := db.Conn.NamedExec(stmt, val)
	if err != nil {
		return 0, err
	}
	id, err := r.LastInsertId()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to get LastInsertId")
	}
	return int(id), nil
}

func getFileParts(db *database.Database, key, uuid string) ([]*model.FilePart, error) {
	stmt := "SELECT name, etag, start, size FROM `sidekiq-dev`.FileParts WHERE awsKey = ? AND uuid = ? ORDER BY start ASC;"
	fmt.Println("fetching file parts for : ", key)

	var parts []*model.FilePart
	err := db.Conn.Select(&parts, stmt, key, uuid)
	if err != nil {
		return nil, err
	}

	return parts, nil
}

func getTotalFileSize(db *database.Database, key, uuid string) (*float64, error) {
	stmt := "SELECT totalSize FROM `sidekiq-dev`.FileParts WHERE awsKey = ? AND uuid = ? ORDER BY start ASC;"
	fmt.Println("fetching file parts for : ", key)
	fmt.Println("key: ", key)
	fmt.Println("uuid: ", uuid)
	var totalSize *float64
	err := db.Conn.Select(&totalSize, stmt, key, uuid)
	if err != nil {
		return nil, err
	}
	return totalSize, nil
}

func getPushSubscriptionsByProfile(db *database.Database, profile int) ([]*notification.PushSubscription, error) {
	fmt.Println("push to client: ", profile)
	stmt := "SELECT id, profileID, type, endpoint, p256dh, auth, expirationTime, createDate FROM `sidekiq-dev`.PushSubscriptions WHERE profileID = ?;"

	subs := []*notification.PushSubscription{}
	err := db.Conn.Select(&subs, stmt, profile)
	if err != nil {
		return nil, err
	}
	return subs, nil
}

func getPushSubscriptionsByUser(db *database.Database, profileID int) ([]*notification.PushSubscription, error) {
	fmt.Println("push to client: ", profileID)
	stmt := "SELECT id, profileID, type, endpoint, p256dh, auth, expirationTime, createDate FROM `sidekiq-dev`.PushSubscriptions WHERE profileID = ?;"

	subs := []*notification.PushSubscription{}
	err := db.Conn.Select(&subs, stmt, profileID)
	if err != nil {
		return nil, err
	}
	return subs, nil
}

func uploadParts(key, fileName string, tmpStore model.FileStorage, fileStore model.FileStorage, meta *model.FileUpload, parts []*model.FilePart) (*model.File, error) {
	pr, pw := io.Pipe()
	hash := md5.New()
	var totalBytes int64 = 0
	var readErr error = nil
	go func() {
		defer util.RecoverGoroutinePanic(nil)
		defer pw.Close()
		for _, part := range parts {
			var f *model.File
			f, readErr = tmpStore.GetFile(key, part.Name)
			if readErr != nil {
				return
			}

			buffer := make([]byte, BufferSize)
			for {
				bytesread, err := f.Reader.Read(buffer)
				if err != nil {
					if err != io.EOF {
						logrus.Error(errors.Wrap(err, "Error Reading in File"))
						return
					}
					break
				}
				totalBytes += int64(bytesread)
				chunk := buffer[:bytesread]
				hash.Write(chunk)
				pw.Write(chunk)
			}
			// written, _ := io.Copy(pw, f.Reader)
			// totalBytes += written
		}
	}()

	outFile := &model.File{
		Name:   meta.Name,
		Type:   meta.Type,
		Reader: pr,
	}
	fmt.Println("meta.Profile: ", meta.Profile)
	_, err := fileStore.StoreFile(key, fileName, outFile)
	if readErr != nil {
		return nil, readErr
	}
	hashInBytes := hash.Sum(nil)
	outFile.ETag = hex.EncodeToString(hashInBytes[:16])
	outFile.Size = totalBytes
	return outFile, err
}

func cleanupTempParts(tmpStore model.FileStorage, db *database.Database, meta *model.FileUpload, parts []*model.FilePart, key string) error {
	failed := []*model.FilePart{}
	fmt.Println("cleaning up temp parts: ", key)
	for _, part := range parts {
		err := tmpStore.DeleteFile(key, part.Name)
		if err != nil {
			// logrus.Error(errors.Wrap(err, fmt.Sprintf("Error Deleting file: %s", part.Name)))
			failed = append(failed, part)
		}
		// delete file part from mysql
		_, err = deleteFilePart(db, part, key)
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			return errors.Wrap(err, "unable to delete file part from mysql")
		}
	}
	if len(failed) > 0 {
		return errors.New(fmt.Sprintf("Failed to delete %d temp file parts for %s", len(failed), meta.Name))
	}
	return nil
}
