package analytics

import (
	"context"
	"gafroshka-main/internal/kafka"
	"go.uber.org/zap"
)

// Service реализует интерфейс AnalyticsService.
type Service struct {
	repo   AnalyticsRepo
	logger *zap.SugaredLogger
}

func NewService(repo AnalyticsRepo, logger *zap.SugaredLogger) AnalyticsService {
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
	case kafka.Search:
		for _, cat := range event.Categories {
			weights[cat] += 1
		}
	case kafka.View:
		if len(event.Categories) > 0 {
			weights[event.Categories[0]] += 2
		}
	case kafka.AddToCart:
		if len(event.Categories) > 0 {
			weights[event.Categories[0]] += 3
		}
	case kafka.Purchase:
		if len(event.Categories) > 0 {
			weights[event.Categories[0]] += 5
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
