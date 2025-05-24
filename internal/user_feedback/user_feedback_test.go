package userFeedback

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	customErrors "gafroshka-main/internal/types/errors"
	types "gafroshka-main/internal/types/user_feedback"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

func setupTestRepo(t *testing.T) (*UserFeedbackRepository, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}

	logger := zaptest.NewLogger(t).Sugar()
	repo := NewUserFeedbackRepository(db, logger)

	return repo, mock, func() { db.Close() }
}

func TestUserFeedbackRepository_Create(t *testing.T) {
	repo, mock, teardown := setupTestRepo(t)
	defer teardown()

	type args struct {
		feedback *UserFeedback
	}
	tests := []struct {
		name     string
		args     args
		mockFunc func()
		wantErr  error
	}{
		{
			name: "success",
			args: args{
				feedback: &UserFeedback{
					UserRecipientID: "recipient-uuid",
					UserWriterID:    "writer-uuid",
					Comment:         "Great job!",
					Rating:          5,
				},
			},
			mockFunc: func() {
				mock.ExpectExec("INSERT INTO user_feedback").
					WithArgs(sqlmock.AnyArg(), "recipient-uuid", "writer-uuid", "Great job!", 5).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: nil,
		},
		{
			name: "db error",
			args: args{
				feedback: &UserFeedback{
					UserRecipientID: "recipient-uuid",
					UserWriterID:    "writer-uuid",
					Comment:         "Oops!",
					Rating:          1,
				},
			},
			mockFunc: func() {
				mock.ExpectExec("INSERT INTO user_feedback").
					WillReturnError(errors.New("db error"))
			},
			wantErr: customErrors.ErrDBInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFunc()
			_, err := repo.Create(context.Background(), tt.args.feedback)

			assert.Equal(t, tt.wantErr, err)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserFeedbackRepository_GetByID(t *testing.T) {
	repo, mock, teardown := setupTestRepo(t)
	defer teardown()

	tests := []struct {
		name     string
		mockFunc func()
		inputID  string
		want     *UserFeedback
		wantErr  error
	}{
		{
			name:    "success",
			inputID: "id1",
			mockFunc: func() {
				rows := sqlmock.NewRows([]string{"id", "user_recipient_id", "user_writer_id", "comment", "rating"}).
					AddRow("id1", "recipient-uuid", "writer-uuid", "Nice work", 4)
				mock.ExpectQuery(`SELECT id, user_recipient_id, user_writer_id, comment, rating FROM user_feedback WHERE id = \$1`).
					WithArgs("id1").
					WillReturnRows(rows)
			},
			want: func() *UserFeedback {
				id := "id1"
				recipientID := "recipient-uuid"
				writerID := "writer-uuid"
				comment := "Nice work"
				rating := 4
				return &UserFeedback{
					ID:              id,
					UserRecipientID: recipientID,
					UserWriterID:    writerID,
					Comment:         comment,
					Rating:          rating,
				}
			}(),
			wantErr: nil,
		},
		{
			name:    "not found",
			inputID: "id2",
			mockFunc: func() {
				mock.ExpectQuery(
					`
						SELECT id, user_recipient_id, user_writer_id, comment, rating 
						FROM user_feedback WHERE id = \$1
						`,
				).
					WithArgs("id2").
					WillReturnError(sql.ErrNoRows)
			},
			want:    nil,
			wantErr: customErrors.ErrNotFound,
		},
		{
			name:    "db error",
			inputID: "id3",
			mockFunc: func() {
				mock.ExpectQuery(
					`
						SELECT id, user_recipient_id, user_writer_id, comment, rating 
						FROM user_feedback WHERE id = \$1
						`,
				).
					WithArgs("id3").
					WillReturnError(errors.New("db connection error"))
			},
			want:    nil,
			wantErr: customErrors.ErrDBInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFunc()
			result, err := repo.GetByID(context.Background(), tt.inputID)

			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.want, result)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserFeedbackRepository_GetByUserID(t *testing.T) {
	repo, mock, teardown := setupTestRepo(t)
	defer teardown()

	tests := []struct {
		name     string
		mockFunc func()
		wantErr  error
		wantLen  int
	}{
		{
			name: "success",
			mockFunc: func() {
				rows := sqlmock.NewRows([]string{"id", "user_recipient_id", "user_writer_id", "comment", "rating"}).
					AddRow("id1", "recipient-uuid", "writer-uuid", "Nice work", 4)
				mock.ExpectQuery("SELECT id, user_recipient_id, user_writer_id, comment, rating").
					WithArgs("recipient-uuid").
					WillReturnRows(rows)
			},
			wantErr: nil,
			wantLen: 1,
		},
		{
			name: "db error",
			mockFunc: func() {
				mock.ExpectQuery("SELECT id, user_recipient_id, user_writer_id, comment, rating").
					WithArgs("recipient-uuid").
					WillReturnError(errors.New("some error"))
			},
			wantErr: customErrors.ErrDBInternal,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFunc()
			result, err := repo.GetByUserID(context.Background(), "recipient-uuid")

			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.wantLen, len(result))
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserFeedbackRepository_Update(t *testing.T) {
	repo, mock, teardown := setupTestRepo(t)
	defer teardown()

	tests := []struct {
		name     string
		input    types.UpdateUserFeedback
		mockFunc func()
		wantErr  error
	}{
		{
			name: "success",
			input: types.UpdateUserFeedback{
				Comment: "Updated!",
				Rating:  3,
			},
			mockFunc: func() {
				mock.ExpectExec("UPDATE user_feedback SET (.+)").
					WillReturnResult(sqlmock.NewResult(0, 1))

				rows := sqlmock.NewRows([]string{"id", "user_recipient_id", "user_writer_id", "comment", "rating"}).
					AddRow("feedback-id", "recipient-uuid", "writer-uuid", "Updated!", 3)

				mock.ExpectQuery("SELECT id, user_recipient_id, user_writer_id, comment, rating FROM user_feedback WHERE id =").
					WithArgs("feedback-id").
					WillReturnRows(rows)
			},
			wantErr: nil,
		},
		{
			name: "not found",
			input: types.UpdateUserFeedback{
				Comment: "Not found",
			},
			mockFunc: func() {
				mock.ExpectExec("UPDATE user_feedback SET").
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: customErrors.ErrNotFoundUserFeedback,
		},
		{
			name: "db error",
			input: types.UpdateUserFeedback{
				Rating: 2,
			},
			mockFunc: func() {
				mock.ExpectExec("UPDATE user_feedback SET").
					WillReturnError(errors.New("db error"))
			},
			wantErr: customErrors.ErrDBInternal,
		},
		{
			name:     "nothing to update",
			input:    types.UpdateUserFeedback{},
			mockFunc: func() {},
			wantErr:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFunc()
			_, err := repo.Update(context.Background(), "feedback-id", tt.input)

			assert.Equal(t, tt.wantErr, err)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserFeedbackRepository_Delete(t *testing.T) {
	repo, mock, teardown := setupTestRepo(t)
	defer teardown()

	tests := []struct {
		name     string
		mockFunc func()
		wantErr  error
	}{
		{
			name: "success",
			mockFunc: func() {
				mock.ExpectExec("DELETE FROM user_feedback").
					WithArgs("feedback-id").
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: nil,
		},
		{
			name: "not found",
			mockFunc: func() {
				mock.ExpectExec("DELETE FROM user_feedback").
					WithArgs("feedback-id").
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: customErrors.ErrNotFoundUserFeedback,
		},
		{
			name: "db error",
			mockFunc: func() {
				mock.ExpectExec("DELETE FROM user_feedback").
					WithArgs("feedback-id").
					WillReturnError(errors.New("delete error"))
			},
			wantErr: customErrors.ErrDBInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFunc()
			err := repo.Delete(context.Background(), "feedback-id")

			assert.Equal(t, tt.wantErr, err)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
