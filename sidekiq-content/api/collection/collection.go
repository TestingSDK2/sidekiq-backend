package collection

import (
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/api/common"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/board"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/collection"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/file"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/note"
	repo "github.com/ProImaging/sidekiq-backend/sidekiq-content/model"
	"github.com/ProImaging/sidekiq-backend/sidekiq-models/model"

	// "github.com/ProImaging/sidekiq-backend/sidekiq-content/app/notification"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/post"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/profile"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/recent"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/storage"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/task"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/thing"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/thingactivity"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/cache"
)

type api struct {
	config *common.Config
	// clientMgr            *notification.ClientManager
	fileService          file.Service
	storageService       storage.Service
	taskService          task.Service
	noteService          note.Service
	profileService       profile.Service
	collectionService    collection.Service
	recentThingsService  recent.Service
	thingActivityService thingactivity.Service
	fileStore            model.FileStorage
	tmpFileStore         model.FileStorage
	cache                *cache.Cache
	boardService         board.Service
	postService          post.Service
	thingService         thing.Service
	repos                *repo.Repos
}

// New creates a new collection api
func New(conf *common.Config, fileService file.Service, storageService storage.Service,
	profileService profile.Service, collectionService collection.Service, taskService task.Service,
	noteService note.Service, recentThingsService recent.Service, thingActivityService thingactivity.Service, repos *repo.Repos, boardService board.Service,
	postService post.Service, thingService thing.Service,
) *api {
	return &api{
		config: conf,
		// clientMgr:            clientMgr,
		postService:          postService,
		fileService:          fileService,
		storageService:       storageService,
		taskService:          taskService,
		noteService:          noteService,
		profileService:       profileService,
		collectionService:    collectionService,
		recentThingsService:  recentThingsService,
		thingActivityService: thingActivityService,
		fileStore:            repos.Storage,
		tmpFileStore:         repos.TmpStorage,
		cache:                repos.Cache,
		boardService:         boardService,
		thingService:         thingService,
		repos:                repos,
	}
}
