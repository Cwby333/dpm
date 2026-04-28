package services

import (
	"context"
	"dpm/internal/models"
	"fmt"
)

type FavorRepo interface {
	CreateFavor(ctx context.Context, lh models.ListeningHistory) error
	ReadFavor(ctx context.Context, lhi models.ListeningHistory) ([]models.ListeningHistoryResponse, error)
	DeleteFavor(ctx context.Context, lhi models.ListeningHistory) error
}

type FavorService struct {
	repo FavorRepo
}

func NewFavorService(repo FavorRepo) *FavorService {
	return &FavorService{
		repo: repo,
	}
}

func (s FavorService) CreateFavor(ctx context.Context, lh models.ListeningHistory) error {
	const op = "./internal/services/favor.go.CreateFavor()"

	err := s.repo.CreateFavor(ctx, lh)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s FavorService) ReadFavor(ctx context.Context, lh models.ListeningHistory) ([]models.ListeningHistoryResponse, error) {
	const op = "./internal/services/favor.go.ReadFavor()"

	slice, err := s.repo.ReadFavor(ctx, lh)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return slice, nil
}

func (s FavorService) DeleteFavor(ctx context.Context, lhi models.ListeningHistory) error {
	const op = "./internal/services/favor.go.DeleteFavor()"

	err := s.repo.DeleteFavor(ctx, lhi)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
