package note

import (
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/board"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/config"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/profile"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/storage"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/cache"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/database"
	repo "github.com/ProImaging/sidekiq-backend/sidekiq-content/model"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/mongodatabase"
	"github.com/ProImaging/sidekiq-backend/sidekiq-models/model"

	// peoplerpc "github.com/sidekiq-people/proto/people"
	peoplerpc "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Service interface {
	FetchNotesByBoard(boardID string, profileID int, owner string, tagArr []string, uploadDate string, limit int, l, page string) (map[string]interface{}, error)
	AddNote(postID primitive.ObjectID, note interface{}) (map[string]interface{}, error)
	AddNotes(postID primitive.ObjectID, notes []interface{}) error
	AddNoteInCollection(note model.Note, collectionID string, profileID int) (map[string]interface{}, error)
	UpdateNote(note map[string]interface{}, boardID, postID, noteID string, profileID int) (map[string]interface{}, error)
	DeleteNote(boardID, noteID string, profileID int) (map[string]interface{}, error)
	GetNoteByID(noteID string, profileID int) (map[string]interface{}, error)
	FetchNotesByProfile(boardID string, profileID, limit int, publicOnly bool) (map[string]interface{}, error)
	FetchNotesByPost(boardID, postID string) ([]map[string]interface{}, error)
	FetchCollectionByPost(boardID, postID string, profileID int) ([]map[string]interface{}, error)
	DeleteNotesOnPost(postID string) error
}

type service struct {
	config         *config.Config
	mongodb        *mongodatabase.DBConfig
	dbMaster       *database.Database
	dbReplica      *database.Database
	boardService   board.Service
	profileService profile.Service
	storageService storage.Service
	cache          *cache.Cache
	peopleRpc      peoplerpc.AccountServiceClient
}

// NewService - creates new File service
func NewService(repos *repo.Repos, conf *config.Config) Service {
	return &service{
		config:         conf,
		mongodb:        repos.MongoDB,
		dbMaster:       repos.MasterDB,
		dbReplica:      repos.ReplicaDB,
		boardService:   board.NewService(repos, conf),
		profileService: profile.NewService(repos, conf),
		storageService: storage.NewService(repos, conf),
		cache:          repos.Cache,
		peopleRpc:      repos.PeopleGrpcServiceClient,
	}
}

func (s *service) FetchNotesByBoard(boardID string, profileID int, owner string, tagArr []string, uploadDate string, limit int, l, page string,
) (map[string]interface{}, error) {
	return getNotesByBoard(s.mongodb, s.dbMaster, s.cache, s.boardService, s.peopleRpc, s.storageService, boardID, profileID, owner, tagArr, uploadDate, limit, l, page)
}

func (s *service) AddNotes(postID primitive.ObjectID, notes []interface{}) error {
	return addNotes(s.mongodb, postID, notes)
}

func (s *service) AddNote(postID primitive.ObjectID, note interface{}) (map[string]interface{}, error) {
	return addNote(s.mongodb, postID, note)
}

func (s *service) AddNoteInCollection(note model.Note, collectionID string, profileID int) (map[string]interface{}, error) {
	return addNoteInCollection(s.cache, s.peopleRpc, s.storageService, s.mongodb, note, collectionID, profileID)
}

func (s *service) UpdateNote(note map[string]interface{}, boardID, postID, noteID string, profileID int) (map[string]interface{}, error) {
	return updateNote(s.cache, s.mongodb, note, boardID, postID, noteID, profileID)
}

func (s *service) DeleteNote(boardID, noteID string, profileID int) (map[string]interface{}, error) {
	return deleteNote(s.cache, s.mongodb, boardID, noteID, profileID)
}

func (s *service) GetNoteByID(noteID string, profileID int) (map[string]interface{}, error) {
	return getNoteByID(s.mongodb, s.profileService, s.storageService, noteID, profileID)
}

func (s *service) FetchNotesByProfile(boardID string, profileID, limit int, publicOnly bool) (map[string]interface{}, error) {
	return fetchNotesByProfile(s.cache, s.mongodb, s.dbMaster, boardID, profileID, limit, publicOnly)
}

func (s *service) FetchNotesByPost(boardID, postID string) ([]map[string]interface{}, error) {
	return fetchNotesByPost(s.mongodb, boardID, postID)
}

func (s *service) FetchCollectionByPost(boardID, postID string, profileID int) ([]map[string]interface{}, error) {
	return fetchCollectionByPost(s.mongodb, boardID, postID, s.peopleRpc, s.storageService, profileID)
}

func (s *service) DeleteNotesOnPost(postID string) error {
	return deleteNotesOnPost(s.mongodb, postID)
}
