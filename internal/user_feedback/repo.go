package userFeedback

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"strconv"
	"strings"

	myErr "gafroshka-main/internal/types/errors"
	types "gafroshka-main/internal/types/user_feedback"
)

type UserFeedbackRepository struct {
	DB     *sql.DB
	Logger *zap.SugaredLogger
}

func NewUserFeedbackRepository(db *sql.DB, logger *zap.SugaredLogger) *UserFeedbackRepository {
	return &UserFeedbackRepository{
		DB:     db,
		Logger: logger,
	}
}

// Create - создает новый отзыв на пользователя
// Возвращает созданный UserFeedback
func (userFeedbackRepository *UserFeedbackRepository) Create(
	ctx context.Context,
	userFeedback *UserFeedback,
) (*UserFeedback, error) {
	userFeedback.ID = uuid.New().String()

	query :=
		`
		INSERT INTO user_feedback (id, user_recipient_id, user_writer_id, comment, rating)
		VALUES ($1, $2, $3, $4, $5)
		`

	_, err := userFeedbackRepository.DB.ExecContext(
		ctx,
		query,
		userFeedback.ID,
		userFeedback.UserRecipientID,
		userFeedback.UserWriterID,
		userFeedback.Comment,
		userFeedback.Rating,
	)

	if err != nil {
		userFeedbackRepository.Logger.Error(
			"Failed save user feedback to DB",
			zap.Error(err),
			zap.String("userFeedbackID", userFeedback.ID),
		)

		return nil, myErr.ErrDBInternal
	}

	userFeedbackRepository.Logger.Info(
		fmt.Sprintf("User feedback with userFeedbackID %s created successfully", userFeedback.ID),
	)

	return userFeedback, nil
}

// GetByID - получает конкретный отзыв по ID
// Возвращает отзыв на пользователя *UserFeedback
func (userFeedbackRepository *UserFeedbackRepository) GetByID(
	ctx context.Context,
	userFeedbackID string,
) (*UserFeedback, error) {
	query :=
		`
		SELECT id, user_recipient_id, user_writer_id, comment, rating
		FROM user_feedback
		WHERE id = $1
 		`

	userFeedback := &UserFeedback{}
	err := userFeedbackRepository.DB.
		QueryRow(query, userFeedbackID).
		Scan(
			&userFeedback.ID,
			&userFeedback.UserRecipientID,
			&userFeedback.UserWriterID,
			&userFeedback.Comment,
			&userFeedback.Rating,
		)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, myErr.ErrNotFound
		}
		userFeedbackRepository.Logger.Warnf("Error while load userFeedback info: %v", err)

		return nil, myErr.ErrDBInternal
	}

	return userFeedback, nil
}

// GetByUserID - получает все отзывы на конкретного пользователя
// Возвращает массив отзывов на пользователя []*UserFeedback
func (userFeedbackRepository *UserFeedbackRepository) GetByUserID(
	ctx context.Context,
	userRecipientID string,
) ([]*UserFeedback, error) {
	query :=
		`
		SELECT id, user_recipient_id, user_writer_id, comment, rating
		FROM user_feedback
		WHERE user_recipient_id=$1
		`

	rows, err := userFeedbackRepository.DB.QueryContext(ctx, query, userRecipientID)
	if err != nil {
		userFeedbackRepository.Logger.Error(
			"Failed to get user feedback from DB",
			zap.Error(err),
			zap.String("userID", userRecipientID),
		)

		return nil, myErr.ErrDBInternal
	}
	defer rows.Close()

	var userFeedbacks []*UserFeedback
	for rows.Next() {
		var userFeedback UserFeedback
		err := rows.Scan(
			&userFeedback.ID,
			&userFeedback.UserRecipientID,
			&userFeedback.UserWriterID,
			&userFeedback.Comment,
			&userFeedback.Rating,
		)
		if err != nil {
			userFeedbackRepository.Logger.Error(
				"Failed to scan user feedback row from DB",
				zap.Error(err),
			)

			return nil, myErr.ErrDBInternal
		}

		userFeedbacks = append(userFeedbacks, &userFeedback)
	}

	if err := rows.Err(); err != nil {
		userFeedbackRepository.Logger.Error(
			"Error occurred while iterating over user feedback rows from DB",
			zap.Error(err),
			zap.String("userID", userRecipientID),
		)

		return nil, myErr.ErrDBInternal
	}

	return userFeedbacks, nil
}

