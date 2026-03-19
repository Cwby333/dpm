package services

import (
	"context"
	"dpm/internal/models"
	"fmt"
)

type ListeningHistoryRepo interface {
	ReadListeningHistory(ctx context.Context, lhi models.ListeningHistory) ([]models.ListeningHistoryResponse, error)
	CreateListeningHistoryItem(ctx context.Context, lhi models.ListeningHistory) (error)
	DeleteListeningHistoryItem(ctx context.Context, lhi models.ListeningHistory) (error)
}

type ListeningHistoryService struct {
	repo ListeningHistoryRepo
}

func NewListeningHistoryService(repo ListeningHistoryRepo) *ListeningHistoryService {
	return &ListeningHistoryService{
		repo: repo,
	}
}

func (ls *ListeningHistoryService) CreateListeningHistoryItem(ctx context.Context, lhi models.ListeningHistory) error {
	const op = "./internal/services/listeningHistory.go.CreateListenHistoryItem()"

	err := ls.repo.CreateListeningHistoryItem(ctx, lhi)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (ls ListeningHistoryService) DeleteListeningHistoryItem(ctx context.Context, lhi models.ListeningHistory) error {
	const op = "./internal/services/listeningHistory.go.DeleteListeningHistoryItem()"

	err := ls.repo.DeleteListeningHistoryItem(ctx, lhi)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (ls ListeningHistoryService) ReadListeningHistory(ctx context.Context, lhi models.ListeningHistory) ([]models.ListeningHistoryResponse, error) {
	const op = "./internal/services/listeningHistory.go.ReadListeningHistory()"

	lhr, err := ls.repo.ReadListeningHistory(ctx, lhi)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return lhr, nil
}