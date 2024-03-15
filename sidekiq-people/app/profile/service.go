package profile

import (
	"github.com/ProImaging/sidekiq-backend/sidekiq-models/model"
	"github.com/ProImaging/sidekiq-backend/sidekiq-people/app/config"
	"github.com/ProImaging/sidekiq-backend/sidekiq-people/app/email"
	"github.com/ProImaging/sidekiq-backend/sidekiq-people/app/storage"
	"github.com/ProImaging/sidekiq-backend/sidekiq-people/cache"
	"github.com/ProImaging/sidekiq-backend/sidekiq-people/database"
	"github.com/ProImaging/sidekiq-backend/sidekiq-people/helper"
	repo "github.com/ProImaging/sidekiq-backend/sidekiq-people/model"
	"github.com/ProImaging/sidekiq-backend/sidekiq-people/mongodatabase"
	"github.com/ProImaging/sidekiq-backend/sidekiq-people/util"
)

// Service - defines Profile service
type Service interface {
	GetConciseProfile(profileID int) (*model.ConciseProfile, error)
	ValidateProfile(profileID int, accountID int) error
	GetProfilesByUserID(userID int) (map[string]interface{}, error)
	GetProfileCountByUserID(userID int) (map[string]interface{}, error)
	AddProfile(profile model.Profile, userID int) (map[string]interface{}, error)
	GetProfileInfo(profileID int) (map[string]interface{}, error)
	FetchTags(profileID int) (map[string]interface{}, error)
	EditProfile(profile model.Profile) (map[string]interface{}, error)
	UpdateProfileTagsNew(profileID string, tags []string) error
	DeleteProfile(userID string, profileID []string) (map[string]interface{}, error)
	GenerateCode(userID, CallerprofileID int, payload model.ConnectionRequest) (map[string]interface{}, error)
	DeleteCode(userID int, codes map[string]interface{}) (map[string]interface{}, error)
	UpdateProfileSettings(profile model.UpdateProfileSettings, userID int) (map[string]interface{}, error)
	UpdateShareableSettings(profileID int, shareableSettings model.ShareableSettings) (map[string]interface{}, error)
	SendCoManagerRequest(profileID int, connReq map[string]interface{}) (map[string]interface{}, error)
	AcceptCoManagerRequest(userID, profileID int, code string) (map[string]interface{}, error)
	GetProfilesWithInfoByUserID(userID int) (map[string]interface{}, error)
	FetchProfilesWithCoManager(profileID int, profiles []model.Profile, search, page, limit string) (map[string]interface{}, error)
	FetchExternalProfiles(userID int, search, page, limit string) (map[string]interface{}, error)
	GetPeopleInfo(profileID, requestType int, limit, page string, searchParameter ...string) (map[string]interface{}, error)
	FetchBoards(profileID int, sectionType string) (map[string]interface{}, error)
	ListAllOpenProfiles() (map[string]interface{}, error)
	MoveConnection(payload map[string]interface{}, profileID int) (map[string]interface{}, error)
	DeleteConnection(payload map[string][]string, profileID int) (map[string]interface{}, error)
	SendConnectionRequest(profileID int, connReq map[string]interface{}) (map[string]interface{}, error)
	AcceptConnectionRequest(profileID int, code string) (map[string]interface{}, error)
	AddConnectionDetails(payload model.Connection) (map[string]interface{}, error)
	GetConnectionDetails(callerProfileID, connectionProfileID string) (map[string]interface{}, error)
	FetchProfileView(profileID, myID string) (map[string]interface{}, error)
	FetchProfileBoardsView(profileID string, myID string) (map[string]interface{}, error)
	GetOrgStaff(profileID int, userID, limit, page, searchParameter string) (map[string]interface{}, error)
	FetchStaffProfile(userID, comanagerID int) (map[string]interface{}, error)
	UpdateStaffProfile(userID, comanagerID int, payload model.OrgStaff) (map[string]interface{}, error)
	SetOrganizationInfo(payload *model.Organization) (map[string]interface{}, error)
	GetOrgInfo(userID int) (map[string]interface{}, error)
	UpdateOrgInfo(payload model.Organization) (map[string]interface{}, error)
	UpdateDefaultThingsBoard(profileID int, defBoardID string) error
}

