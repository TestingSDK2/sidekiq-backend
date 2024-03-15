package user

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/config"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/email"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/app/storage"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/cache"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-content/database"
	repo "github.com/TestingSDK2/sidekiq-backend/sidekiq-content/model"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-models/model"
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-models/model/notification"
)

// Service defines service for operating on Users
type Service interface {
	Createaccount(user model.AccountSignup) (map[string]interface{}, error)
	GetVerificationCode(userID int, emailID string) (map[string]interface{}, error)
	VerifyVerificationCode(userID int, payload map[string]interface{}) (map[string]interface{}, error)
	SetAccountInformation(user model.Account, userID int) (map[string]interface{}, error)
	AuthUser(creds *model.Credentials) (*model.Account, error)
	ValidateJWTToken(token string) (*model.Account, error)
	ValidateSignupJWTToken(token string) (*model.AccountSignup, error)
	CreateJWTToken(user *model.Account, tokenExpiration time.Duration) (*JWTToken, error)
	CreateSignupJWTToken(user *model.AccountSignup, tokenExpiration time.Duration) (*JWTToken, error)
	FetchUser(id int, skipCache bool) (*model.Account, error)
	FetchBasicUser(id int, skipCache bool) (*model.AccountSignup, error)
	FetchCachedUser(id int) (*model.Account, error)
	FetchSignupCachedUser(id int) *model.AccountSignup
	DeleteSignupCachedUser(id int) error
	FetchContacts(userID int) ([]*model.Contact, error)
	FetchPushSubscriptions(userID int) []*notification.PushSubscription
	CreatePushSubscription(*notification.PushSubscription) (int, error)
	RemovePushSubscription(*notification.PushSubscription) error
	ForgotPassword(userEmail string) (map[string]interface{}, error)
	VerifyLink(token string) (map[string]interface{}, error)
	VerifyPin(pin map[string]interface{}) (map[string]interface{}, error)
	ResetPassword(payload *model.ResetPassword) (map[string]interface{}, error)
	SetAccountType(payload *model.SetAccountType) (map[string]interface{}, error)
	FetchAccounts() (map[string]interface{}, error)
	FetchAccountInformation(userID int) (map[string]interface{}, error)
	FetchAccountServices(user model.Account) (map[string]interface{}, error)
	SetOrganizationInfo(payload *model.Organization) (map[string]interface{}, error)
	UpdateAccountInfo(payload model.Account) (map[string]interface{}, error)
}

type service struct {
	config         *config.Config
	dbMaster       *database.Database
	dbReplica      *database.Database
	cache          *cache.Cache
	storageService storage.Service
	emailService   email.Service
}

// NewService create new UserService
func NewService(repos *repo.Repos, conf *config.Config) Service {
	svc := &service{
		config:         conf,
		dbMaster:       repos.MasterDB,
		dbReplica:      repos.ReplicaDB,
		cache:          repos.Cache,
		storageService: storage.NewService(repos, conf),
		emailService:   email.NewService(),
	}
	return svc
}

func (s *service) Createaccount(user model.AccountSignup) (map[string]interface{}, error) {
	return createaccount(s.dbMaster, user)
}

func (s *service) GetVerificationCode(userID int, emailID string) (map[string]interface{}, error) {
	return getVerificationCode(s.dbMaster, s.emailService, userID, emailID)
}

func (s *service) VerifyVerificationCode(userID int, payload map[string]interface{}) (map[string]interface{}, error) {
	return verifyVerificationCode(s.dbMaster, userID, payload)
}

func (s *service) SetAccountInformation(user model.Account, userID int) (map[string]interface{}, error) {
	resp, err := setAccountInformation(s.dbMaster, user, userID)
	if err != nil {
		return nil, err
	}
	if resp["status"] == 0 {
		return resp, nil
	}
	userObj, err := s.FetchUser(resp["data"].(map[string]interface{})["id"].(int), true)
	if err != nil || userObj == nil {
		return nil, err
	}
	return resp, nil
}

func (s *service) ForgotPassword(userEmail string) (map[string]interface{}, error) {
	return forgotPassword(s.dbMaster, s.emailService, userEmail)
}

func (s *service) VerifyLink(token string) (map[string]interface{}, error) {
	return verifyLink(s.dbMaster, s.emailService, token)
}

func (s *service) ResetPassword(payload *model.ResetPassword) (map[string]interface{}, error) {
	return resetPassword(s.dbMaster, s.emailService, payload)
}

