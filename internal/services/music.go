package services

import (
	"context"
	"dpm/internal/models"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/google/uuid"
)

const (
	songPostfix        = "-song"
	songImagePostfix   = "-songImage"
)

type RepoMusic interface {
	CreateMusic(ctx context.Context, product models.Music) error
	GetMusic(ctx context.Context, id string, userID string) (models.Music, models.Like, error)
	GetAllMusic(ctx context.Context, u models.User) ([]models.Music, []models.Like, error)
}

type S3 interface {
	UploadObject(ctx context.Context, key string, data []byte, contentType string) error
	GetObject(ctx context.Context, key string, w io.WriterAt) error
	DeleteObject(ctx context.Context, key string) error
	GetPresignURL(ctx context.Context, id string) (string, error)
}

type MusicService struct {
	repo RepoMusic
	s3 S3
}

func NewMusicService(repo RepoMusic, s3 S3) *MusicService {
	return &MusicService{
		repo: repo,
		s3: s3,
	}
}

func (s *MusicService) CreateMusic(ctx context.Context, songID string, product models.Music) error {
	const op = "./internal/services/music.go.CreateMusic()"

	slog.Info(fmt.Sprintf("%+v", product))

	err := s.repo.CreateMusic(ctx, product)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *MusicService) GetMusic(ctx context.Context, id string, userID string) (models.Music, models.Like, error) {
	const op = "./internal/services/music.go.GetMusic()"

	product, like, err := s.repo.GetMusic(ctx, id, userID)
	if err != nil {
		slog.Info(err.Error())
		return models.Music{}, models.Like{}, fmt.Errorf("%s: %w", op, err)
	}

	return product, like, nil
}

func (s *MusicService) GetAllMusic(ctx context.Context, u models.User) ([]models.Music, []models.Like, error) {
	const op = "./internal/services/music.go.GetAllProducts()"

	p, l, err := s.repo.GetAllMusic(ctx, u)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", op, err)
	}

	return p, l, nil
}

// func (s *MusicService) UploadSong(ctx context.Context, data []byte, contentType string) (string, error) {
// 	const op = "./internal/services/music.go.UploadMusic()"

// 	songID := uuid.NewString()

// 	err := s.s3.UploadObject(ctx, songID, data, contentType)
// 	if err != nil {
// 		return "", fmt.Errorf("%s: %w", op, err)
// 	}

// 	return songID, nil
// }

// func (s *MusicService) UploadMusicCover(ctx context.Context, key string, data []byte, contentType string) error {
// 	const op = "./internal/services/music.go.UploadMusicCover()"

// 	err := s.s3.UploadObject(ctx, key, data, contentType)
// 	if err != nil {
// 		return fmt.Errorf("%s: %w", op, err)
// 	}

// 	return nil
// }

func (s *MusicService) UploadMusic(ctx context.Context, musicData map[string]models.DataAndCT, music models.Music) (error) {
	const op = "./internal/services/music.go.UploadSong()"

	musicID := uuid.NewString()

	songData, ok := musicData["songData"]
	if !ok {
		return fmt.Errorf("%s: %w", op, errors.New("Missing songData, upload is unreally"))
	}
	slog.Info(fmt.Sprint("UploadSong: songData:", fmt.Sprintf("%v, %v, %v", songData.Name, songData.ContentType, songData.Data[:100])))
	slog.Info("songDataSize", slog.Int("size", len(songData.Data)))

	songID := musicID + songPostfix
	err := s.s3.UploadObject(ctx, songID, songData.Data, songData.ContentType)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	coverData := musicData["coverData"]
	slog.Info(fmt.Sprintf("cover data size: %v", len(coverData.Data)))
	coverID := musicID + songImagePostfix
	err = s.s3.UploadObject(ctx, coverID, coverData.Data, songData.ContentType)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	music.SongURL =	songID
	music.CoverURL = coverID
	music.ID = musicID
	slog.Info(fmt.Sprintf("coverURL, songURL: %v, %v", music.CoverURL, music.SongURL))
	err = s.CreateMusic(ctx, "", music)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *MusicService) GetPresignURLSong(ctx context.Context, id string) (string, error) {
	const op = "./internal/services/music.go.GetPresignURL()"

	slog.Info("Get req presingURL")

	url, err := s.s3.GetPresignURL(ctx, id + songPostfix)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	slog.Info(fmt.Sprintf("GET PRESING URL: %v", url))

	return url, nil
}