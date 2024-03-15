package storage

import (
	"fmt"
	"log"
	"mime"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ProImaging/sidekiq-backend/sidekiq-models/model"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pkg/errors"
)

// WasabiStorage - storage adapter for storing files to wasabi
type WasabiStorage struct {
	Session *session.Session
	S3      *s3.S3
	Bucket  string
}

// WasabiStorage - creates a new wasbi storage adapter
func NewWasabiStorage(bucket string, region string, accessKeyID string, secretAccessKey string) (*WasabiStorage, error) {
	sess, err := session.NewSession(&aws.Config{
		Endpoint:    aws.String(fmt.Sprintf("https://s3.%s.wasabisys.com", region)),
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""),
	})
	if err != nil {
		return nil, err
	}
	return &WasabiStorage{
		Session: sess,
		S3:      s3.New(sess),
		Bucket:  bucket,
	}, nil
}

func (s *WasabiStorage) GetFiles(key string) ([]*model.File, error) {
	// fmt.Println("get files from wasabi ", boardId)
	// prefix := fmt.Sprintf("profile/%d/%s/", profile, boardId)
	result, err := s.S3.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: &s.Bucket,
		Prefix: &key,
	})
	if err != nil {
		return nil, err
	}
	files := []*model.File{}
	for i := 0; i < len(result.Contents); i++ {
		obj := result.Contents[i]
		f := &model.File{
			Filename: *obj.Key,
			Name:     filepath.Base(*obj.Key),
			Size:     *obj.Size,
		}
		f.Filename = f.Name
		f.Type = mime.TypeByExtension(strings.ToLower(filepath.Ext(f.Name)))
		files = append(files, f)
	}
	fmt.Println()
	// fmt.Printf("Files of %d in %s: \n", profile, boardId)
	for _, file := range files {
		fmt.Println(file)
	}
	fmt.Println()
	return files, nil
}

func (s *WasabiStorage) GetFile(key string, path string) (*model.File, error) {
	// fmt.Println("GET FILE: ", key)
	result, err := s.S3.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		// Check if the error is due to "NotFound"
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == s3.ErrCodeNoSuchKey {
			// Handle NotFound error
			fmt.Println("File not found for the path", key)

		} else {
			// Handle other errors
			fmt.Println("error in fetching S3 object:", err)
		}
		return &model.File{
			Name:     "",
			Filename: "",
		}, nil
	} else {
		fmt.Println("File found")
	}
	f := &model.File{
		Name:   filepath.Base(key),
		Size:   *result.ContentLength,
		Type:   *result.ContentType,
		Reader: result.Body,
	}
	return f, nil
}

func (s *WasabiStorage) StoreFile(key, fileName string, file *model.File) (*model.File, error) {
	fullPath := fmt.Sprintf("%s%s", key, fileName)
	fmt.Println("STORE FILE: ", fullPath)
	fmt.Println("-------------CONTENT TYPE-----------", file.Type)
	uploader := s3manager.NewUploaderWithClient(s.S3)
	upParams := &s3manager.UploadInput{
		Bucket:             &s.Bucket,
		Key:                &fullPath,
		Body:               file.Reader,
		ContentType:        &file.Type,
		ContentDisposition: aws.String("inline"),
	}
	start := time.Now()
	result, err := uploader.Upload(upParams)
	if err != nil {
		return nil, err
	}
	elapsed := time.Since(start)
	fmt.Println("Time took to upload file from server to wasabi", elapsed)
	return &model.File{
		Filename: result.Location,
		Name:     file.Name,
		Size:     file.Size,
		Type:     file.Type,
		ETag:     file.ETag,
	}, nil
}

func (s *WasabiStorage) DeleteFile(key, fileName string) error {
	fullPath := fmt.Sprintf("%s%s", key, fileName)
	fmt.Println("Deleting file from wasabi: ", fullPath)

	_, err := s.S3.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(fullPath),
	})
	if err != nil {
		return nil
	}

	err = s.S3.WaitUntilObjectNotExists(&s3.HeadObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(fullPath),
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *WasabiStorage) MakeFolder(key string, userID int, path string) (string, error) {
	// TODO: Implement This
	return "", nil
}

