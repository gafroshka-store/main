package feedback

import (
	"database/sql"
	"gafroshka-main/internal/types/errors"
	"gafroshka-main/internal/types/feedback"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type FeedbackDBRepository struct {
	DB     *sql.DB
	Logger *zap.SugaredLogger
}

func NewFeedbackDBRepository(db *sql.DB, l *zap.SugaredLogger) *FeedbackDBRepository {
	return &FeedbackDBRepository{
		DB:     db,
		Logger: l,
	}
}

func (fr *FeedbackDBRepository) Create(f feedback.Feedback) (*feedback.Feedback, error) {
	f.ID = uuid.New().String()
	query := `
        INSERT INTO announcement_feedback (id, announcement_recipient_id, user_writer_id, comment, rating)
        VALUES ($1, $2, $3, $4, $5)
    `
	_, err := fr.DB.Exec(
		query,
		f.ID,
		f.AnnouncementID,
		f.UserWriterID,
		f.Comment,
		f.Rating,
	)
	if err != nil {
		fr.Logger.Warnf("Ошибка при создании отзыва: %v", err)
		return nil, errors.ErrDBInternal
	}

	return &f, nil
}

func (fr *FeedbackDBRepository) Delete(feedbackID string) error {
	query := `DELETE FROM announcement_feedback WHERE id = $1`

	result, err := fr.DB.Exec(query, feedbackID)
	if err != nil {
		fr.Logger.Warnf("Ошибка при удалении отзыва: %v", err)
		return errors.ErrDBInternal
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		fr.Logger.Warnf("Ошибка при проверке удаления отзыва: %v", err)
		return errors.ErrDBInternal
	}

	if rowsAffected == 0 {
		return errors.ErrNotFoundFeedback
	}

	return nil
}

func (fr *FeedbackDBRepository) GetByAnnouncementID(announcementID string) ([]feedback.Feedback, error) {
	query := `
		SELECT id, announcement_recipient_id, user_writer_id, comment, rating
		FROM announcement_feedback
		WHERE announcement_recipient_id = $1
	`

	rows, err := fr.DB.Query(query, announcementID)
	if err != nil {
		fr.Logger.Warnf("Ошибка при получении отзывов: %v", err)
		return nil, errors.ErrDBInternal
	}
	defer rows.Close()

	var feedbacks []feedback.Feedback
	for rows.Next() {
		var fb feedback.Feedback
		err := rows.Scan(
			&fb.ID,
			&fb.AnnouncementID,
			&fb.UserWriterID,
			&fb.Comment,
			&fb.Rating,
		)
		if err != nil {
			fr.Logger.Warnf("Ошибка при сканировании отзыва: %v", err)
			return nil, errors.ErrDBInternal
		}
		feedbacks = append(feedbacks, fb)
	}

	return feedbacks, nil
}
