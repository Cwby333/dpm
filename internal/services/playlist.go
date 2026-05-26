package services

import (
	"context"
	"dpm/internal/models"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

type PlaylistRepo interface {
	CreatePlaylist(ctx context.Context, p models.Playlist) error
	GetPlaylist(ctx context.Context, id string) (models.Playlist, error)
	GetPlaylistMusic(ctx context.Context, id string) ([]models.LikedTrack, error)
	GetUserPlaylists(ctx context.Context, userID string) ([]models.PlaylistInfo, error)
	GetPublicPlaylists(ctx context.Context) ([]models.PlaylistInfo, error)
	GetPublicPlaylistsByID(ctx context.Context, id string) ([]models.PlaylistInfo, error)
	AddMusicToPlaylist(ctx context.Context, playlistID string, musicID string) error
	DeletePlaylist(ctx context.Context, id string) error
	UpdatePlaylist(ctx context.Context, playlist models.PlaylistUpdate) (error)
}

type PlaylistService struct {
	repo PlaylistRepo
	s3   S3
}

func NewPlaylistService(r PlaylistRepo, s3 S3) *PlaylistService {
	return &PlaylistService{
		repo: r,
		s3:   s3,
	}
}

func (s *PlaylistService) Create(ctx context.Context, name string, uploaderID string, coverData []byte, coverContentType string, private bool) (string, error) {
	const op = "./internal/services/playlist.go.Create()"

	playlistID := uuid.NewString()

	coverKey := ""
	if len(coverData) > 0 {
		coverKey = playlistID + "-playlistImage"
		err := s.s3.UploadObject(ctx, coverKey, coverData, coverContentType)
		if err != nil {
			return "", fmt.Errorf("%s: %w", op, err)
		}
	}

	p := models.Playlist{
		ID:         playlistID,
		Name:       name,
		UploaderID: uploaderID,
		Cover:      coverKey,
		Private:    private,
	}

	err := s.repo.CreatePlaylist(ctx, p)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return playlistID, nil
}

func (s *PlaylistService) GetMyPlaylists(ctx context.Context, userID string) ([]models.PlaylistInfo, error) {
	const op = "./internal/services/playlist.go.GetMyPlaylists()"

	pl, err := s.repo.GetUserPlaylists(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	for i := range pl {
		if pl[i].Cover != "" {
			url, err := s.GetPlaylistCoverPresignURL(ctx, pl[i].Cover)
			if err == nil {
				pl[i].Cover = url
			}
		}
	}

	return pl, nil
}

func (s *PlaylistService) GetPublicPlaylists(ctx context.Context) ([]models.PlaylistInfo, error) {
	const op = "./internal/services/playlist.go.GetPublicPlaylists()"

	pl, err := s.repo.GetPublicPlaylists(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	for i := range pl {
		if pl[i].Cover != "" {
			url, err := s.GetPlaylistCoverPresignURL(ctx, pl[i].Cover)
			if err == nil {
				pl[i].Cover = url
			}
		}
	}

	return pl, nil
}

func (s *PlaylistService) GetPlaylist(ctx context.Context, id string) (models.Playlist, error) {
	const op = "./internal/services/playlist.go.GetPlaylist()"

	p, err := s.repo.GetPlaylist(ctx, id)
	if err != nil {
		return models.Playlist{}, fmt.Errorf("%s: %w", op, err)
	}

	if p.Cover != "" {
		url, err := s.GetPlaylistCoverPresignURL(ctx, p.Cover)
		if err == nil {
			p.Cover = url
		}
	}

	return p, nil
}

func (s *PlaylistService) GetPublicPlaylistsByID(ctx context.Context, id string) ([]models.PlaylistInfo, error) {
	const op = "./internal/services/playlist.go.GetPublicPlaylistsByID"

	p, err := s.repo.GetPublicPlaylistsByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	for i := range p {
		if p[i].Cover != "" {
			url, err := s.GetPlaylistCoverPresignURL(ctx, p[i].Cover)
			if err == nil {
				p[i].Cover = url
			}
		}
	}

	return p, nil
}

func (s *PlaylistService) GetPlaylistMusic(ctx context.Context, id string) ([]models.LikedTrack, error) {
	const op = "./internal/services/playlist.go.GetPlaylistMusic()"

	m, err := s.repo.GetPlaylistMusic(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	for i := range m {
		if m[i].MusicCover != "" {
			url, err := s.GetPlaylistCoverPresignURL(ctx, m[i].MusicCover)
			if err == nil {
				m[i].MusicCover = url
			}
		}
	}

	return m, nil
}

func (s *PlaylistService) DeletePlaylist(ctx context.Context, id string) error {
	const op = "./internal/services/playlist.go.DeletePlaylist()"

	err := s.repo.DeletePlaylist(ctx, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *PlaylistService) AddMusic(ctx context.Context, playlistID string, musicID string) error {
	const op = "./internal/services/playlist.go.AddMusic()"

	err := s.repo.AddMusicToPlaylist(ctx, playlistID, musicID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *PlaylistService) GetPlaylistCoverPresignURL(ctx context.Context, coverKey string) (string, error) {
	const op = "./internal/services/playlist.go.GetPlaylistCoverPresignURL()"

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

func (s *PlaylistService) UpdatePlaylist(ctx context.Context, playlist models.PlaylistUpdate) (error) {
	const op = "./internal/services/playlist.go.UpdatePlaylist()"

	err := s.repo.UpdatePlaylist(ctx, playlist)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}