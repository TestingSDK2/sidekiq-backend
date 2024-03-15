package helper

import (
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/storage"
	model "github.com/TestingSDK2/sidekiq-backend/sidekiq-models"
	"github.com/sirupsen/logrus"
)

func GetThumbnails(storageService storage.Service, thumbKey, thumbFileName string, thumbTypes []string) (model.Thumbnails, error) {
	if len(thumbTypes) == 0 {
		thumbTypes = []string{"sm", "md", "lg", "ic"}
	}

	// var s model.Thu
	var thumbs model.Thumbnails
	for i := range thumbTypes {
		finalkey := thumbKey + thumbTypes[i] + "/"
		ThumbfileData, err := storageService.GetUserFile(finalkey, thumbFileName)
		if err != nil {
			logrus.Error(err, "unable to presign thumbnails")
			continue
		}
		if thumbTypes[i] == "sm" {
			thumbs.Small = ThumbfileData.Filename
		} else if thumbTypes[i] == "md" {
			thumbs.Medium = ThumbfileData.Filename
		} else if thumbTypes[i] == "lg" {
			thumbs.Large = ThumbfileData.Filename
		} else if thumbTypes[i] == "ic" {
			thumbs.Icon = ThumbfileData.Filename
		}
	}

	return thumbs, nil
}
