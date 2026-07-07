package services

import (
	"context"
	"dpm/internal/models"
	"errors"
	"fmt"
	"log/slog"

	// "log/slog"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	avatarSuffix = "-avatar"
	defaultUserImageURL = "defaultUserImage.png"
)

type Pg interface {
	CreateUser(ctx context.Context, user models.User) error
	ReadPsw(ctx context.Context, user models.User) (string, error)
	ReadUserID(ctx context.Context, user models.User) (string, error)
	ReadUser(ctx context.Context, user models.User) (models.User, error)
	GetPublicUsers(ctx context.Context, uf models.UserFilter) ([]models.User, error)
	GetUserTracks(ctx context.Context, userID string) ([]models.LikedTrack, error)
}

type UserService struct {
	Pg  Pg
	S3  S3
	Key string
}

func NewUser(pg Pg, s3 S3, k string) *UserService {
	return &UserService{
		Pg:  pg,
		S3:  s3,
		Key: k,
	}
}

func (us *UserService) RegisterUser(ctx context.Context, u models.User, avatarData []byte, ctt string) error {
	const op = "./internal/services/user.go.RegisterUser()"

	userID := uuid.NewString()

	slog.Info(u.HashPsw)
	hash, err := bcrypt.GenerateFromPassword([]byte(u.HashPsw), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	slog.Info(string(hash), len(hash), len("$2a$10$Q24RiuCMdJmGNorSiPtQ5.Lh1z8.nF73r3P52lt2vwRwL38olJ54y"))

	u.HashPsw = string(hash)
	u.ID = userID

	if len(avatarData) == 0 {
		u.Image = defaultUserImageURL
	} else {
		u.Image = userID + avatarSuffix
	}

	err = us.S3.UploadObject(ctx, u.Image, avatarData, ctt)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

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

func (s *UserService) ReadUser(ctx context.Context, user models.User) (models.User, error) {
	const op = "./internal/services/user.go.ReadUser()"

	u, err := s.Pg.ReadUser(ctx, user)
	if err != nil {
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	url, err := s.S3.GetPresignURL(ctx, u.Image)
	if err != nil {
		return models.User{}, fmt.Errorf("%s: %w", op, fmt.Errorf("Get presign url for user image: %w", err))
	}

	u.Image = url

	return u, nil
}

func (s *UserService) GetPublicUsers(ctx context.Context, uf models.UserFilter) ([]models.User, error) {
	const op = "./internal/services/user.go.GetPublicUsers()"

	users, err := s.Pg.GetPublicUsers(ctx, uf)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	for i := range users {
		if users[i].Image != "" {
			url, err := s.S3.GetPresignURL(ctx, users[i].Image)
			if err == nil {
				users[i].Image = url
			}else {
				slog.Error(fmt.Sprintf("%s: %s", "Get presign url for user image: ", err.Error()))
			}
		}else {
			users[i].Image = defaultUserImageURL
			url, err := s.S3.GetPresignURL(ctx, defaultUserImageURL)
			if err == nil {
				users[i].Image = url
			}else {
				slog.Error(fmt.Sprintf("%s: %s", "Get presign url for default user image: ", err.Error()))
			}
		}
	}

	return users, nil
}

func (s *UserService) GetPublicUserProfile(ctx context.Context, userID string) (*models.User, []models.LikedTrack, error) {
	const op = "./internal/services/user.go.GetPublicUserProfile()"

	u, err := s.Pg.ReadUser(ctx, models.User{ID: userID})
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", op, err)
	}

	if u.PrivateProfile {
		return nil, nil, fmt.Errorf("%s: %w", op, errors.New("profile is private"))
	}

	url, err := s.S3.GetPresignURL(ctx, u.Image)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", "Get presign url for user image: ", err)
	}

	u.Image = url

	tracks, err := s.Pg.GetUserTracks(ctx, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", op, err)
	}

	for i := range tracks {
		if tracks[i].MusicCover != "" {
			url, err := s.S3.GetPresignURL(ctx, tracks[i].MusicCover)
			if err == nil {
				tracks[i].MusicCover = url
			}
		}
		if tracks[i].MusicSongURL != "" {
			url, err := s.S3.GetPresignURL(ctx, tracks[i].MusicSongURL)
			if err == nil {
				tracks[i].MusicSongURL = url
			}
		}
	}

	return &u, tracks, nil
}
