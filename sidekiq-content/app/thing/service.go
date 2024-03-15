package thing

import (
	"time"

	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/config"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/post"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/profile"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/storage"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/database"
	repo "github.com/ProImaging/sidekiq-backend/sidekiq-content/model"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/mongodatabase"
	"github.com/ProImaging/sidekiq-backend/sidekiq-models/model"

	// peoplerpc "github.com/sidekiq-people/proto/people"
	peoplerpc "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"
)

type Service interface {
	LikeThing(thingID, thingType string, profileID int) (map[string]interface{}, error)
	DislikeThing(thingID, thingType string, profileID int) (map[string]interface{}, error)
	AddPostComment(post *model.Post, profileID int, comment string) (map[string]interface{}, error)
	AddThingComment(thingID, thingType string, profileID int, comment string) (map[string]interface{}, bool, error)
	AddThingComment2(thingID, thingType string, profileID int, comment string) (map[string]interface{}, error)
	GetPostDetailsByThingID(thingID, thingType string) (*model.Post, error)
	DeleteComment(thingID, thingType, commentID string, profileID int) (map[string]interface{}, error)
	EditComment(thingID, thingType, commentID string, payload map[string]string, profileID int) (map[string]interface{}, error)
	FetchBookmarks(userID, profileID, limit, page int, sortBy, orderBy, filterByThing string) (map[string]interface{}, error)
	AddBookmark(payload model.Bookmark) (map[string]interface{}, error)
	DeleteBookmark(profileID int, thingID string) (map[string]interface{}, error)
	FlagBookmarkForDelete(profileID int, thingID string, date time.Time) (map[string]interface{}, error)
	DeleteAllBookmarks(profileID int) (map[string]interface{}, error)
	FetchReactions(thingID, thingType, reactionType string, profileID, limit, page int) (map[string]interface{}, error)
	IsBookMarkedByProfile(thingID string, profile int) (bool, string, error)
	GetThingBasedOffIDAndType(thingID, thingType string) (map[string]interface{}, error)
}

type service struct {
	config         *config.Config
	mongodb        *mongodatabase.DBConfig
	dbMaster       *database.Database
	peopleRpc      peoplerpc.AccountServiceClient
	postService    post.Service
	storageService storage.Service
	profileService profile.Service
}

// NewService - creates new thing service
func NewService(repos *repo.Repos, conf *config.Config) Service {
	return &service{
		config:         conf,
		mongodb:        repos.MongoDB,
		dbMaster:       repos.MasterDB,
		profileService: profile.NewService(repos, conf),
		postService:    post.NewService(repos, conf),
		storageService: storage.NewService(repos, conf),
		peopleRpc:      repos.PeopleGrpcServiceClient,
	}
}

func (s *service) LikeThing(thingID, thingType string, profileID int) (map[string]interface{}, error) {
	return likeThing(s.mongodb, thingID, thingType, profileID)
}

func (s *service) DislikeThing(thingID, thingType string, profileID int) (map[string]interface{}, error) {
	return dislikeThing(s.mongodb, thingID, thingType, profileID)
}

func (s *service) AddThingComment(thingID, thingType string, profileID int, comment string) (map[string]interface{}, bool, error) {
	return addThingComment(s.mongodb, thingID, thingType, profileID, comment)
}

func (s *service) AddThingComment2(thingID, thingType string, profileID int, comment string) (map[string]interface{}, error) {
	return addThingComment2(s.mongodb, thingID, thingType, profileID, comment)
}

func (s *service) AddPostComment(post *model.Post, profileID int, comment string) (map[string]interface{}, error) {
	return addPostComment(s.mongodb, post, profileID, comment)
}

func (s *service) DeleteComment(thingID, thingType, commentID string, profileID int) (map[string]interface{}, error) {
	return deleteComment(s.mongodb, thingID, thingType, commentID, profileID)
}

func (s *service) EditComment(thingID, thingType, commentID string, payload map[string]string, profileID int) (map[string]interface{}, error) {
	return editComment(s.mongodb, thingID, thingType, commentID, payload, profileID)
}

func (s *service) FetchBookmarks(userID, profileID, limit, page int, sortBy, orderBy, filterByThing string) (map[string]interface{}, error) {
	return fetchBookmarks(s.postService, s.storageService, s.peopleRpc, s.dbMaster, s.mongodb, userID, profileID, limit, page, sortBy, orderBy, filterByThing)
}

func (s *service) AddBookmark(payload model.Bookmark) (map[string]interface{}, error) {
	return addBookmark(s.dbMaster, s.mongodb, payload)
}

func (s *service) DeleteBookmark(profileID int, thingID string) (map[string]interface{}, error) {
	return deleteBookmark(s.mongodb, profileID, thingID)
}

func (s *service) FlagBookmarkForDelete(profileID int, thingID string, date time.Time) (map[string]interface{}, error) {
	return flagBookmarkForDelete(s.mongodb, profileID, thingID, date)
}

func (s *service) DeleteAllBookmarks(profileID int) (map[string]interface{}, error) {
	return deleteAllBookmarks(s.mongodb, profileID)
}

func (s *service) FetchReactions(thingID, thingType, reactionType string, profileID, limit, page int) (map[string]interface{}, error) {
	return fetchReactions(s.mongodb, s.peopleRpc, s.storageService, thingID, thingType, reactionType, profileID, limit, page)
}

func (s *service) IsBookMarkedByProfile(thingID string, profile int) (bool, string, error) {
	return isBookmarkedByProfile(s.mongodb, thingID, profile)
}

func (s *service) GetPostDetailsByThingID(thingID, thingType string) (*model.Post, error) {
	return getPostDetailsByThingID(s.mongodb, thingID, thingType)
}

func (s *service) GetThingBasedOffIDAndType(thingID, thingType string) (map[string]interface{}, error) {
	return getThingBasedOffIDAndType(s.mongodb, thingID, thingType)
}
