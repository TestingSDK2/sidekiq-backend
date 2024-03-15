package jwtauth

import (
	"time"

	"github.com/ProImaging/sidekiq-backend/sidekiq-auth-server/app/config"
	"github.com/ProImaging/sidekiq-backend/sidekiq-auth-server/cache"
	"github.com/ProImaging/sidekiq-backend/sidekiq-auth-server/database"
	repo "github.com/ProImaging/sidekiq-backend/sidekiq-auth-server/model"
	"github.com/ProImaging/sidekiq-backend/sidekiq-models/model"

	"github.com/dgrijalva/jwt-go"
)

type Service interface {
	FetchJWTToken(token string) (*Claims, error)
	CreateJWTToken(UserID int, tokenExpiration time.Duration, JWTKey string) (*JWTToken, error)
}

type service struct {
	config    *config.Config
	dbMaster  *database.Database
	dbReplica *database.Database
	cache     *cache.Cache
}

func NewService(repos *repo.Repos, conf *config.Config) Service {
	svc := &service{
		config:    conf,
		dbMaster:  repos.MasterDB,
		dbReplica: repos.ReplicaDB,
		cache:     repos.Cache,
	}
	return svc
}

func (s *service) FetchJWTToken(token string) (*Claims, error) {
	claims, err := fetchJWTToken(token, s.config.JWTKey)
	if err != nil {
		return nil, err
	}

	return claims, nil
}

func (s *service) CreateSignupJWTToken(user *model.AccountSignup, tokenExpiration time.Duration) (*JWTToken, error) {
	expirationTime := time.Now().Add(tokenExpiration * time.Hour)
	claims := &Claims{
		UserID: user.ID,
		StandardClaims: jwt.StandardClaims{
			// In JWT, the expiry time is expressed as unix milliseconds
			ExpiresAt: expirationTime.Unix(),
		},
	}

	// Declare the token with the algorithm used for signing, and the claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// Create the JWT string
	tokenString, err := token.SignedString([]byte(s.config.JWTKey))
	if err != nil {
		return nil, err
	}
	return &JWTToken{
		Value:     tokenString,
		ExpiresAt: expirationTime,
	}, nil
}

func (s *service) CreateJWTToken(UserID int, tokenExpiration time.Duration, JWTKey string) (*JWTToken, error) {
	expirationTime := time.Now().Add(tokenExpiration * time.Hour)
	claims := &Claims{
		UserID: int(UserID),
		// Username: user.Username,
		StandardClaims: jwt.StandardClaims{
			// In JWT, the expiry time is expressed as unix milliseconds
			ExpiresAt: expirationTime.Unix(),
		},
	}

	// Declare the token with the algorithm used for signing, and the claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// Create the JWT string
	tokenString, err := token.SignedString([]byte(JWTKey))
	if err != nil {
		return nil, err
	}
	return &JWTToken{
		Value:     tokenString,
		ExpiresAt: expirationTime,
	}, nil
}
