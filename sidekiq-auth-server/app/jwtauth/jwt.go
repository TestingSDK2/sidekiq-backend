package jwtauth

import (
	"errors"
	"time"

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
