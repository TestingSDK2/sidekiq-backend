package account

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ProImaging/sidekiq-backend/sidekiq-people/app/config"
	"github.com/ProImaging/sidekiq-backend/sidekiq-people/app/email"

	"github.com/ProImaging/sidekiq-backend/sidekiq-people/cache"

	"github.com/ProImaging/sidekiq-backend/sidekiq-people/database"

	"github.com/ProImaging/sidekiq-backend/sidekiq-models/model"
	"github.com/ProImaging/sidekiq-backend/sidekiq-people/app/storage"
	repo "github.com/ProImaging/sidekiq-backend/sidekiq-people/model"
)

// Service defines service for operating on Accounts
type Service interface {
	AuthAccount(creds *model.Credentials) (*model.Account, error)
	FetchAccounts() (map[string]interface{}, error)
	FetchAccount(id int, skipCache bool) (*model.Account, error)
	FetchContacts(accountID int) ([]*model.Contact, error)
	CreateAccount(account model.AccountSignup) (map[string]interface{}, error)
	FetchCachedAccount(id int) (*model.Account, error)
	GetVerificationCode(accountID int, emailID string) (map[string]interface{}, error)
	VerifyVerificationCode(userID int, payload map[string]interface{}) (map[string]interface{}, error)
	VerifyLink(token string) (map[string]interface{}, error)
	ForgotPassword(userEmail string) (map[string]interface{}, error)
	ResetPassword(payload *model.ResetPassword) (map[string]interface{}, error)
	SetAccountType(payload *model.SetAccountType) (map[string]interface{}, error)
	FetchAccountInformation(accountID int) (map[string]interface{}, error)
	FetchAccountServices(account model.Account) (map[string]interface{}, error)
	VerifyPin(pin map[string]interface{}) (map[string]interface{}, error)
	SetAccountInformation(user model.Account, userID int) (map[string]interface{}, error)
	DeleteSignupCachedUser(id int) error
	UpdateAccountInfo(payload model.Account) (map[string]interface{}, error)
}

type service struct {
	config         *config.Config
	dbMaster       *database.Database
	dbReplica      *database.Database
	cache          *cache.Cache
	emailService   email.Service
	storageService storage.Service
}

// NewService create new AccountService
func NewService(repos *repo.Repos, conf *config.Config) Service {
	svc := &service{
		config:         conf,
		dbMaster:       repos.MasterDB,
		dbReplica:      repos.ReplicaDB,
		cache:          repos.Cache,
		emailService:   email.NewService(),
		storageService: storage.NewService(repos, conf),
	}
	return svc
}

// AuthAccount - fetches account and verifies their password
func (s *service) AuthAccount(creds *model.Credentials) (*model.Account, error) {
	accountData, err := fetchAccountForAuthByEmail(s.cache, s.dbMaster, creds.Email)
	if err != nil {
		return nil, err
	}
	if accountData == nil {
		return nil, nil
	}
	if accountData.Password != creds.Password {
		return nil, errors.New("incorrect password")
	}
	newAccountObj, err := s.FetchAccount(accountData.ID, true)
	if err != nil {
		return nil, err
	}
	return newAccountObj, nil
}

func (s *service) FetchAccount(id int, skipCache bool) (*model.Account, error) {
	key := getCacheKey(id)
	if !skipCache {
		cachedAccount, err := s.FetchCachedAccount(id)
		if err != nil {
			return nil, err
		}
		if cachedAccount == nil {
			return nil, errors.New("cachedAccount empty")
		}
		return cachedAccount, nil
	}
	accountData, err := getAccountFromDB(s.dbMaster, id)
	if err != nil {
		return nil, err
	}
	accountData.Accounts = getAccountPermissions(s.dbMaster, id)
	err = s.cache.SetValue(key, accountData.ToJSON())
	if err != nil {
		return nil, err
	}
	s.cache.ExpireKey(key, cache.Expire18HR)
	return accountData, nil
}

func (s *service) FetchCachedAccount(id int) (*model.Account, error) {
	key := getCacheKey(id)
	val, err := s.cache.GetValue(key)
	if err != nil {
		return nil, err
	}
	var accountData *model.Account
	json.Unmarshal([]byte(val), &accountData)
	return accountData, nil
}

func (s *service) FetchAccounts() (map[string]interface{}, error) {
	return fetchAccounts(s.dbMaster)
}

func (s *service) FetchContacts(userID int) ([]*model.Contact, error) {
	return getContacts(s.dbMaster, userID)
}

func (s *service) CreateAccount(account model.AccountSignup) (map[string]interface{}, error) {
	return createAccount(s.dbMaster, account)
}

func (s *service) GetVerificationCode(accountID int, emailID string) (map[string]interface{}, error) {
	return getVerificationCode(s.dbMaster, s.emailService, accountID, emailID)
}

func (s *service) VerifyVerificationCode(userID int, payload map[string]interface{}) (map[string]interface{}, error) {
	return verifyVerificationCode(s.dbMaster, userID, payload)
}

func (s *service) VerifyLink(token string) (map[string]interface{}, error) {
	return verifyLink(s.dbMaster, s.emailService, token)
}

func (s *service) ForgotPassword(userEmail string) (map[string]interface{}, error) {
	return forgotPassword(s.dbMaster, s.emailService, userEmail)
}

func (s *service) ResetPassword(payload *model.ResetPassword) (map[string]interface{}, error) {
	return resetPassword(s.dbMaster, s.emailService, payload)
}

func (s *service) SetAccountType(payload *model.SetAccountType) (map[string]interface{}, error) {
	return setAccountType(s.dbMaster, s.storageService, payload)
}

func (s *service) FetchAccountInformation(userID int) (map[string]interface{}, error) {
	return fetchAccountInformation(s.dbMaster, userID)
}

func (s *service) FetchAccountServices(user model.Account) (map[string]interface{}, error) {
	return fetchAccountServices(s.dbMaster, user)
}

func (s *service) VerifyPin(pin map[string]interface{}) (map[string]interface{}, error) {
	return verifyPin(s.dbMaster, pin)
}

func (s *service) SetAccountInformation(user model.Account, userID int) (map[string]interface{}, error) {
	resp, err := setAccountInformation(s.dbMaster, user, userID)
	if err != nil {
		return nil, err
	}
	if resp["status"] == 0 {
		return resp, nil
	}
	userObj, err := s.FetchAccount(resp["data"].(map[string]interface{})["id"].(int), true)
	if err != nil || userObj == nil {
		return nil, err
	}
	return resp, nil
}

func getSignupCacheKey(userID int) string {
	return fmt.Sprintf("signupuser:%d", userID)
}

func (s *service) DeleteSignupCachedUser(id int) error {
	key := getSignupCacheKey(id)
	err := s.cache.DeleteValue(key)
	return err
}

func (s *service) UpdateAccountInfo(user model.Account) (map[string]interface{}, error) {
	return updateAccountInfo(s.dbMaster, user)
}
