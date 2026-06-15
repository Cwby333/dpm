package services

import (
	"bytes"
	"context"
	"dpm/internal/models"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/tcolgate/mp3"
)

type AlbumRepo interface {
	CreateAlbum(ctx context.Context, album models.Album) error
	GetAlbum(ctx context.Context, id string) (models.Album, error)
	DeleteAlbum(ctx context.Context, id string) error
	GetAlbumsMusic(ctx context.Context, id string) ([]models.LikedTrack, error)
	GetAlbumInfo(ctx context.Context, id string) (models.AlbumInfo, error)
	GetAlbumsInfo(ctx context.Context, af models.AlbumFilter) ([]models.AlbumInfo, error)
	GetUserAlbums(ctx context.Context, userID string) ([]models.AlbumInfo, error)
	AddMusicToAlbum(ctx context.Context, albumID string, musicID string) error
	GetUploadedByUserAlbums(ctx context.Context, id string, af models.AlbumFilter) ([]models.Album, error)
	CreateMusic(ctx context.Context, product models.Music) error
	UpdateAlbum(ctx context.Context, album models.Album) (error)
	LikeAlbum(ctx context.Context, albumID string, userID string) (error)
	DeleteLikeAlbum(ctx context.Context, albumID string, userID string) (error)
	GetLikedAlbums(ctx context.Context, userID string, af models.AlbumFilter) ([]models.AlbumInfo, error)
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

func (s *AlbumsService) DeleteAlbum(ctx context.Context, id string, userID string) error {
	const op = "./internal/services/album.go.DeleteAlbum()"

	a, err := s.repo.GetAlbum(ctx, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if a.UploaderID != userID {
		return fmt.Errorf("%s: %w", op, err)
	}

	err = s.repo.DeleteAlbum(ctx, id)
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

func (s *AlbumsService) GetAlbumsInfo(ctx context.Context, af models.AlbumFilter) ([]models.AlbumInfo, error) {
	const op = "./internal/services/album.go.GetAlbumsInfo()"

	a, err := s.repo.GetAlbumsInfo(ctx, af)
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
		
		tmpDir, err := os.MkdirTemp("", musicID)
		if err != nil {
			return "", fmt.Errorf("%s: MkdirTemp %w", op, err)
		}
		
		mp3Path := filepath.Join(tmpDir, "input.mp3")
		err = os.WriteFile(mp3Path, songs[i].Data, 0644)
		if err != nil {
			return "", fmt.Errorf("%s: WriteTempFile %w", op, err)
		}

		cmd := convertToHLS(mp3Path, tmpDir)

		if cmd != nil {
			slog.Error(cmd.Error())
			return "", fmt.Errorf("%s: CMD %w", op, cmd)
		}

		files, err := os.ReadDir(tmpDir)
		if err != nil {
			return "", fmt.Errorf("%s: ReadTmpDir %w", op, err)
		}

		for _, f := range files {
			slog.Debug("UploadHLSMusic", slog.String("fName", f.Name()))

			data, err := os.ReadFile(filepath.Join(tmpDir, f.Name()))
			if err != nil {
				return "", fmt.Errorf("%s: ReadFile %w", op, err)
			}
			
			ct := "audio/mpeg"
			if strings.HasSuffix(f.Name(), "m3u8") {
				ct = "application/vnd.apple.mpegurl"
			}
			
			slog.Debug(albumID + "/" + musicID + "-hls/" + f.Name())
			err = s.s3.UploadObject(ctx, albumID + "/" + musicID + "-hls/" + f.Name(), data, ct)
			if err != nil {
				return "", fmt.Errorf("%s: UploadObject %w", op, err)
			}
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
			SongURL:     albumID + "/" + musicID + "-hls/playlist.m3u8",
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

		os.RemoveAll(tmpDir)
	}

	return albumID, nil
}

func (s *AlbumsService) GetUserAlbums(ctx context.Context, userID string) ([]models.AlbumInfo, error) {
	const op = "./internal/services/album.go.GetUserAlbums()"

	a, err := s.repo.GetUserAlbums(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	for i := range a {
		if a[i].Cover != "" {
			url, err := s.GetAlbumCoverPresignURL(ctx, a[i].Cover)
			if err == nil {
				a[i].Cover = url
			}
		}
	}

	return a, nil
}

func (s *AlbumsService) LikeAlbum(ctx context.Context, albumID string, userID string) (error) {
	const op = "./internal/services/album.go.LikeAlbum()"
	slog.Debug(op, slog.String("albumID", albumID), slog.String("userID", userID))

	err := s.repo.LikeAlbum(ctx, albumID, userID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *AlbumsService) DeleteLikeAlbum(ctx context.Context, albumID string, userID string) (error) {
	const op = "./internal/services/album.go.DeleteLikeAlbum()"
	slog.Debug(op, slog.String("albumID", albumID), slog.String("userID", userID))

	err := s.repo.DeleteLikeAlbum(ctx, albumID, userID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *AlbumsService) GetLikedAlbums(ctx context.Context, userID string, af models.AlbumFilter) ([]models.AlbumInfo, error) {
	const op = "./internal/services/album.go.GetLikedAlbums()"
	slog.Debug(op, slog.String("userID", userID))

	al, err := s.repo.GetLikedAlbums(ctx, userID, af)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	for i := range al {
		if al[i].Cover != "" {
			url, err := s.GetAlbumCoverPresignURL(ctx, al[i].Cover)
			if err == nil {
				al[i].Cover = url
			}
		}
	}

	return al, nil
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

func (s *AlbumsService) GetUploadedByUserAlbums(ctx context.Context, id string, af models.AlbumFilter) ([]models.Album, error) {
	const op = "./internal/services/album.go.GetUploadedByUserAlbums()"

	a, err := s.repo.GetUploadedByUserAlbums(ctx, id, af)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	for i := range a {
		if a[i].Cover != "" {
			url, err := s.s3.GetPresignURL(ctx, a[i].Cover)
			if err != nil {
				slog.Error("GetUploadedByUserAlbums s3", slog.String("error", err.Error()))
				continue
			}

			a[i].Cover = url
		}
	}

	return a, nil
}

func (s *AlbumsService) UpdateAlbum(ctx context.Context, album models.Album, coverData []byte, ctt string, updaterID string) (error) {
	const op = "./internal/services/album.go.UpdateAlbum()"

	a, err := s.repo.GetAlbum(ctx, album.ID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if a.UploaderID != updaterID {
		return fmt.Errorf("%s: %w", op, errors.New("UpdateAlbum forbidden"))
	}

	coverKey := album.ID + "-albumImage"

	if len(coverData) > 0 {
		err := s.s3.UploadObject(ctx, coverKey, coverData, ctt)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	cover := ""
	if len(coverData) > 0 {
		cover = coverKey
		album.Cover = cover
	}

	err = s.repo.UpdateAlbum(ctx, album)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}