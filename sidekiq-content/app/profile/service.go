package profile

import (
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/config"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/email"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/app/storage"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/cache"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/database"
	repo "github.com/ProImaging/sidekiq-backend/sidekiq-content/model"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/mongodatabase"
	"github.com/ProImaging/sidekiq-backend/sidekiq-content/util"
	"github.com/ProImaging/sidekiq-backend/sidekiq-models/model"
)

// Service - defines Profile service
type Service interface {
	ListAllOpenProfiles() (map[string]interface{}, error)
	ValidateProfile(profileID int, userID int) error
	GetProfilesByUserID(userID int, storageService storage.Service) (map[string]interface{}, error)
	GetProfileCountByUserID(userID int) (map[string]interface{}, error)
	SetProfile(userID int) (*model.Account, error)
	AddProfile(profile model.Profile, userID int) (map[string]interface{}, error)
	DeleteProfile(userID string, profileID []string) (map[string]interface{}, error)
	EditProfile(profile model.Profile) (map[string]interface{}, error)
	UpdateProfileSettings(profile model.UpdateProfileSettings, userID int) (map[string]interface{}, error)
	GetProfileInfo(profileID int) (map[string]interface{}, error)
	UpdateProfileTags(profile model.Profile, tags []string) error
	UpdateProfileTagsNew(profileID string) error
	FetchTags(profileID int) (map[string]interface{}, error)
	GetPeopleInfo(storageService storage.Service, profileID, requestType int, limit, page string, searchParameter ...string) (map[string]interface{}, error)
	DeleteConnection(payload map[string][]string, profileID int) (map[string]interface{}, error)
	GenerateCode(storageService storage.Service, userID, CallerprofileID int, payload model.ConnectionRequest) (map[string]interface{}, error)
	DeleteCode(storageService storage.Service, userID int, codes map[string]interface{}) (map[string]interface{}, error)
	SendConnectionRequest(storageService storage.Service, profileID int, connReq map[string]interface{}) (map[string]interface{}, error)
	AcceptConnectionRequest(profileID int, code string) (map[string]interface{}, error)
	MoveConnection(payload map[string]interface{}, profileID int) (map[string]interface{}, error)
	FetchProfilesWithCoManager(storageService storage.Service, profileID int, profiles []model.Profile, search, page, limit string) (map[string]interface{}, error)
	GetProfilesWithInfoByUserID(storageService storage.Service, userID int) (map[string]interface{}, error)
	FetchManagingProfiles(profileID int) (map[string]interface{}, error)
	FetchExternalProfiles(storageService storage.Service, userID int, search, page, limit string) (map[string]interface{}, error)
	AcceptCoManagerRequest(storageService storage.Service, userID, profileID int, code string) (map[string]interface{}, error)
	SendCoManagerRequest(profileID int, connReq map[string]interface{}) (map[string]interface{}, error)
	LeaveProfile(profileToLeave int) (map[string]interface{}, error)
	UpdateShareableSettings(profileID int, shareableSettings model.ShareableSettings) (map[string]interface{}, error)
	GetOrgStaff(storageService storage.Service, profileID int, userID, limit, page, searchParameter string) (map[string]interface{}, error)
	GetOrgInfo(userID int) (map[string]interface{}, error)
	FetchMembershipDetails(userID string) (map[string]interface{}, error)
	FetchStaffProfile(userID, comanagerID int) (map[string]interface{}, error)
	UpdateStaffProfile(userID, comanagerID int, payload model.OrgStaff) (map[string]interface{}, error)
	FetchBoards(profileID int, sectionType string) (map[string]interface{}, error)
	FetchProfileBoardsView(storageService storage.Service, profileID string, myID string) (map[string]interface{}, error)
	FetchProfileView(storageService storage.Service, profileID, myID string) (map[string]interface{}, error)
	// FetchConciseProfile(id int) (*model.ConciseProfile, error)
	UpdateOrgInfo(payload model.Organization) (map[string]interface{}, error)
	AddConnectionDetails(payload model.Connection) (map[string]interface{}, error)
	GetConnectionDetails(callerProfileID, connectionProfileID string) (map[string]interface{}, error)
	FetchDefaultBoardID(profileID int) (string, error)
	GetScreenName(profileID, connProfileID int) (string, error)
	GetOwnerInfoUsingProfileIDs(profileIDs []string) ([]model.ConciseProfile, error)
	UpdateDefaultThingsBoard(profileID int, defBoardID string) error
}

type service struct {
	config         *config.Config
	dbMaster       *database.Database
	dbReplica      *database.Database
	mongodb        *mongodatabase.DBConfig
	cache          *cache.Cache
	storageService storage.Service
	emailService   email.Service
}

