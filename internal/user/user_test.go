package user

import (
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	customErrors "gafroshka-main/internal/types/errors"
	types "gafroshka-main/internal/types/user"

	"go.uber.org/zap/zaptest"
)

func TestUserDBRepository_Info(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := zaptest.NewLogger(t).Sugar()
	repository := NewUserDBRepository(db, logger, nil)

	tests := []struct {
		name        string
		userID      string
		mockQuery   func()
		expected    User
		expectError error
	}{
		{
			name:   "valid user",
			userID: "123",
			mockQuery: func() {
				mock.ExpectQuery(`
				SELECT user_id, 
					name,
					surname,
					registration_data,
					email,
					phone_number,
					balance,
					deals_count 
		   		FROM users WHERE user_id = \$1
				`).
					WithArgs("123").
					WillReturnRows(sqlmock.NewRows([]string{
						"user_id", "name", "surname", "registration_data",
						"email", "phone_number", "balance", "deals_count",
					}).AddRow("123", "John", "Doe", time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC), "john@example.com", "1234567890", 100.0, 5))
			},
			expected: User{
				ID:               "123",
				Name:             "John",
				Surname:          "Doe",
				RegistrationDate: time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC),
				Email:            "john@example.com",
				PhoneNumber:      "1234567890",
				Balance:          100.0,
				DealsCount:       5,
			},
			expectError: nil,
		},
		{
			name:   "user not found",
			userID: "999",
			mockQuery: func() {
				mock.ExpectQuery(`
				SELECT user_id, 
					name,
					surname,
					registration_data,
					email,
					phone_number,
					balance,
					deals_count 
		   		FROM users WHERE user_id = \$1`).
					WithArgs("999").
					WillReturnError(sql.ErrNoRows)
			},
			expected:    User{},
			expectError: customErrors.ErrDBInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockQuery()

			user, err := repository.Info(tt.userID)
			assert.Equal(t, tt.expected, user)
			assert.Equal(t, tt.expectError, err)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserDBRepository_ChangeProfile(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := zaptest.NewLogger(t).Sugar()
	repository := NewUserDBRepository(db, logger, nil)

	tests := []struct {
		name           string
		userID         string
		update         types.ChangeUser
		mockQuery      func()
		expectedResult User
		expectError    error
	}{
		{
			name:   "update name and email",
			userID: "123",
			update: types.ChangeUser{
				Name:  "Alice",
				Email: "alice@example.com",
			},
			mockQuery: func() {
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE users SET name = $1, email = $2 WHERE user_id = $3`)).
					WithArgs("Alice", "alice@example.com", "123").
					WillReturnResult(sqlmock.NewResult(0, 1))

				mock.ExpectQuery(`SELECT user_id,.*FROM users WHERE user_id = \$1`).
					WithArgs("123").
					WillReturnRows(sqlmock.NewRows([]string{
						"user_id", "name", "surname", "registration_data",
						"email", "phone_number", "balance", "deals_count",
					}).AddRow("123", "Alice", "Doe", time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC), "alice@example.com", "1234567890", 100.0, 5))
			},
			expectedResult: User{
				ID:               "123",
				Name:             "Alice",
				Surname:          "Doe",
				RegistrationDate: time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC),
				Email:            "alice@example.com",
				PhoneNumber:      "1234567890",
				Balance:          100.0,
				DealsCount:       5,
			},
			expectError: nil,
		},
		{
			name:   "no update fields",
			userID: "123",
			update: types.ChangeUser{},
			mockQuery: func() {
				mock.ExpectQuery(`SELECT user_id,.*FROM users WHERE user_id = \$1`).
					WithArgs("123").
					WillReturnRows(sqlmock.NewRows([]string{
						"user_id", "name", "surname", "registration_data",
						"email", "phone_number", "balance", "deals_count",
					}).AddRow("123", "John", "Doe", time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC), "john@example.com", "1234567890", 100.0, 5))
			},
			expectedResult: User{
				ID:               "123",
				Name:             "John",
				Surname:          "Doe",
				RegistrationDate: time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC),
				Email:            "john@example.com",
				PhoneNumber:      "1234567890",
				Balance:          100.0,
				DealsCount:       5,
			},
			expectError: nil,
		},
		{
			name:   "db error on update",
			userID: "123",
			update: types.ChangeUser{
				Name: "Alice",
			},
			mockQuery: func() {
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE users SET name = $1 WHERE user_id = $2`)).
					WithArgs("Alice", "123").
					WillReturnError(errors.New("db failure"))
			},
			expectedResult: User{},
			expectError:    customErrors.ErrDBInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockQuery()

			result, err := repository.ChangeProfile(tt.userID, tt.update)
			assert.Equal(t, tt.expectedResult, result)
			assert.Equal(t, tt.expectError, err)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
