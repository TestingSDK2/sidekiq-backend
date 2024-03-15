package user

import (
	"errors"
	"time"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-models/model"
	"github.com/dgrijalva/jwt-go"
)

// Claims - a struct that will be encoded to JWT
type Claims struct {
	UserID   int    `json:"userID"`
	UserName string `json:"userName"`
	jwt.StandardClaims
}

// JWTToken - JWT Token
type JWTToken struct {
	Value     string
	ExpiresAt time.Time
}

func (s *service) ValidateJWTToken(token string) (*model.Account, error) {
	claims, err := fetchJWTToken(token, s.config.JWTKey)
	if err != nil {
		return nil, err
	}

	cachedUser, err := s.FetchCachedUser(claims.UserID)
	if err != nil {
		return nil, err
	}
	if cachedUser == nil {
		return nil, errors.New("cachedUser empty")
	}
	cachedUser.Token = token

	// check from db

	return cachedUser, nil
}

func (s *service) CreateJWTToken(user *model.Account, tokenExpiration time.Duration) (*JWTToken, error) {
	expirationTime := time.Now().Add(tokenExpiration * time.Hour)
	claims := &Claims{
		UserID: user.ID,
		// Username: user.Username,
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

func (s *service) ValidateSignupJWTToken(token string) (*model.AccountSignup, error) {
	claims, err := fetchJWTToken(token, s.config.JWTKey)
	if err != nil {
		return nil, err
	}

	cachedUser := s.FetchSignupCachedUser(claims.UserID)
	if cachedUser == nil {
		return nil, jwt.ErrSignatureInvalid
	}
	cachedUser.Token = token

	return cachedUser, nil
}

func fetchJWTToken(tokenStr string, jwtKey string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtKey), nil
	})

	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("the JWT Token is invalid")
	}

	return claims, nil
}