// NewService - creates new Profile service
func NewService(repos *repo.Repos, conf *config.Config) Service {
	return &service{
		config:         conf,
		mongodb:        repos.MongoDB,
		dbMaster:       repos.MasterDB,
		dbReplica:      repos.ReplicaDB,
		cache:          repos.Cache,
		storageService: storage.NewService(repos, conf),
		emailService:   email.NewService(),
	}
}

func (s *service) ListAllOpenProfiles() (map[string]interface{}, error) {
	return ListAllOpenProfiles(s.dbMaster)
}

func (s *service) ValidateProfile(profileID int, userID int) error {
	return ValidateProfileByUser(s.dbMaster, profileID, userID)
}

func (s *service) GetProfilesByUserID(userID int, storageService storage.Service) (map[string]interface{}, error) {
	return getProfilesByUserID(s.dbMaster, userID, storageService)
}

func (s *service) GetProfileCountByUserID(userID int) (map[string]interface{}, error) {
	return getProfileCountByUserID(s.dbMaster, userID)
}

func (s *service) SetProfile(userID int) (*model.Account, error) {
	return setProfile(s.dbMaster, userID)
}

func (s *service) AddProfile(profile model.Profile, userID int) (map[string]interface{}, error) {
	return addProfile(s.dbMaster, profile, userID)
}

func (s *service) DeleteProfile(userID string, profileIDs []string) (map[string]interface{}, error) {
	return deleteProfile(s.dbMaster, userID, profileIDs)
}

func (s *service) EditProfile(profile model.Profile) (map[string]interface{}, error) {
	return editProfile(s.dbMaster, profile)
}

func (s *service) UpdateProfileSettings(profile model.UpdateProfileSettings, userID int) (map[string]interface{}, error) {
	return updateProfileSettings(s.dbMaster, profile, userID)
}

func (s *service) GetProfileInfo(profileID int) (map[string]interface{}, error) {
	return getProfileInfo(s.dbMaster, profileID, s.storageService)
}

func (s *service) UpdateProfileTags(profile model.Profile, tags []string) error {
	return updateProfileTags(s.dbMaster, profile, tags)
}

func (s *service) FetchTags(profileID int) (map[string]interface{}, error) {
	return fetchTags(s.dbMaster, profileID)
}

func (s *service) GetPeopleInfo(storageService storage.Service, profileID, requestType int, limit, page string, searchParameter ...string) (map[string]interface{}, error) {
	switch requestType {
	case 1:
		return fetchProfileConnections(s.mongodb, s.dbMaster, storageService, profileID, limit, page, searchParameter[0])
	case 2:
		return fetchBoardFollowers(s.dbMaster, storageService, profileID, limit, page, searchParameter[0], searchParameter[1])
	case 3:
		return fetchFollowingBoards(s.dbMaster, storageService, profileID, limit, page, searchParameter[0], searchParameter[1])
	case 4:
		return fetchConnectionRequests(s.mongodb, s.dbMaster, storageService, profileID, limit, page)
	case 5:
		return fetchArchivedConnections(s.mongodb, s.dbMaster, storageService, profileID, limit, page, searchParameter[0])
	case 6:
		return fetchBlockedConnections(s.mongodb, s.dbMaster, storageService, profileID, limit, page, searchParameter[0])
	}
	return util.SetResponse(nil, 0, "Request type invalid"), nil
}

func (s *service) DeleteConnection(payload map[string][]string, profileID int) (map[string]interface{}, error) {
	return deleteConnection(s.mongodb, payload, profileID)
}

func (s *service) SendConnectionRequest(storageService storage.Service, profileID int, connReq map[string]interface{}) (map[string]interface{}, error) {
	return sendConnectionRequest(s.dbMaster, s.mongodb, s.emailService, storageService, profileID, connReq)
}

func (s *service) AcceptConnectionRequest(profileID int, code string) (map[string]interface{}, error) {
	return acceptConnectionRequest(s.dbMaster, s.mongodb, profileID, code)
}

func (s *service) MoveConnection(payload map[string]interface{}, profileID int) (map[string]interface{}, error) {
	return moveConnection(s.mongodb, payload, profileID)
}

func (s *service) AcceptCoManagerRequest(storageService storage.Service, userID, profileID int, code string) (map[string]interface{}, error) {
	return acceptCoManagerRequest(s.dbMaster, s.mongodb, storageService, userID, profileID, code)
}

func (s *service) GenerateCode(storageService storage.Service, userID, CallerprofileID int, payload model.ConnectionRequest) (map[string]interface{}, error) {
	return generateCode(s.mongodb, s.dbMaster, storageService, userID, CallerprofileID, payload)
}

func (s *service) DeleteCode(storageService storage.Service, userID int, codes map[string]interface{}) (map[string]interface{}, error) {
	return deleteCode(s.mongodb, s.dbMaster, storageService, userID, codes)
}