func (s *service) SetAccountType(payload *model.SetAccountType) (map[string]interface{}, error) {
	return setAccountType(s.dbMaster, s.storageService, payload)
}

// FetchAccounts
func (s *service) FetchAccounts() (map[string]interface{}, error) {
	return fetchAccounts(s.dbMaster)
}

// AuthUser - fetches user and verifies their password
func (s *service) AuthUser(creds *model.Credentials) (*model.Account, error) {
	// user := fetchUserForAuthByUsername(s.cache, s.dbMaster, creds.Username)
	user, err := fetchUserForAuthByEmail(s.cache, s.dbMaster, creds.Email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}
	if user.Password != creds.Password {
		return nil, errors.New("incorrect password")
	}
	newUserObj, err := s.FetchUser(user.ID, true)
	if err != nil {
		return nil, err
	}
	return newUserObj, nil
}

func (s *service) FetchUser(id int, skipCache bool) (*model.Account, error) {
	key := getCacheKey(id)
	if !skipCache {
		cachedUser, err := s.FetchCachedUser(id)
		if err != nil {
			return nil, err
		}
		if cachedUser == nil {
			return nil, errors.New("cachedUser empty")
		}
		return cachedUser, nil
	}
	user, err := getUserFromDB(s.dbMaster, id)
	if err != nil {
		return nil, err
	}
	user.Accounts = getAccountPermissions(s.dbMaster, id)
	err = s.cache.SetValue(key, user.ToJSON())
	if err != nil {
		return nil, err
	}
	s.cache.ExpireKey(key, cache.Expire18HR)
	return user, nil
}

func (s *service) FetchBasicUser(id int, skipCache bool) (*model.AccountSignup, error) {
	key := getSignupCacheKey(id)
	if !skipCache {
		cachedUser := s.FetchSignupCachedUser(id)
		if cachedUser != nil {
			return cachedUser, nil
		}
	}
	user, err := getBasicUserFromDB(s.dbMaster, id)
	if err != nil {
		return nil, err
	}
	user.Accounts = getAccountPermissions(s.dbMaster, id)
	s.cache.SetValue(key, user.ToJSON())
	s.cache.ExpireKey(key, cache.Expire18HR)
	return user, nil
}

func (s *service) FetchCachedUser(id int) (*model.Account, error) {
	key := getCacheKey(id)
	val, err := s.cache.GetValue(key)
	if err != nil {
		return nil, err
	}
	var user *model.Account
	json.Unmarshal([]byte(val), &user)
	return user, nil
}

func (s *service) FetchAccountServices(user model.Account) (map[string]interface{}, error) {
	return fetchAccountServices(s.dbMaster, user)
}

func (s *service) FetchSignupCachedUser(id int) *model.AccountSignup {
	key := getSignupCacheKey(id)
	val, err := s.cache.GetValue(key)
	if err != nil {
		return nil
	}
	var user *model.AccountSignup
	json.Unmarshal([]byte(val), &user)
	return user
}

func (s *service) DeleteSignupCachedUser(id int) error {
	key := getSignupCacheKey(id)
	err := s.cache.DeleteValue(key)
	return err
}

func (s *service) FetchContacts(userID int) ([]*model.Contact, error) {
	return getContacts(s.dbMaster, userID)
}

func (s *service) FetchPushSubscriptions(userID int) []*notification.PushSubscription {
	return fetchPushSubscriptions(s.cache, s.dbMaster, userID)
}

func (s *service) CreatePushSubscription(sub *notification.PushSubscription) (int, error) {
	return insertPushSubscriptions(s.dbMaster, sub)
}

func (s *service) RemovePushSubscription(sub *notification.PushSubscription) error {
	return removePushSubscription(s.dbMaster, sub)
}

func (s *service) FetchAccountInformation(userID int) (map[string]interface{}, error) {
	return fetchAccountInformation(s.dbMaster, userID)
}

func (s *service) SetOrganizationInfo(payload *model.Organization) (map[string]interface{}, error) {
	return setOrganizationInfo(s.dbMaster, payload)
}

func (s *service) UpdateAccountInfo(user model.Account) (map[string]interface{}, error) {
	return updateAccountInfo(s.dbMaster, user)
}

func (s *service) VerifyPin(pin map[string]interface{}) (map[string]interface{}, error) {
	return verifyPin(s.dbMaster, pin)
}
