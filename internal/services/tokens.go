package services

import (
	"context"
	"dpm/internal/models"
	"errors"
	"fmt"
	"log/slog"
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
	slog.Info(fmt.Sprintf("createTokensSubject: %v", access.Subject))

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

func (s UserService) CheckAccessToken(ctx context.Context, token string) (jwt.MapClaims, error) {
	const op = "./internal/services/tokens.go.CheckAccessToken()"

	slog.Info("CheckAccessToken")

	sk := s.Key

	t, err := jwt.ParseWithClaims(token, jwt.MapClaims{}, func(t *jwt.Token) (any, error) {
		return []byte(sk), nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if t == nil {
		slog.Info("T is nil")
	}

	if !t.Valid {
		slog.Error("token invalid")
		return nil, fmt.Errorf("%s: %w", op, errors.New("token invalid"))
	}

	claims, ok := t.Claims.(jwt.MapClaims)
	if !ok {
		slog.Info("not jwtMapClaims")
		return jwt.MapClaims{}, nil
	}

	// for i := range claims {
	// 	slog.Info(i, claims[i])
	// }

	sub, err := claims.GetSubject()
	if err != nil {
		slog.Info("haven't subject")
		return claims, nil
	}
	_ = sub

	return claims, nil
}
