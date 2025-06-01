package user

import (
	"database/sql"
	"errors"
	"golang.org/x/crypto/bcrypt"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	myErr "gafroshka-main/internal/types/errors"
	types "gafroshka-main/internal/types/user"

	"go.uber.org/zap/zaptest"
)

func TestUserDBRepository_CreateUser(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &UserDBRepository{DB: db, Logger: nil}

	u := types.CreateUser{
		Name:        "John",
		Surname:     "Doe",
		DateOfBirth: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		Sex:         true,
		Email:       "john@example.com",
		PhoneNumber: "1234567890",
		Password:    "securepass123",
	}

	t.Run("successfully_create_user", func(t *testing.T) {
		// 1. CheckUser — не найден
		mock.ExpectQuery(`SELECT .* FROM users WHERE email = \$1`).
			WithArgs(u.Email).
			WillReturnError(sql.ErrNoRows)

		// 2. INSERT INTO users
		mock.ExpectExec(`INSERT INTO users`).
			WithArgs(sqlmock.AnyArg(), u.Name, u.Surname, u.DateOfBirth, u.Sex,
				sqlmock.AnyArg(), u.Email, u.PhoneNumber, sqlmock.AnyArg(),
				0, 0, 0, 0).
			WillReturnResult(sqlmock.NewResult(1, 1))

		created, err := repo.CreateUser(u)
		require.NoError(t, err)
		require.NotNil(t, created)
		require.Equal(t, u.Name, created.Name)
		require.Equal(t, u.Email, created.Email)
	})

	t.Run("user_already_exists", func(t *testing.T) {
		hash, _ := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost) // nolint:errcheck

		rows := sqlmock.NewRows([]string{
			"id", "name", "surname", "day_of_birth", "sex", "registration_date",
			"email", "phone_number", "password_hash", "balance", "deals_count", "rating", "rating_count",
		}).AddRow("some-id", u.Name, u.Surname, u.DateOfBirth, u.Sex, time.Now(),
			u.Email, u.PhoneNumber, string(hash), 0, 0, 0, 0)

		mock.ExpectQuery(`SELECT .* FROM users WHERE email = \$1`).
			WithArgs(u.Email).
			WillReturnRows(rows)

		_, err := repo.CreateUser(u)
		require.ErrorIs(t, err, myErr.ErrAlreadyExists)
	})
}

