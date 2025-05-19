package userFeedback

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"strconv"
	"strings"

	errors "gafroshka-main/internal/types/errors"
	types "gafroshka-main/internal/types/userFeedback"
)

type UserFeedbackRepository struct {
	DB     *sql.DB
	Logger *zap.SugaredLogger
}

func NewUserFeedbackRepository(DB *sql.DB, logger *zap.SugaredLogger) *UserFeedbackRepository {
	return &UserFeedbackRepository{
		DB:     DB,
		Logger: logger,
	}
}

func (userFeedbackRepository *UserFeedbackRepository) Create(
	ctx context.Context,
	userFeedback *UserFeedback,
) (string, error) {
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

		return "", errors.ErrDBInternal
	}

	userFeedbackRepository.Logger.Info(
		fmt.Sprintf("User feedback with userFeedbackID %s created successfully", userFeedback.ID),
	)

	return userFeedback.ID, nil
}

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

		return nil, errors.ErrDBInternal
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

			return nil, errors.ErrDBInternal
		}

		userFeedbacks = append(userFeedbacks, &userFeedback)
	}

	if err := rows.Err(); err != nil {
		userFeedbackRepository.Logger.Error(
			"Error occurred while iterating over user feedback rows from DB",
			zap.Error(err),
			zap.String("userID", userRecipientID),
		)

		return nil, errors.ErrDBInternal
	}

	return userFeedbacks, nil
}

func (userFeedbackRepository *UserFeedbackRepository) Update(
	ctx context.Context,
	userFeedbackID string,
	updateUserFeedback types.UpdateUserFeedback,
) error {
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
		return nil // Если ничего не обновляется, просто вернуть nil
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

		return errors.ErrDBInternal
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		userFeedbackRepository.Logger.Error(
			"Failed to get rows affected while updating user feedback",
			zap.Error(err),
			zap.String("userFeedbackID", userFeedbackID),
		)

		return errors.ErrDBInternal
	}

	if rowsAffected != 1 {
		userFeedbackRepository.Logger.Info(
			fmt.Sprintf("No user feedback with feedbackID %s found to update", userFeedbackID),
		)

		return errors.ErrNotFoundUserFeedback
	}

	userFeedbackRepository.Logger.Info(
		fmt.Sprintf("User feedback with userFeedbackID %s updated successfully", userFeedbackID),
	)

	return nil
}

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

		return errors.ErrDBInternal
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		userFeedbackRepository.Logger.Error(
			"Failed to get rows affected while deleting user feedback",
			zap.Error(err),
			zap.String("userFeedbackID", userFeedbackID),
		)

		return errors.ErrDBInternal
	}

	if rowsAffected != 1 {
		userFeedbackRepository.Logger.Info(
			fmt.Sprintf("No user feedback with feedbackID %s found to delete", userFeedbackID),
		)

		return errors.ErrNotFoundUserFeedback
	}

	userFeedbackRepository.Logger.Info(
		fmt.Sprintf("User feedback with userFeedbackID %s deleted successfully", userFeedbackID),
	)

	return nil
}
