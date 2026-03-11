package services

import (
	"context"
	"dpm/internal/models"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/golang-jwt/jwt/v5"
)

func (s UserService) createTokens(ctx context.Context, user models.User) (access models.JWTAccess, refresh models.JWTRefresh, err error) {
	const op = "./internal/service/userService/tokens.go.createTokens"

	accessTokenID := uuid.NewString()
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, models.JWTAccess{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "muteproject issuer",
			Subject:   user.ID,
			NotBefore: jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        accessTokenID,
		},
		Type:    "access",
		Role:    "user",
		TokenID: accessTokenID,
	})

	accessSign, err := accessToken.SignedString([]byte(s.Key))
	if err != nil {
		return models.JWTAccess{}, models.JWTRefresh{}, fmt.Errorf("%s: %w", op, err)
	}
	access = accessToken.Claims.(models.JWTAccess)
	access.Sign = accessSign
	access.Type = "access"

	refreshTokenID := uuid.NewString()
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, models.JWTRefresh{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "muteproject issuer",
			Subject:   user.ID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
			NotBefore: jwt.NewNumericDate(time.Now()),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        refreshTokenID,
		},
		Type:    "refresh",
		Role:    "user",
		TokenID: refreshTokenID,
	})
	refreshSign, err := refreshToken.SignedString([]byte(s.Key))

	if err != nil {
		return models.JWTAccess{}, models.JWTRefresh{}, fmt.Errorf("%s: %w", op, err)
	}

	refresh = refreshToken.Claims.(models.JWTRefresh)
	refresh.Sign = refreshSign
	refresh.Type = "refresh"

	return access, refresh, nil
}