func TestUserDBRepository_CheckUser(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := zaptest.NewLogger(t).Sugar()
	repository := NewUserDBRepository(db, logger)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("correct_password"), bcrypt.DefaultCost) // nolint:errcheck

	tests := []struct {
		name        string
		email       string
		password    string
		mockQuery   func()
		expectUser  bool
		expectError error
	}{
		{
			name:     "valid credentials",
			email:    "valid@example.com",
			password: "correct_password",
			mockQuery: func() {
				mock.ExpectQuery(`SELECT id,.*FROM users WHERE email = \$1`).
					WithArgs("valid@example.com").
					WillReturnRows(sqlmock.NewRows([]string{
						"id", "name", "surname", "day_of_birth", "sex",
						"registration_date", "email", "phone_number", "password_hash",
						"balance", "deals_count", "rating", "rating_count",
					}).AddRow(
						"123", "John", "Doe", time.Time{}, SexManT,
						time.Now(), "valid@example.com", "1234567890", string(hashedPassword),
						100.0, 5, 4.5, 10,
					))
			},
			expectUser:  true,
			expectError: nil,
		},
		{
			name:     "user not found",
			email:    "notfound@example.com",
			password: "whatever",
			mockQuery: func() {
				mock.ExpectQuery(`SELECT id,.*FROM users WHERE email = \$1`).
					WithArgs("notfound@example.com").
					WillReturnError(sql.ErrNoRows)
			},
			expectUser:  false,
			expectError: myErr.ErrNotFound,
		},
		{
			name:     "wrong password",
			email:    "valid@example.com",
			password: "wrong_password",
			mockQuery: func() {
				mock.ExpectQuery(`SELECT id,.*FROM users WHERE email = \$1`).
					WithArgs("valid@example.com").
					WillReturnRows(sqlmock.NewRows([]string{
						"id", "name", "surname", "day_of_birth", "sex",
						"registration_date", "email", "phone_number", "password_hash",
						"balance", "deals_count", "rating", "rating_count",
					}).AddRow(
						"123", "John", "Doe", time.Time{}, SexManT,
						time.Now(), "valid@example.com", "1234567890", string(hashedPassword),
						100.0, 5, 4.5, 10,
					))
			},
			expectUser:  false,
			expectError: myErr.ErrBadPassword,
		},
		{
			name:     "db error",
			email:    "error@example.com",
			password: "irrelevant",
			mockQuery: func() {
				mock.ExpectQuery(`SELECT id,.*FROM users WHERE email = \$1`).
					WithArgs("error@example.com").
					WillReturnError(errors.New("db failure"))
			},
			expectUser:  false,
			expectError: myErr.ErrDBInternal,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.mockQuery()
			user, err := repository.CheckUser(tt.email, tt.password)

			if tt.expectUser {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.email, user.Email)
			} else {
				assert.Nil(t, user)
				assert.ErrorContains(t, err, tt.expectError.Error())
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserDBRepository_Info(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := zaptest.NewLogger(t).Sugar()
	repository := NewUserDBRepository(db, logger)

	tests := []struct {
		name        string
		userID      string
		mockQuery   func()
		expected    *User
		expectError error
	}{
		{
			name:   "valid user",
			userID: "123",
			mockQuery: func() {
				mock.ExpectQuery(`SELECT 
				id, name, 
				surname, day_of_birth, 
				sex, registration_date, 
				email, phone_number, 
				balance, deals_count, 
				rating, rating_count 
				FROM users 
				WHERE id = \$1`).
					WithArgs("123").
					WillReturnRows(sqlmock.NewRows([]string{
						"id", "name", "surname", "day_of_birth", "sex", "registration_date",
						"email", "phone_number", "balance", "deals_count", "rating", "rating_count",
					}).AddRow(
						"123", "John", "Doe", time.Time{},
						SexManT, time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC),
						"john@example.com", "1234567890", 100.0, 5, 4.0, 2,
					))

			},
			expected: &User{
				ID:               "123",
				Name:             "John",
				Surname:          "Doe",
				DayOfBirth:       time.Time{},
				Sex:              SexManT,
				RegistrationDate: time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC),
				Email:            "john@example.com",
				PhoneNumber:      "1234567890",
				Balance:          100.0,
				DealsCount:       5,
				Rating:           4.0,
				RatingCount:      2,
			},
			expectError: nil,
		},
		{
			name:   "user not found",
			userID: "999",
			mockQuery: func() {
				mock.ExpectQuery(`
				SELECT id, 
					name,
					surname,
					day_of_birth,
					sex,
					registration_date,
					email,
					phone_number,
					balance,
					deals_count,
					rating,
					rating_count
		   		FROM users WHERE id = \$1`).
					WithArgs("999").
					WillReturnError(sql.ErrNoRows)
			},
			expected:    nil,
			expectError: myErr.ErrNotFound,
		},
	}

	for _, tt := range tests {
		tt := tt
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
	t.Parallel()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := zaptest.NewLogger(t).Sugar()
	repository := NewUserDBRepository(db, logger)

	tests := []struct {
		name           string
		userID         string
		update         types.ChangeUser
		mockQuery      func()
		expectedResult *User
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
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE users SET name = $1, email = $2 WHERE id = $3`)).
					WithArgs("Alice", "alice@example.com", "123").
					WillReturnResult(sqlmock.NewResult(0, 1))

				mock.ExpectQuery(`SELECT id, 
					name, 
					surname, 
					day_of_birth, 
					sex, 
					registration_date, 
					email, 
					phone_number, 
					balance, 
					deals_count, 
					rating, 
					rating_count 
				FROM users 
				WHERE id = \$1`).
					WithArgs("123").
					WillReturnRows(sqlmock.NewRows([]string{
						"id", "name", "surname", "day_of_birth", "sex", "registration_date",
						"email", "phone_number", "balance", "deals_count", "rating", "rating_count",
					}).AddRow(
						"123", "Alice", "Doe", time.Time{},
						SexWomenT, time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC),
						"alice@example.com", "1234567890", 100.0, 5, 0.0, 0,
					))

			},
			expectedResult: &User{
				ID:               "123",
				Name:             "Alice",
				Surname:          "Doe",
				DayOfBirth:       time.Time{},
				Sex:              SexWomenT,
				RegistrationDate: time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC),
				Email:            "alice@example.com",
				PhoneNumber:      "1234567890",
				Balance:          100.0,
				DealsCount:       5,
				Rating:           0,
				RatingCount:      0,
			},
			expectError: nil,
		},
		{
			name:   "no update fields",
			userID: "123",
			update: types.ChangeUser{},
			mockQuery: func() {
				mock.ExpectQuery(`SELECT id,.*FROM users WHERE id = \$1`).
					WithArgs("123").
					WillReturnRows(sqlmock.NewRows([]string{
						"id", "name", "surname", "day_of_birth", "sex", "registration_date",
						"email", "phone_number", "balance", "deals_count", "rating", "rating_count",
					}).AddRow(
						"123", "John", "Doe", time.Time{}, SexManT,
						time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC),
						"john@example.com", "1234567890", 100.0, 5, 0, 0,
					))
			},
			expectedResult: &User{
				ID:               "123",
				Name:             "John",
				Surname:          "Doe",
				DayOfBirth:       time.Time{},
				Sex:              SexManT,
				RegistrationDate: time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC),
				Email:            "john@example.com",
				PhoneNumber:      "1234567890",
				Balance:          100.0,
				DealsCount:       5,
				Rating:           0,
				RatingCount:      0,
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
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE users SET name = $1 WHERE id = $2`)).
					WithArgs("Alice", "123").
					WillReturnError(errors.New("db failure"))
			},
			expectedResult: nil,
			expectError:    myErr.ErrDBInternal,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.mockQuery()

			result, err := repository.ChangeProfile(tt.userID, tt.update)
			assert.Equal(t, tt.expectedResult, result)
			assert.Equal(t, tt.expectError, err)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserDBRepository_GetBalanceByUserID(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	logger := zaptest.NewLogger(t).Sugar()
	repository := NewUserDBRepository(db, logger)

	tests := []struct {
		name        string
		userID      string
		mockQuery   func()
		wantBalance int64
		wantErr     error
	}{
		{
			name:   "success",
			userID: "123",
			mockQuery: func() {
				mock.ExpectQuery(`SELECT balance FROM users WHERE id = \$1`).
					WithArgs("123").WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(52))
			},
			wantBalance: 52,
			wantErr:     nil,
		},

		{
			name:   "not found",
			userID: "124",
			mockQuery: func() {
				mock.ExpectQuery(`SELECT balance FROM users WHERE id = \$1`).
					WithArgs("124").WillReturnError(sql.ErrNoRows)
			},
			wantBalance: 0,
			wantErr:     myErr.ErrNotFound,
		},

		{
			name:   "db error on query",
			userID: "125",
			mockQuery: func() {
				mock.ExpectQuery(`SELECT balance FROM users WHERE id = \$1`).
					WithArgs("125").WillReturnError(myErr.ErrDBInternal)
			},
			wantBalance: 0,
			wantErr:     myErr.ErrDBInternal,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.mockQuery()
			gotBalance, err := repository.GetBalanceByUserID(tt.userID)
			assert.Equal(t, tt.wantBalance, gotBalance)
			assert.Equal(t, tt.wantErr, err)
			assert.NoError(t, mock.ExpectationsWereMet())

		})
	}
}

