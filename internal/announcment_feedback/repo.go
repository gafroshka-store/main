package announcmentfeedback

import (
	"database/sql"
	"strings"

	"gafroshka-main/internal/types/errors"

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

func (fr *FeedbackDBRepository) Create(f Feedback) (Feedback, error) {
	f.ID = uuid.New().String()
	query := `
        INSERT INTO announcement_feedback (id, announcement_recipient_id, user_writer_id, comment, rating)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (announcement_recipient_id, user_writer_id) DO NOTHING
        RETURNING id
    `
	var insertedID string
	err := fr.DB.QueryRow(
		query,
		f.ID,
		f.AnnouncementID,
		f.UserWriterID,
		f.Comment,
		f.Rating,
	).Scan(&insertedID)

	if err != nil {
		// Если нет строк (ON CONFLICT DO NOTHING), Scan вернет sql.ErrNoRows
		if err == sql.ErrNoRows {
			return Feedback{}, errors.ErrAlreadyLeftFeedback
		}
		if strings.Contains(err.Error(), "ON CONFLICT specification") {
			fr.Logger.Warnf("ON CONFLICT constraint missing in DB: %v", err)
			return Feedback{}, errors.ErrAlreadyLeftFeedback
		}
		fr.Logger.Warnf("Ошибка при создании отзыва: %v", err)
		return Feedback{}, errors.ErrDBInternal
	}

	f.ID = insertedID
	return f, nil
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

	if rowsAffected != 1 {
		return errors.ErrNotFoundFeedback
	}

	return nil
}

func (fr *FeedbackDBRepository) GetByAnnouncementID(announcementID string) ([]Feedback, error) {
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

	var feedbacks []Feedback
	for rows.Next() {
		var fb Feedback
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

func (fr *FeedbackDBRepository) Update(feedbackID string, comment string, rating int) (Feedback, error) {
	query := `
        UPDATE announcement_feedback
        SET comment = $1, rating = $2
        WHERE id = $3
        RETURNING id, announcement_recipient_id, user_writer_id, comment, rating
    `
	var fb Feedback
	err := fr.DB.QueryRow(query, comment, rating, feedbackID).Scan(
		&fb.ID,
		&fb.AnnouncementID,
		&fb.UserWriterID,
		&fb.Comment,
		&fb.Rating,
	)
	if err != nil {
		fr.Logger.Warnf("Ошибка при обновлении отзыва: %v", err)
		return Feedback{}, errors.ErrDBInternal
	}
	return fb, nil
}
