package model

import (
	"encoding/json"
	"io"
	"time"
)

type AccountService struct {
	ID          int       `json:"id" db:"id"`
	Description string    `json:"description" db:"description"`
	Service     string    `json:"service" db:"service"`
	Image       string    `json:"image" db:"image"`
	Fee         float32   `json:"fee" db:"fee"`
	Profiles    int       `json:"profiles" db:"profiles"`
	Recurring   int       `json:"recurring" db:"recurring"`
	ExpiryDate  time.Time `json:"expiry"`
}

// User - model for user data
type Account struct {
	ID               int                   `json:"id" db:"id"`
	AccountType      int                   `json:"accountType" db:"accountType"`
	UserName         string                `json:"userName" db:"userName"`
	FirstName        string                `json:"firstName" db:"firstName"`
	LastName         string                `json:"lastName" db:"lastName"`
	Photo            string                `json:"photo" db:"photo"`
	Thumbs           Thumbnails            `json:"thumbs"`
	Email            string                `json:"email" db:"email"`
	RecoveryEmail    string                `json:"recoveryEmail" db:"recoveryEmail"`
	Phone            string                `json:"phone" db:"phone"`
	Password         string                `json:"password,omitempty" db:"password"`
	CreateDate       time.Time             `json:"createDate" db:"createDate"`
	LastModifiedDate time.Time             `json:"lastModifiedDate" db:"lastModifiedDate"`
	Token            string                `json:"token"`
	Accounts         []*AccountPermimssion `json:"accounts"`
	IsActive         bool                  `json:"isActive" db:"isActive"`
	ResetToken       string                `json:"resetToken" db:"resetToken"`
	ResetStatus      bool                  `json:"resetStatus" db:"resetStatus"`
	ResetTime        []uint8               `json:"resetTime" db:"resetTime"`
}

type RegistrationUser struct {
	ID    int    `json:"id" db:"id"`
	Email string `json:"email" db:"email"`
	Phone string `json:"phone" db:"phone"`
	Type  string `json:"type"`
}

type AccountSignup struct {
	ID               int                   `json:"id" db:"id"`
	Email            string                `json:"email" db:"email" validate:"required"`
	Phone            string                `json:"phone" db:"phone" validate:"required"`
	CreateDate       time.Time             `json:"createDate" db:"createDate"`
	Token            string                `json:"token"`
	Accounts         []*AccountPermimssion `json:"accounts"`
	VerificationCode string                `json:"verificationCode" db:"verificationCode"`
}

// Credentials - user credentials
type Credentials struct {
	UserName string `json:"userName"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AccountPermimssion - permissions to account for user
type AccountPermimssion struct {
	AccountID    int    `json:"accountID" db:"orgID"`
	Company      string `json:"company" db:"company"`
	IsOwner      bool   `json:"isOwner" db:"owner"`
	HasAPIAccess bool   `json:"hasAPIAccess" db:"apiAccess"`
}

// ResetPassword - structure sent by frontend during reset password process
type ResetPassword struct {
	Password   string `json:"password" db:"password"`
	ResetToken string `json:"resetToken" db:"resetToken"`
}

type SetAccountType struct {
	AccountType int    `json:"accountType" db:"accountType"`
	AccountId   string `json:"id" db:"id"`
}

type AccountTypes struct {
	ID          int     `json:"id" db:"id"`
	Service     string  `json:"service" db:"service"`
	Description string  `json:"description" db:"description"`
	Fee         float64 `json:"fee" db:"fee"`
	Profiles    int     `json:"profiles" db:"profiles"`
}

type Organization struct {
	ID                 int        `json:"-" db:"id"`
	AccountID          int        `json:"accountID" db:"accountID"`
	OrganizationName   string     `json:"organizationName" db:"organizationName" validate:"required"`
	Website            string     `json:"website" db:"website" validate:"required"`
	RegistrationNumber string     `json:"registrationNumber" db:"registrationNumber" validate:"required"`
	Email              string     `json:"email" db:"email" validate:"required"`
	Bio                string     `json:"bio" db:"bio"`
	City               string     `json:"city" db:"city"`
	State              string     `json:"state" db:"state"`
	Zip                string     `json:"zip" db:"zip"`
	Country            string     `json:"country" db:"country"`
	Phone              string     `json:"phone" db:"phone"`
	Address            string     `json:"address1" db:"address1"`
	Address2           string     `json:"address2" db:"address2"`
	Photo              string     `json:"photo" db:"photo"`
	Thumbs             Thumbnails `json:"thumbs"`
	Abv                string     `json:"abv" db:"abv"`
	Mission            string     `json:"mission" db:"mission"`
}

type AccountInfoResponse struct {
	ServiceType             string        `json:"serviceType" db:"serviceType"`
	AccountInformation      Account       `json:"accountInfo" db:""`
	OrganizationInformation *Organization `json:"organizationInfo" db:""`
}

type SetAccountTypeResponse struct {
	FirstName string     `json:"firstName" db:"firstName"`
	LastName  string     `json:"lastName" db:"lastName"`
	Photo     string     `json:"photo" db:"photo"`
	Thumbs    Thumbnails `json:"thumbs"`
}

// ToJSON converts user to json string
func (u *Account) ToJSON() string {
	pass := u.Password
	u.Password = ""
	json, _ := json.Marshal(u)
	u.Password = pass
	return string(json)
}

func (u *AccountSignup) ToJSON() string {
	json, _ := json.Marshal(u)
	return string(json)
}

// WriteToJSON encode model directly to writer
func (u *Account) WriteToJSON(w io.Writer) {
	json.NewEncoder(w).Encode(u)
}

// ReadUserFromJSON create User from io.Reader
func ReadUserFromJSON(data io.Reader) *Account {
	var user *Account
	err := json.NewDecoder(data).Decode(&user)
	if err != nil {
		return nil
	}
	return user
}

// ToJSON converts user to json string
func (u *Credentials) ToJSON() string {
	json, _ := json.Marshal(u)
	return string(json)
}

// CredentialsFromJReader create User from io.Reader
func CredentialsFromJReader(data io.Reader) *Credentials {
	var creds *Credentials
	err := json.NewDecoder(data).Decode(&creds)
	if err != nil {
		return nil
	}
	return creds
}

// CredentialsFromJSON create UserAuth from string
func CredentialsFromJSON(data string) *Credentials {
	var creds *Credentials
	json.Unmarshal([]byte(data), &creds)
	return creds
}

type AuthResponse struct {
	User    *Account
	Profile int
	ErrCode int
	ErrMsg  string
	Error   error
}
