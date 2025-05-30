package analytics

import (
	"context"
	"gafroshka-main/internal/kafka"
	"go.uber.org/zap"
)

type Service struct {
	repo   *Repository
	logger *zap.SugaredLogger
}

func NewService(repo *Repository, logger *zap.SugaredLogger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

func (s *Service) ProcessEvent(ctx context.Context, event kafka.Event) error {
	if event.UserID == "" {
		return nil // Игнорируем события без пользователя
	}

	weights := make(map[int]int)
	switch event.Type {
	case kafka.EventTypeSearch:
		for _, cat := range event.Categories {
			weights[cat] += 1
		}
	case kafka.EventTypeView:
		if len(event.Categories) > 0 {
			weights[event.Categories[0]] += 2
		}
	case kafka.EventTypePurchase:
		if len(event.Categories) > 0 {
			weights[event.Categories[0]] += 3
		}
	}

	if len(weights) == 0 {
		return nil
	}

	return s.repo.UpdatePreferences(ctx, event.UserID, weights)
}

func (s *Service) GetTopCategories(ctx context.Context, userID string, limit int) ([]int, error) {
	return s.repo.GetTopCategories(ctx, userID, limit)
}