func (s *WasabiStorage) MoveFile(oldKey string, newKey string) error {
	src := "s3://sidekiq/" + oldKey
	dst := "s3://sidekiq/" + newKey
	fmt.Println("src", src)
	fmt.Println("dst", dst)
	endpointurl := "https://s3.us-east-2.wasabisys.com"

	// Construct the modified AWS CLI command
	cmd := exec.Command("aws", "s3", "mv", src, dst, "--endpoint", endpointurl, "--recursive")
	cmd.Env = append(os.Environ(), "AWS_ACCESS_KEY_ID=72I5N34INHRZFSNE9U3Q", "AWS_SECRET_ACCESS_KEY=pQIWZEoQQK38SjzCwh8slsacDVT6DkWlBmeswgTe", "AWS_REGION=us-east-2", "AWS_CONFIG_FILE= ~/.aws/config")

	fmt.Println(cmd.String())

	out, err := cmd.CombinedOutput()
	if err != nil {
		exitError, ok := err.(*exec.ExitError)
		if ok && !exitError.Success() {
			log.Printf("aws command exited with status: %d\n", exitError.ExitCode())
			log.Printf("stderr output:\n%s\n", string(exitError.Stderr))
			return errors.Wrap(err, "error in moving file")
		} else {
			log.Fatalf("Error executing aws command: %v", err)
			return errors.Wrap(err, "error in moving file")
		}
	}

	fmt.Println("OUTPUT STRING", string(out))

	return nil
}

func (s *WasabiStorage) GetPresignedDownloadURL(key, name string) (string, error) {
	// Combine the key and name to form the full object key
	objectKey := key + name
	// fmt.Println("Object Key", objectKey)
	// Call the HeadObject API operation to check if the object exists
	_, err := s.S3.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(objectKey),
	})

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NotFound" {
			fmt.Printf(`Path "%s" not found`+"\n", objectKey)
			return "", nil
		} else {
			fmt.Printf(`Unable to fetch presigned URL for path %s`+"\n", objectKey)
			return "", err
		}
	} else {
		// Object exists
		// fmt.Println("Object exists")
	}

	parts := strings.Split(name, ".")
	extension := parts[len(parts)-1]
	var req *request.Request
	if extension == "pdf" {
		req, _ = s.S3.GetObjectRequest(&s3.GetObjectInput{
			Bucket:                     aws.String(s.Bucket),
			Key:                        aws.String(objectKey),
			ResponseContentDisposition: aws.String("inline"),
			ResponseContentType:        aws.String("application/octet-stream"),
		})
	} else {
		req, _ = s.S3.GetObjectRequest(&s3.GetObjectInput{
			Bucket: aws.String(s.Bucket),
			Key:    aws.String(objectKey),
		})
	}

	// Generate the pre-signed URL with an expiration time
	preSignedURL, err := req.Presign(150 * time.Hour)
	if err != nil {
		fmt.Printf("Failed to generate pre-signed URL: %v\n", err)
		return "", err
	}

	if strings.Contains(preSignedURL, `\u0026`) {
		preSignedURL = strings.ReplaceAll(preSignedURL, `\u0026`, "&")
	}

	return preSignedURL, nil
}

// GetFilesSize - get size of all files for given key
func (s *WasabiStorage) GetUserStorage(key string) (*model.FileStorageSpace, error) {
	size := model.FileStorageSpace{}
	// Set up variables to store total size and object count
	totalSize := int64(0)
	totalCount := int64(0)

	// Call ListObjectsV2 to retrieve all objects with the given prefix
	err := s.S3.ListObjectsV2Pages(&s3.ListObjectsV2Input{
		Bucket: aws.String(s.Bucket),
		Prefix: aws.String(key),
	}, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		// For each page of results, add up the sizes of all objects
		for _, obj := range page.Contents {
			totalSize += *obj.Size
			totalCount++
		}
		return true
	})
	if err != nil {
		return nil, err
	}

	totalSizeGB := float64(totalSize) / (1024 * 1024 * 1024)
	totalSizeMB := float64(totalSize) / (1024 * 1024)

	size.GB = totalSizeGB
	size.MB = totalSizeMB
	return &size, nil
}
