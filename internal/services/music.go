package services

import (
	"context"
	"dpm/internal/models"
	"fmt"
	"log/slog"
)

type RepoMusic interface {
	CreateMusic(ctx context.Context, product models.Music) error
	GetMusic(ctx context.Context, id string) (models.Music, error)
	GetAllMusic(ctx context.Context) ([]models.Music, error)
}

type MusicService struct {
	repo RepoMusic
}

func NewMusicService(repo RepoMusic) *MusicService {
	return &MusicService{
		repo: repo,
	}
}

func (s *MusicService) CreateMusic(ctx context.Context, product models.Music) error {
	const op = "./internal/services/music.go.CreateMusic()"

	err := s.repo.CreateMusic(ctx, product)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *MusicService) GetMusic(ctx context.Context, id string) (models.Music, error) {
	const op = "./internal/services/music.go.GetMusic()"

	product, err := s.repo.GetMusic(ctx, id)
	if err != nil {
		slog.Info(err.Error())
		return models.Music{}, fmt.Errorf("%s: %w", op, err)
	}

	return product, nil
}

func (s *MusicService) GetAllMusic(ctx context.Context) ([]models.Music, error) {
	const op = "./internal/services/music.go.GetAllProducts()"

	p, err := s.repo.GetAllMusic(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return p, nil
}
