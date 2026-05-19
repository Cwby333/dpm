package services

import (
	"bytes"
	"context"
	"dpm/internal/models"
	"fmt"
	"log/slog"
	"math"

	"github.com/google/uuid"
	"github.com/tcolgate/mp3"
)

type AlbumRepo interface {
	CreateAlbum(ctx context.Context, album models.Album) error
	GetAlbum(ctx context.Context, id string) (models.Album, error)
	DeleteAlbum(ctx context.Context, id string) error
	GetAlbumsMusic(ctx context.Context, id string) ([]models.LikedTrack, error)
	GetAlbumInfo(ctx context.Context, id string) (models.AlbumInfo, error)
	GetAlbumsInfo(ctx context.Context) ([]models.AlbumInfo, error)
	AddMusicToAlbum(ctx context.Context, albumID string, musicID string) error
	CreateMusic(ctx context.Context, product models.Music) error
}

type AlbumsService struct {
	repo AlbumRepo
	s3   S3
}

func NewAlbumServices(repo AlbumRepo, s3 S3) *AlbumsService {
	return &AlbumsService{
		repo: repo,
		s3:   s3,
	}
}

func (s *AlbumsService) CreateAlbum(ctx context.Context, a models.Album) error {
	const op = "./internal/services/album.go.CreateAlbum()"

	err := s.repo.CreateAlbum(ctx, a)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *AlbumsService) GetAlbum(ctx context.Context, id string) (models.Album, error) {
	const op = "./internal/services/album.go.CreateAlbum()"

	a, err := s.repo.GetAlbum(ctx, id)
	if err != nil {
		return models.Album{}, fmt.Errorf("%s: %w", op, err)
	}

	return a, nil
}

func (s *AlbumsService) DeleteAlbum(ctx context.Context, id string) error {
	const op = "./internal/services/album.go.CreateAlbum()"

	err := s.repo.DeleteAlbum(ctx, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *AlbumsService) GetAlbumsMusic(ctx context.Context, id string) ([]models.LikedTrack, error) {
	const op = "./internal/services/album.go.CreateAlbum()"

	m, err := s.repo.GetAlbumsMusic(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return m, nil
}

func (s *AlbumsService) GetAlbumInfo(ctx context.Context, id string) (models.AlbumInfo, error) {
	const op = "./internal/services/album.go.GetAlbumInfo()"

	a, err := s.repo.GetAlbumInfo(ctx, id)
	if err != nil {
		return models.AlbumInfo{}, fmt.Errorf("%s: %w", op, err)
	}

	return a, nil
}

func (s *AlbumsService) GetAlbumsInfo(ctx context.Context) ([]models.AlbumInfo, error) {
	const op = "./internal/services/album.go.GetAlbumsInfo()"

	a, err := s.repo.GetAlbumsInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return a, nil
}

type SongUpload struct {
	Name string
	Data []byte
	ContentType string
}

func (s *AlbumsService) UploadAlbum(ctx context.Context, albumName string, uploaderID string, coverData []byte, coverContentType string, songs []SongUpload) (string, error) {
	const op = "./internal/services/album.go.UploadAlbum()"

	albumID := uuid.NewString()
	coverKey := albumID + "-albumImage"

	if len(coverData) > 0 {
		err := s.s3.UploadObject(ctx, coverKey, coverData, coverContentType)
		if err != nil {
			return "", fmt.Errorf("%s: %w", op, err)
		}
	}

	cover := ""
	if len(coverData) > 0 {
		cover = coverKey
	}

	album := models.Album{
		ID:          albumID,
		Name:        albumName,
		UploaderID:  uploaderID,
		Cover:       cover,
	}

	err := s.repo.CreateAlbum(ctx, album)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	for i := range songs {
		musicID := uuid.NewString()
		songKey := musicID + "-song"

		err := s.s3.UploadObject(ctx, songKey, songs[i].Data, songs[i].ContentType)
		if err != nil {
			return "", fmt.Errorf("%s: %w", op, err)
		}

		durSec := 0
		if len(songs[i].Data) > 0 {
			durSec = parseMP3Duration(songs[i].Data)
		}

		m := models.Music{
			ID:          musicID,
			Name:        songs[i].Name,
			UploaderID:  uploaderID,
			DurationSec: durSec,
			SongURL:     songKey,
			CoverURL:    coverKey,
		}

		err = s.repo.CreateMusic(ctx, m)
		if err != nil {
			return "", fmt.Errorf("%s: %w", op, err)
		}

		err = s.repo.AddMusicToAlbum(ctx, albumID, musicID)
		if err != nil {
			return "", fmt.Errorf("%s: %w", op, err)
		}
	}

	return albumID, nil
}

func (s *AlbumsService) GetAlbumCoverPresignURL(ctx context.Context, coverKey string) (string, error) {
	const op = "./internal/services/album.go.GetAlbumCoverPresignURL()"

	if coverKey == "" {
		return "", nil
	}

	slog.Info(coverKey)

	url, err := s.s3.GetPresignURL(ctx, coverKey)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return url, nil
}

func parseMP3Duration(data []byte) int {
	dec := mp3.NewDecoder(bytes.NewReader(data))
	var f mp3.Frame
	skipped := 0
	count := 0
	for {
		if err := dec.Decode(&f, &skipped); err != nil {
			break
		}
		count++
	}
	return int(math.Round((float64(count) * 26.0) / 1000.0))
}
