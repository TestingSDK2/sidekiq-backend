package model

import (
	"encoding/json"
	"io"
)

// FileStorage - interface for backend data store adapters
type FileStorage interface {
	GetFiles(key string) ([]*File, error)
	GetFile(key string, name string) (*File, error)
	StoreFile(key, fileName string, file *File) (*File, error)
	DeleteFile(key, name string) error
	MakeFolder(key string, id int, path string) (string, error)
	GetPresignedDownloadURL(key, name string) (string, error)
	MoveFile(oldkey string, newKey string) error
	GetUserStorage(key string) (*FileStorageSpace, error)
}

// File container to hold handles for cache / db repos
type File struct {
	Name     string    `json:"name"`
	Filename string    `json:"-"`
	Type     string    `json:"type"`
	Size     int64     `json:"size"`
	ETag     string    `json:"etag"`
	Location string    `json:"location"`
	Reader   io.Reader `json:"-"`
	Writer   io.Writer `json:"-"`
}

type FileStorageSpace struct {
	MB float64 `json:"mb"`
	GB float64 `json:"gb"`
}
type FileUpload struct {
	ID        string `json:"id" db:"id"`
	UserID    int    `json:"userID" db:"userID"`
	Profile   int    `json:"Profile" db:"Profile"`
	BoardID   string `json:"boardID" db:"boardID"`
	Name      string `json:"name" db:"name"`
	Type      string `json:"type" db:"type"`
	UUID      string `json:"uuid" db:"uuid"`
	Start     int    `json:"start" db:"start"`
	TotalSize int64  `json:"totalSize" db:"totalSize"`
}

type FilePart struct {
	Name  string `json:"name" db:"name"`
	Start int    `json:"start" db:"start"`
	Size  int    `json:"size" db:"size"`
	ETag  string `json:"etag" db:"etag"`
}

// type FileMetaData struct {
// 	ID           string `json:"id" db:"id"`
// 	UserId       int    `json:"userId" db:"userId"`
// 	Name         string `json:"fileName" db:"fileName"`
// 	SizeBytes    int64  `json:"sizeBytes" db:"sizeBytes"`
// 	Size         string `json:"fileSize" db:"fileSize"`
// 	Type         string `json:"fileType" db:"fileType"`
// 	UploadStatus string `json:"uploadStatus" db:"uploadStatus"`
// }

// ToJSON converts message to json string
func (f *File) ToJSON() string {
	json, _ := json.Marshal(f)
	return string(json)
}

func (f *FileUpload) ToJSON() string {
	json, _ := json.Marshal(f)
	return string(json)
}
