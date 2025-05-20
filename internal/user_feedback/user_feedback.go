package user_feedback

import (
	"context"
	types "gafroshka-main/internal/types/user_feedback"
)

// UserFeedback - структура отзыва на пользователя
type UserFeedback struct {
	ID              string `json:"feedback_id"`       // uuid
	UserRecipientID string `json:"user_recipient_id"` // uuid
	UserWriterID    string `json:"user_writer_id"`    // uuid
	Comment         string `json:"comment"`
	Rating          int    `json:"rating"`
}

type UserFeedbackRepo interface {
	// Create - создает новый отзыв на пользователя
	// Возвращает созданный UserFeedback
	Create(ctx context.Context, userFeedback *UserFeedback) (*UserFeedback, error)

	// GetByID - получает конкретный отзыв по ID
	// Возвращает отзыв на пользователя *UserFeedback
	GetByID(ctx context.Context, userFeedbackID string) (*UserFeedback, error)

	// GetByUserID - получает все отзывы на конкретного пользователя
	// Возвращает массив отзывов на пользователя []*UserFeedback
	GetByUserID(ctx context.Context, userRecipientID string) ([]*UserFeedback, error)

	// Update - обновляет существующий отзыв на пользователя
	// Возвращает обновленный UserFeedback
	Update(
		ctx context.Context,
		userFeedbackID string,
		updateUserFeedback types.UpdateUserFeedback,
	) (*UserFeedback, error)

	// Delete - удаляет существующий отзыв на пользователя
	// Возвращает error
	Delete(ctx context.Context, userFeedbackID string) error
}
