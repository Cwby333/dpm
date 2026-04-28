package services

import (
	"context"
	"dpm/internal/models"
	"fmt"
)

type LikeRepo interface {
	CreateLike(ctx context.Context, l models.Like) error
	ReadLikes(ctx context.Context, l models.Like) ([]models.Like, error)
	DeleteLike(ctx context.Context, l models.Like) error
	ReadLikedTracks(ctx context.Context, u models.User) ([]models.LikedTrack, error)
}

type LikeService struct {
	repo LikeRepo
}

func NewLikeService(r LikeRepo) *LikeService {
	return &LikeService{
		repo: r,
	}
}

func (s *LikeService) CreateLike(ctx context.Context, l models.Like) error {
	const op = "./internal/services/likes.go.CreateLike()"

	err := s.repo.CreateLike(ctx, l)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *LikeService) ReadLikes(ctx context.Context, l models.Like) ([]models.Like, error) {
	const op = "./internal/services/likes.go.ReadLikes()"

	li, err := s.repo.ReadLikes(ctx, l)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return li, nil
}

func (s *LikeService) DeleteLike(ctx context.Context, l models.Like) error {
	const op = "./internal/services/likes.go.DeleteLike()"

	err := s.repo.DeleteLike(ctx, l)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *LikeService) ReadLikedTracks(ctx context.Context, u models.User) ([]models.LikedTrack, error) {
	const op = "./internal/services/likes.go.ReadLikedTracks"

	l, err := s.repo.ReadLikedTracks(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return l, nil
}