type service struct {
	config         *config.Config
	dbMaster       *database.Database
	dbReplica      *database.Database
	mongodb        *mongodatabase.DBConfig
	cache          *cache.Cache
	emailService   email.Service
	storageService storage.Service
}

func NewService(repos *repo.Repos, conf *config.Config) Service {
	return &service{
		config:         conf,
		mongodb:        repos.MongoDB,
		dbMaster:       repos.MasterDB,
		dbReplica:      repos.ReplicaDB,
		cache:          repos.Cache,
		emailService:   email.NewService(),
		storageService: storage.NewService(repos, conf),
	}
}

func (s *service) GetConciseProfile(profileID int) (*model.ConciseProfile, error) {
	return helper.GetConciseProfile(s.dbMaster, profileID, s.storageService)
}

func (s *service) ValidateProfile(profileID int, accountID int) error {
	return ValidateProfileByAccountID(s.dbMaster, profileID, accountID)
}

func (s *service) GetProfilesByUserID(userID int) (map[string]interface{}, error) {
	return getProfilesByUserID(s.dbMaster, userID, s.storageService)
}

func (s *service) GetProfileCountByUserID(userID int) (map[string]interface{}, error) {
	return getProfileCountByUserID(s.dbMaster, userID)
}

func (s *service) AddProfile(profile model.Profile, userID int) (map[string]interface{}, error) {
	return addProfile(s.dbMaster, profile, userID)
}

func (s *service) GetProfileInfo(profileID int) (map[string]interface{}, error) {
	return getProfileInfo(s.dbMaster, profileID, s.storageService)
}

func (s *service) FetchTags(profileID int) (map[string]interface{}, error) {
	return fetchTags(s.dbMaster, profileID)
}

func (s *service) EditProfile(profile model.Profile) (map[string]interface{}, error) {
	return editProfile(s.dbMaster, profile)
}

func (s *service) UpdateProfileTagsNew(profileID string, tags []string) error {
	return updateProfileTagsNew(s.mongodb, s.dbMaster, profileID, tags)
}

func (s *service) DeleteProfile(userID string, profileIDs []string) (map[string]interface{}, error) {
	return deleteProfile(s.dbMaster, userID, profileIDs)
}

func (s *service) GenerateCode(userID, CallerprofileID int, payload model.ConnectionRequest) (map[string]interface{}, error) {
	return generateCode(s.mongodb, s.dbMaster, s.storageService, userID, CallerprofileID, payload)
}

func (s *service) DeleteCode(userID int, codes map[string]interface{}) (map[string]interface{}, error) {
	return deleteCode(s.mongodb, s.dbMaster, s.storageService, userID, codes)
}

func (s *service) UpdateProfileSettings(profile model.UpdateProfileSettings, userID int) (map[string]interface{}, error) {
	return updateProfileSettings(s.dbMaster, profile, userID)
}

func (s *service) UpdateShareableSettings(profileID int, shareableSettings model.ShareableSettings) (map[string]interface{}, error) {
	return updateShareableSettings(s.dbMaster, s.mongodb, profileID, shareableSettings)
}

func (s *service) SendCoManagerRequest(profileID int, connReq map[string]interface{}) (map[string]interface{}, error) {
	return sendCoManagerRequest(s.dbMaster, s.mongodb, s.emailService, profileID, connReq)
}

func (s *service) AcceptCoManagerRequest(userID, profileID int, code string) (map[string]interface{}, error) {
	return acceptCoManagerRequest(s.dbMaster, s.mongodb, s.storageService, userID, profileID, code)
}

func (s *service) GetProfilesWithInfoByUserID(userID int) (map[string]interface{}, error) {
	return getProfilesWithInfoByUserID(s.storageService, s.dbMaster, userID)
}

func (s *service) FetchProfilesWithCoManager(profileID int, profiles []model.Profile, search, page, limit string) (map[string]interface{}, error) {
	return fetchProfilesWithCoManager(s.storageService, s.dbMaster, s.mongodb, profileID, profiles, search, page, limit)
}

func (s *service) FetchExternalProfiles(userID int, search, page, limit string) (map[string]interface{}, error) {
	return fetchExternalProfiles(s.storageService, s.dbMaster, userID, search, page, limit)
}

