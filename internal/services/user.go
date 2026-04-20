package services

import (
	"context"
	"dpm/internal/models"
	"fmt"
	"log/slog"

	// "log/slog"

	"golang.org/x/crypto/bcrypt"
)

type Pg interface {
	CreateUser(ctx context.Context, user models.User) error
	ReadPsw(ctx context.Context, user models.User) (string, error)
	ReadUserID(ctx context.Context, user models.User) (string, error)
}

type UserService struct {
	Pg  Pg
	Key string
}

func NewUser(pg Pg, k string) *UserService {
	return &UserService{
		Pg:  pg,
		Key: k,
	}
}

func (us *UserService) RegisterUser(ctx context.Context, u models.User) error {
	const op = "./internal/services/user.go.RegisterUser()"

	slog.Info(u.HashPsw)
	hash, err := bcrypt.GenerateFromPassword([]byte(u.HashPsw), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	slog.Info(string(hash), len(hash), len("$2a$10$Q24RiuCMdJmGNorSiPtQ5.Lh1z8.nF73r3P52lt2vwRwL38olJ54y"))

	u.HashPsw = string(hash)

	err = us.Pg.CreateUser(ctx, u)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (us *UserService) Login(ctx context.Context, u models.User) (models.JWTAccess, models.JWTRefresh, error) {
	const op = "./internal/services/user.go.Login()"

	hashPsw, err := us.Pg.ReadPsw(ctx, u)
	if err != nil {
		return models.JWTAccess{}, models.JWTRefresh{}, fmt.Errorf("%s: %w", op, err)
	}

	id, err := us.Pg.ReadUserID(ctx, u)
	if err != nil {
		return models.JWTAccess{}, models.JWTRefresh{}, fmt.Errorf("%s: %w", op, err)
	}
	u.ID = id

	slog.Info(u.HashPsw, hashPsw)
	err = bcrypt.CompareHashAndPassword([]byte(hashPsw), []byte(u.HashPsw))
	if err != nil {
		return models.JWTAccess{}, models.JWTRefresh{}, fmt.Errorf("%s CompareHash: %w", op, err)
	}
	slog.Info("Login subject " + u.ID)

	access, refresh, err := us.createTokens(ctx, u)
	slog.Info(fmt.Sprint("access token " + access.Sign))
	slog.Info(fmt.Sprint("access refresh " + refresh.Sign))

	return access, refresh, nil
}
