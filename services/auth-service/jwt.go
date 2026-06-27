package main

import (
	"errors"
	"time"

	"core_project/shared/sdk/config"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	UserID         string `json:"userId"`
	TenantID       string `json:"tenantId"`
	Role           string `json:"role"`
	IsDataVerified bool   `json:"isDataVerified"`
	jwt.RegisteredClaims
}

type Tokens struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

func generateTokens(userID, tenantID, role string, isDataVerified bool) (*Tokens, error) {
	cfg := config.GlobalConfig
	if cfg == nil {
		return nil, errors.New("config not loaded")
	}
	secret := []byte(cfg.JWTSecret)

	// Access Token (15 mins)
	accessClaims := Claims{
		UserID:         userID,
		TenantID:       tenantID,
		Role:           role,
		IsDataVerified: isDataVerified,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.New().String(),
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessString, err := accessToken.SignedString(secret)
	if err != nil {
		return nil, err
	}

	// Refresh Token (7 days)
	refreshClaims := Claims{
		UserID:         userID,
		TenantID:       tenantID,
		Role:           role,
		IsDataVerified: isDataVerified,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.New().String(),
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshString, err := refreshToken.SignedString(secret)
	if err != nil {
		return nil, err
	}

	return &Tokens{
		AccessToken:  accessString,
		RefreshToken: refreshString,
	}, nil
}

func validateToken(tokenString string) (*Claims, error) {
	cfg := config.GlobalConfig
	if cfg == nil {
		return nil, errors.New("config not loaded")
	}
	secret := []byte(cfg.JWTSecret)

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
