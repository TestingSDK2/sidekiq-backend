package board

import (
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/api/common"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/board"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/collection"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/file"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/message"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/note"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/post"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/profile"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/recent"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/storage"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/task"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/thing"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/thingactivity"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/cache"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/model"
)

type api struct {
	config               *common.Config
	boardService         board.Service
	recentThingsService  recent.Service
	taskService          task.Service
	noteService          note.Service
	fileService          file.Service
	storageService       storage.Service
	profileService       profile.Service
	collectionService    collection.Service
	thingActivityService thingactivity.Service
	thingService         thing.Service
	messageService       message.Service
	cache                *cache.Cache
	repos                *model.Repos
	postService          post.Service
}

// New creates a new board api
func New(conf *common.Config, boardService board.Service, recentThingsService recent.Service,
	taskService task.Service, noteService note.Service,
	fileService file.Service, profileService profile.Service, collectionService collection.Service, storageService storage.Service,
	thingActivityService thingactivity.Service, repos *model.Repos, thingService thing.Service, messageService message.Service, postService post.Service,
) *api {
	return &api{
		config:               conf,
		boardService:         boardService,
		recentThingsService:  recentThingsService,
		taskService:          taskService,
		noteService:          noteService,
		fileService:          fileService,
		storageService:       storageService,
		thingActivityService: thingActivityService,
		profileService:       profileService,
		collectionService:    collectionService,
		cache:                repos.Cache,
		repos:                repos,
		thingService:         thingService,
		messageService:       messageService,
		postService:          postService,
	}
}
