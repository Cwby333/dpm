package services

import (
	"context"
	"dpm/internal/models"
	"fmt"
)

type AlbumRepo interface {
	CreateAlbum(ctx context.Context, album models.Album) error
	GetAlbum(ctx context.Context, id string) (models.Album, error)
	DeleteAlbum(ctx context.Context, id string) error
	GetAlbumsMusic(ctx context.Context, id string) ([]models.LikedTrack, error)
	GetAlbumInfo(ctx context.Context, id string) (models.AlbumInfo, error)
	GetAlbumsInfo(ctx context.Context) ([]models.AlbumInfo, error)
}

type AlbumsService struct {
	repo AlbumRepo
}

func NewAlbumServices(repo AlbumRepo) *AlbumsService {
	return &AlbumsService{
		repo: repo,
	}
}

func (s *AlbumsService) CreateAlbum(ctx context.Context, a models.Album) (error) {
	const op = "./internal/services/album.go.CreateAlbum()"

	err := s.repo.CreateAlbum(ctx, a)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *AlbumsService) GetAlbum(ctx context.Context,id string) (models.Album, error) {
	const op = "./internal/services/album.go.CreateAlbum()"

	a, err := s.repo.GetAlbum(ctx, id)
	if err != nil {
		return models.Album{}, fmt.Errorf("%s: %w", op, err)
	}

	return a, nil
}

func (s *AlbumsService) DeleteAlbum(ctx context.Context, id string) (error) {
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