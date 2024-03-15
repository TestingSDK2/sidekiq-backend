package storage

import (
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"os"
	"path"
	"path/filepath"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-models/model"
)

// LocalStorage - storage adapter for storing files to local filesystem
type LocalStorage struct {
	Root string
}

// NewLocalStorage - creates a new local storage adapter
func NewLocalStorage(rootPath string) (*LocalStorage, error) {
	return &LocalStorage{
		Root: rootPath,
	}, nil
}

func (s *LocalStorage) MoveFile(oldkey string, newKey string) error {
	return nil
}

// GetFiles - get files for given user
func (s *LocalStorage) GetFiles(key string) ([]*model.File, error) {
	folder := fmt.Sprintf("%s/%s", s.Root, key)
	fmt.Println()
	fmt.Println("reading files from the folder @ local.go:32: ", folder)
	fmt.Println()
	iFiles, err := ioutil.ReadDir(folder)
	if err != nil {
		return []*model.File{}, err
	}
	files := []*model.File{}
	for i := 0; i < len(iFiles); i++ {
		f := &model.File{
			Name: iFiles[i].Name(),
			Size: iFiles[i].Size(),
		}
		f.Filename = f.Name
		f.Type = mime.TypeByExtension(filepath.Ext(f.Name))
		files = append(files, f)
	}
	fmt.Println()
	fmt.Println("files @ storage/local.go:46 ", files)
	fmt.Println()

	return files, nil
}

// GetFile - get requested file
func (s *LocalStorage) GetFile(key string, name string) (*model.File, error) {
	filePath := fmt.Sprintf("%s/%s/%s", s.Root, key, name)
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}
	f := &model.File{
		Name: fileInfo.Name(),
		Size: fileInfo.Size(),
	}
	f.Filename = f.Name
	f.Type = mime.TypeByExtension(filepath.Ext(f.Name))
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	f.Reader = file
	return f, nil
}

// StoreFile - get files for given user
func (s *LocalStorage) StoreFile(key, fileName string, file *model.File) (*model.File, error) {
	fmt.Println("reached StoreFile(local)....")
	folder := fmt.Sprintf("%s/%s", s.Root, key)
	fmt.Println(key)
	fmt.Println(folder)
	s.MakeFolder(key, 0, folder)
	outFile, err := os.Create(filepath.Join(folder, file.Name))
	if err != nil {
		return nil, err
	}
	defer outFile.Close()

	// copy from reader data into writer file
	size, err := io.Copy(outFile, file.Reader)
	if err != nil {
		return nil, err
	}

	fmt.Println("file details: ", &model.File{
		Name: filepath.Base(outFile.Name()),
		Size: size,
		Type: file.Type,
	})

	return &model.File{
		Name: filepath.Base(outFile.Name()),
		Size: size,
		Type: file.Type,
	}, nil
}

// DeleteFile - delete files for user
func (s *LocalStorage) DeleteFile(key, fileName string) error {
	folder := fmt.Sprintf("%s/%s", s.Root, key)
	path := filepath.Join(folder, fileName)
	if fileExists(path) {
		return os.Remove(path)
	}
	return nil
}

func (s *LocalStorage) GetPresignedDownloadURL(key string, name string) (string, error) {
	// TODO: Implement This
	return "", nil
}

func (s *LocalStorage) MakeFolder(key string, id int, folderPath string) (string, error) {
	folder := path.Join(s.Root, key)
	if folderPath != "" {
		// folder = fmt.Sprintf("%s%s", folder, folderPath)
		folder = fmt.Sprintf("%s", folderPath)
	}
	if _, err := os.Stat(folder); os.IsNotExist(err) {
		err := os.MkdirAll(folder, 0o755)
		if err != nil {
			return "", err
		}
	}
	return folder, nil
}

func getFilesInFolder(path string) ([]os.FileInfo, error) {
	return ioutil.ReadDir(path)
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// GetFilesSize - get size of all files for given key
func (s *LocalStorage) GetUserStorage(key string) (*model.FileStorageSpace, error) {
	// TODO: Implement This
	return &model.FileStorageSpace{}, nil
}