func TestUserDBRepository_TopUpBalance(t *testing.T) {
	t.Parallel()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error when opening stub db: %s", err)
	}
	defer db.Close()

	logger := zaptest.NewLogger(t).Sugar()
	repository := NewUserDBRepository(db, logger)

	tests := []struct {
		name        string
		userID      string
		amount      int64
		mockQuery   func()
		wantBalance int64
		wantErr     error
	}{
		{
			name:   "success",
			userID: "123",
			amount: 100,
			mockQuery: func() {
				mock.ExpectQuery(`UPDATE users SET balance = balance \+ \$1 WHERE id = \$2 RETURNING balance`).
					WithArgs(int64(100), "123").
					WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(150))
			},
			wantBalance: 150,
			wantErr:     nil,
		},
		{
			name:   "invalid amount",
			userID: "123",
			amount: 0,
			mockQuery: func() {
				// No DB call expected
			},
			wantBalance: 0,
			wantErr:     myErr.ErrInvalidAmount,
		},
		{
			name:   "user not found",
			userID: "124",
			amount: 50,
			mockQuery: func() {
				mock.ExpectQuery(`UPDATE users SET balance = balance \+ \$1 WHERE id = \$2 RETURNING balance`).
					WithArgs(int64(50), "124").
					WillReturnError(sql.ErrNoRows)
			},
			wantBalance: 0,
			wantErr:     myErr.ErrNotFound,
		},
		{
			name:   "db error",
			userID: "125",
			amount: 30,
			mockQuery: func() {
				mock.ExpectQuery(`UPDATE users SET balance = balance \+ \$1 WHERE id = \$2 RETURNING balance`).
					WithArgs(int64(30), "125").
					WillReturnError(errors.New("db failure"))
			},
			wantBalance: 0,
			wantErr:     myErr.ErrDBInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockQuery != nil {
				tt.mockQuery()
			}
			gotBalance, err := repository.TopUpBalance(tt.userID, tt.amount)
			assert.Equal(t, tt.wantBalance, gotBalance)
			assert.Equal(t, tt.wantErr, err)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
