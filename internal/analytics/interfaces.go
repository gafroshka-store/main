package analytics

import (
	"context"
	"gafroshka-main/internal/kafka"
)

// AnalyticsRepo — интерфейс репозитория для работы с предпочтениями пользователей.
type AnalyticsRepo interface {
	UpdatePreferences(ctx context.Context, userID string, weights map[int]int) error
	GetTopCategories(ctx context.Context, userID string, limit int) ([]int, error)
}

// AnalyticsService — интерфейс сервиса аналитики.
type AnalyticsService interface {
	ProcessEvent(ctx context.Context, event kafka.Event) error
	GetTopCategories(ctx context.Context, userID string, limit int) ([]int, error)
}