// Update - обновляет существующий отзыв на пользователя
// Возвращает обновленный UserFeedback
func (userFeedbackRepository *UserFeedbackRepository) Update(
	ctx context.Context,
	userFeedbackID string,
	updateUserFeedback types.UpdateUserFeedback,
) (*UserFeedback, error) {
	fields := []string{}
	args := []interface{}{}
	argID := 1

	// Динамически добавляем поля в обновление
	if updateUserFeedback.Comment != "" {
		fields = append(fields, "comment = $"+strconv.Itoa(argID))
		args = append(args, updateUserFeedback.Comment)
		argID++
	}
	if updateUserFeedback.Rating != 0 {
		fields = append(fields, "rating = $"+strconv.Itoa(argID))
		args = append(args, updateUserFeedback.Rating)
		argID++
	}

	if len(fields) == 0 {
		return nil, nil // Если ничего не обновляется, просто вернуть nil, nil
	}

	query := "UPDATE user_feedback SET " + strings.Join(fields, ", ") +
		" WHERE id = $" + strconv.Itoa(argID) //nolint:gosec
	args = append(args, userFeedbackID)

	result, err := userFeedbackRepository.DB.Exec(query, args...)
	if err != nil {
		userFeedbackRepository.Logger.Error(
			"Failed to update user feedback",
			zap.Error(err),
			zap.String("userFeedbackID", userFeedbackID),
		)

		return nil, myErr.ErrDBInternal
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		userFeedbackRepository.Logger.Error(
			"Failed to get rows affected while updating user feedback",
			zap.Error(err),
			zap.String("userFeedbackID", userFeedbackID),
		)

		return nil, myErr.ErrDBInternal
	}

	if rowsAffected != 1 {
		userFeedbackRepository.Logger.Info(
			fmt.Sprintf("No user feedback with feedbackID %s found to update", userFeedbackID),
		)

		return nil, myErr.ErrNotFoundUserFeedback
	}

	userFeedbackRepository.Logger.Info(
		fmt.Sprintf("User feedback with userFeedbackID %s updated successfully", userFeedbackID),
	)

	return userFeedbackRepository.GetByID(ctx, userFeedbackID)
}

// Delete - удаляет существующий отзыв на пользователя
// Возвращает error
func (userFeedbackRepository *UserFeedbackRepository) Delete(
	ctx context.Context,
	userFeedbackID string,
) error {
	query :=
		`
		DELETE FROM user_feedback
		WHERE id=$1
		`

	result, err := userFeedbackRepository.DB.ExecContext(ctx, query, userFeedbackID)
	if err != nil {
		userFeedbackRepository.Logger.Error(
			"Failed to delete user feedback",
			zap.Error(err),
			zap.String("userFeedbackID", userFeedbackID),
		)

		return myErr.ErrDBInternal
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		userFeedbackRepository.Logger.Error(
			"Failed to get rows affected while deleting user feedback",
			zap.Error(err),
			zap.String("userFeedbackID", userFeedbackID),
		)

		return myErr.ErrDBInternal
	}

	if rowsAffected != 1 {
		userFeedbackRepository.Logger.Info(
			fmt.Sprintf("No user feedback with feedbackID %s found to delete", userFeedbackID),
		)

		return myErr.ErrNotFoundUserFeedback
	}

	userFeedbackRepository.Logger.Info(
		fmt.Sprintf("User feedback with userFeedbackID %s deleted successfully", userFeedbackID),
	)

	return nil
}
