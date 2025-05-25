package shopping_cart

import (
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"

	myErr "gafroshka-main/internal/types/errors"
)

func setup(t *testing.T) (*ShoppingCartRepository, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("ошибка при создании mock db: %s", err)
	}

	logger := zaptest.NewLogger(t).Sugar()
	repo := &ShoppingCartRepository{
		DB:     db,
		Logger: logger,
	}

	cleanup := func() {
		db.Close()
	}

	return repo, mock, cleanup
}

func TestAddAnnouncement(t *testing.T) {
	tests := []struct {
		name          string
		mockBehavior  func(mock sqlmock.Sqlmock)
		expectedError error
	}{
		{
			name: "успешное добавление",
			mockBehavior: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO shopping_cart(user_id, announcement_id) VALUES ($1, $2)")).
					WithArgs("user123", "ann456").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedError: nil,
		},
		{
			name: "ошибка БД",
			mockBehavior: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO shopping_cart(user_id, announcement_id) VALUES ($1, $2)")).
					WithArgs("user123", "ann456").
					WillReturnError(errors.New("db error"))
			},
			expectedError: myErr.ErrDBInternal, // сравним через errors.Is
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setup(t)
			defer cleanup()

			tt.mockBehavior(mock)

			err := repo.AddAnnouncement("user123", "ann456")
			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.expectedError))
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDeleteAnnouncement(t *testing.T) {
	tests := []struct {
		name          string
		mockBehavior  func(mock sqlmock.Sqlmock)
		expectedError error
	}{
		{
			name: "успешное удаление",
			mockBehavior: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta("DELETE FROM shopping_cart WHERE user_id = $1 AND announcement_id = $2")).
					WithArgs("user123", "ann456").
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedError: nil,
		},
		{
			name: "ошибка БД",
			mockBehavior: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta("DELETE FROM shopping_cart WHERE user_id = $1 AND announcement_id = $2")).
					WithArgs("user123", "ann456").
					WillReturnError(errors.New("delete failed"))
			},
			expectedError: myErr.ErrDBInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setup(t)
			defer cleanup()

			tt.mockBehavior(mock)

			err := repo.DeleteAnnouncement("user123", "ann456")
			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.expectedError))
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestGetByUserID(t *testing.T) {
	tests := []struct {
		name           string
		mockBehavior   func(mock sqlmock.Sqlmock)
		expectedResult []string
		expectedError  error
	}{
		{
			name: "успешный возврат",
			mockBehavior: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"announcement_id"}).
					AddRow("ann1").
					AddRow("ann2")
				mock.ExpectQuery(regexp.QuoteMeta("SELECT announcement_id FROM shopping_cart WHERE user_id = $1")).
					WithArgs("user123").
					WillReturnRows(rows)
			},
			expectedResult: []string{"ann1", "ann2"},
			expectedError:  nil,
		},
		{
			name: "ошибка БД",
			mockBehavior: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT announcement_id FROM shopping_cart WHERE user_id = $1")).
					WithArgs("user123").
					WillReturnError(errors.New("db failure"))
			},
			expectedResult: nil,
			expectedError:  myErr.ErrDBInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setup(t)
			defer cleanup()

			tt.mockBehavior(mock)

			res, err := repo.GetByUserID("user123")
			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.expectedError))
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, res)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
