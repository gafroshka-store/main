package feedback

import (
	"regexp"
	"testing"

	"gafroshka-main/internal/types/errors"
	"gafroshka-main/internal/types/feedback"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func setupTestRepo(t *testing.T) (*FeedbackDBRepository, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Не удалось создать mock db: %v", err)
	}

	logger := zap.NewNop().Sugar()
	repo := NewFeedbackDBRepository(db, logger)

	return repo, mock, func() { db.Close() }
}

func TestCreateFeedback(t *testing.T) {
	repo, mock, teardown := setupTestRepo(t)
	defer teardown()

	testFeedback := feedback.Feedback{
		AnnouncementID: "1",
		UserWriterID:   "2",
		Comment:        "Отличное объявление!",
		Rating:         5,
	}

	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO announcement_feedback (id, announcement_recipient_id, user_writer_id, comment, rating)
		VALUES ($1, $2, $3, $4, $5)
	`)).
		WithArgs(sqlmock.AnyArg(), // ID генерируется автоматически
			testFeedback.AnnouncementID,
			testFeedback.UserWriterID,
			testFeedback.Comment,
			testFeedback.Rating).
		WillReturnResult(sqlmock.NewResult(1, 1))

	created, err := repo.Create(testFeedback)

	assert.NoError(t, err)
	assert.NotNil(t, created)
	assert.NotEmpty(t, created.ID)

	assert.Equal(t, testFeedback.AnnouncementID, created.AnnouncementID)
	assert.Equal(t, testFeedback.UserWriterID, created.UserWriterID)
	assert.Equal(t, testFeedback.Comment, created.Comment)
	assert.Equal(t, testFeedback.Rating, created.Rating)
}

func TestDeleteFeedback(t *testing.T) {
	repo, mock, teardown := setupTestRepo(t)
	defer teardown()

	t.Run("успешное удаление", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM announcement_feedback WHERE id = $1`)).
			WithArgs("1").
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Delete("1")
		assert.NoError(t, err)
	})

	t.Run("отзыв не найден", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM announcement_feedback WHERE id = $1`)).
			WithArgs("2").
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.Delete("2")
		assert.ErrorIs(t, err, errors.ErrNotFoundFeedback)
	})

	t.Run("ошибка базы данных", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM announcement_feedback WHERE id = $1`)).
			WithArgs("3").
			WillReturnError(assert.AnError)

		err := repo.Delete("3")
		assert.ErrorIs(t, err, errors.ErrDBInternal)
	})
}

func TestGetByAnnouncementID(t *testing.T) {
	repo, mock, teardown := setupTestRepo(t)
	defer teardown()

	t.Run("найдены отзывы", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "announcement_recipient_id", "user_writer_id", "comment", "rating"}).
			AddRow("1", "10", "100", "Отзыв 1", 5).
			AddRow("2", "10", "101", "Отзыв 2", 4)

		mock.ExpectQuery(regexp.QuoteMeta(`
			SELECT id, announcement_recipient_id, user_writer_id, comment, rating
			FROM announcement_feedback
			WHERE announcement_recipient_id = $1
		`)).
			WithArgs("10").
			WillReturnRows(rows)

		result, err := repo.GetByAnnouncementID("10")

		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "1", result[0].ID)
		assert.Equal(t, "2", result[1].ID)
	})

	t.Run("ошибка запроса", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta(`
			SELECT id, announcement_recipient_id, user_writer_id, comment, rating
			FROM announcement_feedback
			WHERE announcement_recipient_id = $1
		`)).
			WithArgs("11").
			WillReturnError(assert.AnError)

		_, err := repo.GetByAnnouncementID("11")
		assert.ErrorIs(t, err, errors.ErrDBInternal)
	})
}
