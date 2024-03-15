package board

import (
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/config"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/profile"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/storage"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/cache"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/database"
	repo "github.com/TestingSDK2/sidekiq-backend/sidekiq-content/model"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/mongodatabase"
	model "github.com/TestingSDK2/sidekiq-backend/sidekiq-models"

	// peoplerpc "github.com/sidekiq-people/proto/people"
	peoplerpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-people/v1"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Service - defines Board service
type Service interface {
	FetchBoards(profileID int, fetchSubBoards bool, page, limit string) (map[string]interface{}, error)
	FetchFollowedBoards(search string, profileID int, limit, page int, sortBy string, orderBy string) (map[string]interface{}, error)
	FetchBoardsAndPostByState(profileID int, state string, limit, page int, sortBy, orderBy string, fetchPost bool, searchKeyword string) (map[string]interface{}, error)
	SearchBoards(profileID int, boardName string, fetchSubBoards bool, page, limit string) (map[string]interface{}, error)
	FetchSubBoards(parentID string, profileID, limit int) (map[string]interface{}, error)
	FetchSubBoardsOfProfile(profileID int, page, limit string) (map[string]interface{}, error)
	FetchSubBoardsByProfile(parentID string, profileID, limit int, publicOnly bool) (map[string]interface{}, error)
	FetchSubBoardsOfBoard(profileID int, boardID, page, limit string) (map[string]interface{}, error)
	AddBoard(payload model.Board, profileID int) (map[string]interface{}, error)
	FetchBoardByID(boardID string, role ...string) (map[string]interface{}, error)
	FetchBoardDetailsByID(boardID string) (map[string]interface{}, error)
	AddViewerInBoardByID(boardID, profileID string) (map[string]interface{}, error)
	// FetchOpenedBoardInfo(boardID string, s.profileService profile.Service, s.storageService storage.Service role ...string) (map[string]interface{}, error)
	UpdateBoard(payload map[string]interface{}, boardID string, profileID int) (map[string]interface{}, error)
	DeleteBoard(Id string, profileID int) (map[string]interface{}, error)
	FindBoardMappings(boards []*model.Board) ([]map[string]interface{}, error)
	GetBoardPermissionByProfile(boards []*model.Board, profileID int) (*model.BoardPermission, error)
	AddBoardMapping(boardMapping *model.BoardMapping) error
	GetParentBoards(boardID primitive.ObjectID) ([]map[string]interface{}, error)
	BoardUnfollow(boardID string, profileID int) (map[string]interface{}, error)
	BoardFollow(payload model.BoardFollowInfo) (map[string]interface{}, error)
	GetBoardMembers(boardID, limit, page, search, role string) (map[string]interface{}, error)
	GetBoardMembers2(boardID, limit, page, search, role string) (map[string]interface{}, error)
	FetchConnectionsMembers(profileID int, boardID string) (map[string]interface{}, error)
	InviteMembers(boardID string, profileID int, invites []model.BoardMemberRoleRequest) (map[string]interface{}, error)
	HandleBoardInvitation(profileID int, boardInvitation model.HandleBoardInvitation) (map[string]interface{}, error)
	ListBoardInvites(profileID int) (map[string]interface{}, error)
	ChangeProfileRole(profileID int, boardID string, cbp model.ChangeProfileRole) (map[string]interface{}, error)
	BlockMembers(profileID int, boardID string, blockMembers []model.BoardMemberRoleRequest) (map[string]interface{}, error)
	UnblockMembers(profileID int, boardID string, bmr []model.BoardMemberRoleRequest) (map[string]interface{}, error)
	ListBlockedMembers(profileID int, page, limit, boardID, search string) (map[string]interface{}, error)
	RemoveMembers(profileID int, boardID string, blockMembers []model.BoardMemberRoleRequest) (map[string]interface{}, error)
	BoardSettings(boardID string, profileID int, payload map[string]interface{}) (map[string]interface{}, error)
	BoardAuth(boardID primitive.ObjectID, password string) (map[string]interface{}, error)
	GetBoardFollowers(boardID, query, page, limit string) (map[string]interface{}, error)
	GetThingLocationOnBoard(boardID string) (string, error)
	GetSharedBoards(profileID int, search, page, limit, sortBy, orderBy string) (map[string]interface{}, error)
	GetBoardThingOwners(boardID, profileID string, userID int) (map[string]interface{}, error)
	FetchBoardThingExt(boardID, profileID string, userID int) (map[string]interface{}, error)
	FetchBoardInfo(boardID string, fields ...string) (map[string]interface{}, error)
	UpdateBoardThingsTags(profileID int, boardID, thingID string, tags []string) error
	GetBoardThingsTags(boardID string) ([]string, error)
	DeleteFromBoardThingsTags(boardID, thingID string) (map[string]interface{}, error)
	GetBoardProfileRole(boardID string, profileID string) (string, error)
	GetProfileTags(profileID int) ([]string, error)
}

type service struct {
	config         *config.Config
	dbMaster       *database.Database
	dbReplica      *database.Database
	mongodb        *mongodatabase.DBConfig
	profileService profile.Service
	storageService storage.Service
	cache          *cache.Cache
	peopleRpc      peoplerpc.AccountServiceClient
}

// NewService - creates new Board service
func NewService(repos *repo.Repos, conf *config.Config) Service {
	return &service{
		config:         conf,
		mongodb:        repos.MongoDB,
		dbMaster:       repos.MasterDB,
		dbReplica:      repos.ReplicaDB,
		profileService: profile.NewService(repos, conf),
		storageService: storage.NewService(repos, conf),
		cache:          repos.Cache,
		peopleRpc:      repos.PeopleGrpcServiceClient,
	}
}

func (s *service) FetchSubBoards(parentID string, profileID, limit int) (map[string]interface{}, error) {
	return fetchSubBoards(s.cache, s.peopleRpc, s.storageService, s.mongodb, s.dbMaster, parentID, profileID, limit)
}

func (s *service) FetchBoards(profileID int, fetchSubBoards bool, page, limit string) (map[string]interface{}, error) {
	return getBoardsByProfile(s.mongodb, s.dbMaster, s.cache, profileID, s.peopleRpc, s.storageService, fetchSubBoards, page, limit)
}

func (s *service) FetchFollowedBoards(search string, profileID int, limit, page int, sortBy string, orderBy string) (map[string]interface{}, error) {
	return getFollowedBoardsByProfile(s.mongodb, s.dbMaster, s.cache, search, profileID, s.peopleRpc, s.storageService, limit, page, sortBy, orderBy)
}

func (s *service) FetchBoardsAndPostByState(profileID int, state string, limit, page int, sortBy, orderBy string, fetchPost bool, searchKeyword string) (map[string]interface{}, error) {
	return getBoardsAndPostByState(s.mongodb, s.dbMaster, s.cache, profileID, s.peopleRpc, s.storageService, state, limit, page, sortBy, orderBy, fetchPost, searchKeyword)
}

func (s *service) SearchBoards(profileID int, boardName string, fetchSubBoards bool, page, limit string) (map[string]interface{}, error) {
	return searchBoardsByProfile(s.mongodb, s.dbMaster, s.cache, profileID, s.profileService, s.storageService, boardName, fetchSubBoards, page, limit)
}

func (s *service) AddBoard(payload model.Board, profileID int) (map[string]interface{}, error) {
	return addBoard(s.cache, s.mongodb, s.peopleRpc, s.storageService, payload, profileID)
}

func (s *service) FetchBoardByID(boardID string, role ...string) (map[string]interface{}, error) {
	return getBoardByID(s.dbMaster, s.mongodb, boardID, s.peopleRpc, s.storageService, role[0])
}

func (s *service) FetchBoardDetailsByID(boardID string) (map[string]interface{}, error) {
	return getBoardDetailsByID(s.mongodb, boardID)
}

func (s *service) AddViewerInBoardByID(boardID, profileID string) (map[string]interface{}, error) {
	return addViewerInBoard(s.mongodb, boardID, profileID)
}

// func (s *service) FetchOpenedBoardInfo(boardID string, s.profileService profile.Service, s.storageService storage.Service, role ...string) (map[string]interface{}, error) {
// 	return fetchOpenedBoardInfo(s.dbMaster, s.mongodb, boardID, s.profileService, s.storageService, role[0])
// }

func (s *service) UpdateBoard(payload map[string]interface{}, boardID string, profileID int) (map[string]interface{}, error) {
	return updateBoard(s.cache, s.mongodb, payload, boardID, profileID)
}

func (s *service) DeleteBoard(Id string, profileID int) (map[string]interface{}, error) {
	return deleteBoard(s.cache, s.mongodb, Id, profileID)
}

func (s *service) FindBoardMappings(boards []*model.Board) ([]map[string]interface{}, error) {
	return findBoardMappings(s.mongodb, boards)
}

func (s *service) GetBoardPermissionByProfile(boards []*model.Board, profileID int) (*model.BoardPermission, error) {
	return getBoardPermissionsByProfile(s.mongodb, boards, profileID)
}

func (s *service) AddBoardMapping(boardMapping *model.BoardMapping) error {
	return addBoardMapping(s.mongodb, boardMapping)
}

func (s *service) GetParentBoards(boardID primitive.ObjectID) ([]map[string]interface{}, error) {
	return getParentBoards(s.mongodb, boardID)
}

func (s *service) BoardUnfollow(boardID string, profileID int) (map[string]interface{}, error) {
	return boardUnfollow(s.dbMaster, s.mongodb, s.cache, boardID, profileID)
}

func (s *service) BoardFollow(payload model.BoardFollowInfo) (map[string]interface{}, error) {
	return boardFollow(s.mongodb, s.dbMaster, s.cache, payload)
}

func (s *service) BoardSettings(boardID string, profileID int, payload map[string]interface{}) (map[string]interface{}, error) {
	return boardSettings(s.cache, s.peopleRpc, s.storageService, s.dbMaster, s.mongodb, boardID, profileID, payload)
}

func (s *service) GetBoardMembers(boardID, limit, page, search, role string) (map[string]interface{}, error) {
	return getBoardMembers(s.mongodb, s.dbMaster, s.peopleRpc, s.storageService, boardID, limit, page, search, role)
}

func (s *service) GetBoardMembers2(boardID, limit, page, search, role string) (map[string]interface{}, error) {
	return getBoardMembers2(s.mongodb, s.dbMaster, s.peopleRpc, s.storageService, boardID, limit, page, search, role)
}

func (s *service) FetchConnectionsMembers(profileID int, boardID string) (map[string]interface{}, error) {
	return fetchConnectionsMembers(s.mongodb, s.dbMaster, profileID, boardID)
}

func (s *service) InviteMembers(boardID string, profileID int, invites []model.BoardMemberRoleRequest) (map[string]interface{}, error) {
	return inviteMembers(s.cache, s.profileService, s.storageService, s.mongodb, s.dbMaster, boardID, profileID, invites)
}

func (s *service) ListBoardInvites(profileID int) (map[string]interface{}, error) {
	return listBoardInvites(s.storageService, s.mongodb, s.dbMaster, profileID)
}

func (s *service) HandleBoardInvitation(profileID int, boardInvitation model.HandleBoardInvitation) (map[string]interface{}, error) {
	return handleBoardInvitation(s.cache, s.mongodb, s.dbMaster, profileID, boardInvitation)
}

func (s *service) ChangeProfileRole(profileID int, boardID string, cbp model.ChangeProfileRole) (map[string]interface{}, error) {
	return changeProfileRole(s.cache, s.mongodb, profileID, boardID, cbp)
}

func (s *service) BlockMembers(profileID int, boardID string, membersToBlock []model.BoardMemberRoleRequest) (map[string]interface{}, error) {
	return blockMembers(s.cache, s.mongodb, profileID, boardID, membersToBlock)
}

func (s *service) ListBlockedMembers(profileID int, page, limit, boardID, search string) (map[string]interface{}, error) {
	return getBlockedMembers(s.mongodb, s.dbMaster, s.cache, s.profileService, s.storageService, profileID, page, limit, boardID, search)
}

func (s *service) UnblockMembers(profileID int, boardID string, bmr []model.BoardMemberRoleRequest) (map[string]interface{}, error) {
	return unblockMembers(s.mongodb, s.cache, profileID, boardID, bmr)
}

func (s *service) RemoveMembers(profileID int, boardID string, membersToRemove []model.BoardMemberRoleRequest) (map[string]interface{}, error) {
	return removeMembers(s.cache, s.mongodb, s.peopleRpc, s.storageService, profileID, boardID, membersToRemove)
}

func (s *service) FetchSubBoardsByProfile(parentID string, profileID, limit int, publicOnly bool) (map[string]interface{}, error) {
	return fetchSubBoardsByProfile(s.cache, s.mongodb, s.dbMaster, parentID, profileID, limit, publicOnly)
}

func (s *service) BoardAuth(boardID primitive.ObjectID, password string) (map[string]interface{}, error) {
	return boardAuth(s.mongodb, boardID, password)
}

func (s *service) FetchSubBoardsOfProfile(profileID int, page, limit string) (map[string]interface{}, error) {
	return fetchSubBoardsOfProfile(s.mongodb, s.peopleRpc, s.storageService, profileID, page, limit)
}

func (s *service) FetchSubBoardsOfBoard(profileID int, boardID, page, limit string) (map[string]interface{}, error) {
	return fetchSubBoardsOfBoard(s.mongodb, s.peopleRpc, s.storageService, profileID, boardID, page, limit)
}

func (s *service) GetBoardFollowers(boardID, query, page, limit string) (map[string]interface{}, error) {
	return getBoardFollowers(s.dbMaster, s.mongodb, s.peopleRpc, s.storageService, boardID, query, page, limit)
}

func (s *service) GetThingLocationOnBoard(boardID string) (string, error) {
	return fetchThingLocationOnBoard(s.mongodb, boardID)
}

func (s *service) GetSharedBoards(profileID int, search, page, limit, sortBy, orderBy string) (map[string]interface{}, error) {
	return getSharedBoards(s.mongodb, s.cache, s.peopleRpc, s.storageService, profileID, search, page, limit, sortBy, orderBy)
}

func (s *service) GetBoardThingOwners(boardID, profileID string, userID int) (map[string]interface{}, error) {
	return getBoardThingOwners(s.dbMaster, s.mongodb, s.profileService, boardID, profileID, userID)
}

func (s *service) FetchBoardThingExt(boardID, profileID string, userID int) (map[string]interface{}, error) {
	return getBoardThingExt(s.dbMaster, s.mongodb, s.profileService, boardID, profileID, userID)
}

func (s *service) FetchBoardInfo(boardID string, fields ...string) (map[string]interface{}, error) {
	return fetchBoardInfo(s.mongodb, boardID, fields)
}

func (s *service) UpdateBoardThingsTags(profileID int, boardID, thingID string, tags []string) error {
	return updateBoardThingsTags(s.mongodb, s.dbMaster, profileID, boardID, thingID, tags)
}

func (s *service) GetBoardThingsTags(boardID string) ([]string, error) {
	return getBoardThingsTags(s.mongodb, boardID)
}

func (s *service) DeleteFromBoardThingsTags(boardID, thingID string) (map[string]interface{}, error) {
	return deleteFromBoardThingsTags(s.mongodb, boardID, thingID)
}

func (s *service) GetBoardProfileRole(boardID string, profileID string) (string, error) {
	return getBoardProfileRole(s.mongodb, s.cache, boardID, profileID)
}

func (s *service) GetProfileTags(profileID int) ([]string, error) {
	return getProfileTags(s.mongodb, profileID)
}