func (s *service) SendCoManagerRequest(profileID int, connReq map[string]interface{}) (map[string]interface{}, error) {
	return sendCoManagerRequest(s.dbMaster, s.mongodb, s.emailService, profileID, connReq)
}

func (s *service) FetchProfilesWithCoManager(storageService storage.Service, profileID int, profiles []model.Profile, search, page, limit string) (map[string]interface{}, error) {
	return fetchProfilesWithCoManager(storageService, s.dbMaster, s.mongodb, profileID, profiles, search, page, limit)
}

func (s *service) GetProfilesWithInfoByUserID(storageService storage.Service, userID int) (map[string]interface{}, error) {
	return getProfilesWithInfoByUserID(storageService, s.dbMaster, userID)
}

func (s *service) FetchManagingProfiles(profileID int) (map[string]interface{}, error) {
	return fetchManagingProfiles(s.dbMaster, profileID)
}

func (s *service) FetchExternalProfiles(storageService storage.Service, userID int, search, page, limit string) (map[string]interface{}, error) {
	return fetchExternalProfiles(storageService, s.dbMaster, userID, search, page, limit)
}

func (s *service) LeaveProfile(profileToLeave int) (map[string]interface{}, error) {
	return leaveProfile(s.dbMaster, profileToLeave)
}

func (s *service) UpdateShareableSettings(profileID int, shareableSettings model.ShareableSettings) (map[string]interface{}, error) {
	return updateShareableSettings(s.dbMaster, s.mongodb, profileID, shareableSettings)
}

func (s *service) GetOrgStaff(storageService storage.Service, profileID int, userID, limit, page, searchParameter string) (map[string]interface{}, error) {
	return getOrgStaff(storageService, s.dbMaster, profileID, userID, limit, page, searchParameter)
}

func (s *service) GetOrgInfo(userID int) (map[string]interface{}, error) {
	return getOrgInfo(s.dbMaster, userID, s.storageService)
}

func (s *service) UpdateOrgInfo(payload model.Organization) (map[string]interface{}, error) {
	return updateOrgInfo(s.dbMaster, payload)
}

func (s *service) FetchMembershipDetails(userID string) (map[string]interface{}, error) {
	return fetchMembershipDetails(s.dbMaster, userID)
}

func (s *service) FetchStaffProfile(userID, profileID int) (map[string]interface{}, error) {
	return fetchStaffProfile(s.dbMaster, userID, profileID, s.storageService)
}

func (s *service) FetchBoards(profileID int, sectionType string) (map[string]interface{}, error) {
	return fetchBoards(s.dbMaster, profileID, sectionType)
}

func (s *service) FetchProfileBoardsView(storageService storage.Service, profileID string, myID string) (map[string]interface{}, error) {
	return fetchProfileBoardsView(storageService, s.dbMaster, s.mongodb, profileID, myID)
}

func (s *service) FetchProfileView(storageService storage.Service, profileID, myID string) (map[string]interface{}, error) {
	return fetchProfileView(storageService, s.dbMaster, s.mongodb, profileID, myID)
}

// func (s *service) FetchConciseProfile(id int) (*model.ConciseProfile, error) {
// 	return getConciseProfile(s.dbMaster, id, s.storageService)
// }

func (s *service) UpdateStaffProfile(userID, comanagerID int, payload model.OrgStaff) (map[string]interface{}, error) {
	return updateStaffProfile(s.dbMaster, userID, comanagerID, payload)
}

func (s *service) AddConnectionDetails(payload model.Connection) (map[string]interface{}, error) {
	return addConnectionDetails(s.dbMaster, s.mongodb, payload, s.storageService)
}

func (s *service) GetConnectionDetails(callerProfileID, connectionProfileID string) (map[string]interface{}, error) {
	return getConnectionDetails(s.mongodb, callerProfileID, connectionProfileID)
}

func (s *service) FetchDefaultBoardID(profileID int) (string, error) {
	return fetchDefaultBoardID(s.dbMaster, profileID)
}

func (s *service) UpdateProfileTagsNew(profileID string) error {
	return updateProfileTagsNew(s.mongodb, s.dbMaster, profileID)
}

func (s *service) GetScreenName(profileID, connProfileID int) (string, error) {
	return getScreenName(s.mongodb, s.dbMaster, profileID, connProfileID)
}

func (s *service) GetOwnerInfoUsingProfileIDs(profileIDs []string) ([]model.ConciseProfile, error) {
	return getOwnerInfoUsingProfileIDs(s.dbMaster, profileIDs)
}

func (s *service) UpdateDefaultThingsBoard(profileID int, defBoardID string) error {
	return updateDefaultThingsBoard(s.dbMaster, profileID, defBoardID)
}