func (s *service) GetPeopleInfo(profileID, requestType int, limit, page string, searchParameter ...string) (map[string]interface{}, error) {
	switch requestType {
	case 1:
		return fetchProfileConnections(s.mongodb, s.dbMaster, s.storageService, profileID, limit, page, searchParameter[0])
	case 2:
		return fetchBoardFollowers(s.dbMaster, s.storageService, profileID, limit, page, searchParameter[0], searchParameter[1])
	case 3:
		return fetchFollowingBoards(s.dbMaster, s.storageService, profileID, limit, page, searchParameter[0], searchParameter[1])
	case 4:
		return fetchConnectionRequests(s.mongodb, s.dbMaster, s.storageService, profileID, limit, page)
	case 5:
		return fetchArchivedConnections(s.mongodb, s.dbMaster, s.storageService, profileID, limit, page, searchParameter[0])
	case 6:
		return fetchBlockedConnections(s.mongodb, s.dbMaster, s.storageService, profileID, limit, page, searchParameter[0])
	}
	return util.SetResponse(nil, 0, "Request type invalid"), nil
}

func (s *service) FetchBoards(profileID int, sectionType string) (map[string]interface{}, error) {
	return fetchBoards(s.dbMaster, profileID, sectionType)
}

func (s *service) ListAllOpenProfiles() (map[string]interface{}, error) {
	return listAllOpenProfiles(s.dbMaster)
}

func (s *service) MoveConnection(payload map[string]interface{}, profileID int) (map[string]interface{}, error) {
	return moveConnection(s.mongodb, payload, profileID)
}

func (s *service) DeleteConnection(payload map[string][]string, profileID int) (map[string]interface{}, error) {
	return deleteConnection(s.mongodb, payload, profileID)
}

func (s *service) SendConnectionRequest(profileID int, connReq map[string]interface{}) (map[string]interface{}, error) {
	return sendConnectionRequest(s.dbMaster, s.mongodb, s.emailService, s.storageService, profileID, connReq)
}

func (s *service) AcceptConnectionRequest(profileID int, code string) (map[string]interface{}, error) {
	return acceptConnectionRequest(s.dbMaster, s.mongodb, profileID, code)
}

func (s *service) AddConnectionDetails(payload model.Connection) (map[string]interface{}, error) {
	return addConnectionDetails(s.dbMaster, s.mongodb, payload, s.storageService)
}

func (s *service) GetConnectionDetails(callerProfileID, connectionProfileID string) (map[string]interface{}, error) {
	return getConnectionDetails(s.dbMaster, s.mongodb, s.storageService, callerProfileID, connectionProfileID)
}

func (s *service) FetchProfileView(profileID, myID string) (map[string]interface{}, error) {
	return fetchProfileView(s.storageService, s.dbMaster, s.mongodb, profileID, myID)
}

func (s *service) FetchProfileBoardsView(profileID string, myID string) (map[string]interface{}, error) {
	return fetchProfileBoardsView(s.storageService, s.dbMaster, s.mongodb, profileID, myID)
}

func (s *service) GetOrgStaff(profileID int, userID, limit, page, searchParameter string) (map[string]interface{}, error) {
	return getOrgStaff(s.storageService, s.dbMaster, profileID, userID, limit, page, searchParameter)
}

func (s *service) FetchStaffProfile(userID, profileID int) (map[string]interface{}, error) {
	return fetchStaffProfile(s.dbMaster, userID, profileID, s.storageService)
}

func (s *service) UpdateStaffProfile(userID, comanagerID int, payload model.OrgStaff) (map[string]interface{}, error) {
	return updateStaffProfile(s.dbMaster, userID, comanagerID, payload)
}

func (s *service) SetOrganizationInfo(payload *model.Organization) (map[string]interface{}, error) {
	return setOrganizationInfo(s.dbMaster, payload)
}

func (s *service) GetOrgInfo(userID int) (map[string]interface{}, error) {
	return getOrgInfo(s.dbMaster, userID, s.storageService)
}

func (s *service) UpdateOrgInfo(payload model.Organization) (map[string]interface{}, error) {
	return updateOrgInfo(s.dbMaster, payload)
}

func (s *service) UpdateDefaultThingsBoard(profileID int, defBoardID string) error {
	return updateDefaultThingsBoard(s.dbMaster, profileID, defBoardID)
}
