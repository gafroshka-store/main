package announcement

import (
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	types "gafroshka-main/internal/types/announcement"
	customErrors "gafroshka-main/internal/types/errors"

	"go.uber.org/zap/zaptest"
)

func TestAnnouncementDBRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := zaptest.NewLogger(t).Sugar()
	repo := NewAnnouncementDBRepository(db, logger)

	tests := []struct {
		name        string
		input       types.CreateAnnouncement
		mock        func()
		expected    Announcement
		expectError error
	}{
		{
			name: "successful creation",
			input: types.CreateAnnouncement{
				Name:         "Test Item",
				Description:  "Test Description",
				UserSellerID: "123",
				Price:        1000,
				Category:     1,
				Discount:     10,
			},
			mock: func() {
				mock.ExpectQuery(regexp.QuoteMeta(`
					INSERT INTO announcement (
						name, description, user_seller_id, 
						price, category, discount
					) VALUES ($1, $2, $3, $4, $5, $6)
					RETURNING *
				`)).
					WithArgs("Test Item", "Test Description", "123", int64(1000), 1, 10).
					WillReturnRows(sqlmock.NewRows([]string{
						"id", "name", "description", "user_seller_id",
						"price", "category", "discount", "is_active",
						"rating", "rating_count", "created_at",
					}).
						AddRow(
							"abc123", "Test Item", "Test Description", "123",
							1000, 1, 10, true,
							0.0, 0, time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), // Ensure this line ends with a comma if there are more parameters
						))
			},
			expected: Announcement{
				ID:           "abc123",
				Name:         "Test Item",
				Description:  "Test Description",
				UserSellerID: "123",
				Price:        1000,
				Category:     1,
				Discount:     10,
				IsActive:     true,
				Rating:       0.0,
				RatingCount:  0,
				CreatedAt:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			expectError: nil,
		},
		{
			name: "database error",
			input: types.CreateAnnouncement{
				Name: "Invalid Item",
			},
			mock: func() {
				mock.ExpectQuery(regexp.QuoteMeta(`
					INSERT INTO announcement (
						name, description, user_seller_id, 
						price, category, discount
					) VALUES ($1, $2, $3, $4, $5, $6)
					RETURNING *
				`)).
					WithArgs("Invalid Item", "", "", int64(0), 0, 0).
					WillReturnError(errors.New("database error"))
			},
			expected:    Announcement{},
			expectError: customErrors.ErrDBInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()

			result, err := repo.Create(tt.input)
			assert.Equal(t, tt.expected, result)
			assert.Equal(t, tt.expectError, err)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAnnouncementDBRepository_GetTopN(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := zaptest.NewLogger(t).Sugar()
	repo := NewAnnouncementDBRepository(db, logger)

	now := time.Now()

	tests := []struct {
		name        string
		limit       int
		mock        func()
		expected    []Announcement
		expectError error
	}{
		{
			name:  "get top 3",
			limit: 3,
			mock: func() {
				mock.ExpectQuery(regexp.QuoteMeta(`
					SELECT * 
					FROM announcement 
					WHERE is_active = TRUE 
					ORDER BY rating DESC 
					LIMIT $1
				`)).
					WithArgs(3).
					WillReturnRows(sqlmock.NewRows([]string{
						"id", "name", "description", "user_seller_id",
						"price", "category", "discount", "is_active",
						"rating", "rating_count", "created_at",
					}).
						AddRow(
							"1", "Item 1", "Desc 1", "123",
							100, 1, 0, true,
							4.5, 2, now,
						).
						AddRow(
							"2", "Item 2", "Desc 2", "456",
							200, 2, 5, true,
							4.0, 5, now,
						))
			},
			expected: []Announcement{
				{
					ID:           "1",
					Name:         "Item 1",
					Description:  "Desc 1",
					UserSellerID: "123",
					Price:        100,
					Category:     1,
					Discount:     0,
					IsActive:     true,
					Rating:       4.5,
					RatingCount:  2,
					CreatedAt:    now,
				},
				{
					ID:           "2",
					Name:         "Item 2",
					Description:  "Desc 2",
					UserSellerID: "456",
					Price:        200,
					Category:     2,
					Discount:     5,
					IsActive:     true,
					Rating:       4.0,
					RatingCount:  5,
					CreatedAt:    now,
				},
			},
			expectError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()

			result, err := repo.GetTopN(tt.limit)
			assert.Equal(t, tt.expected, result)
			assert.Equal(t, tt.expectError, err)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAnnouncementDBRepository_GetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := zaptest.NewLogger(t).Sugar()
	repo := NewAnnouncementDBRepository(db, logger)

	now := time.Now()

	tests := []struct {
		name        string
		id          string
		mock        func()
		expected    Announcement
		expectError error
	}{
		{
			name: "existing announcement",
			id:   "1",
			mock: func() {
				mock.ExpectQuery(regexp.QuoteMeta(`
					SELECT * 
					FROM announcement 
					WHERE id = $1
				`)).
					WithArgs("1").
					WillReturnRows(sqlmock.NewRows([]string{
						"id", "name", "description", "user_seller_id",
						"price", "category", "discount", "is_active",
						"rating", "rating_count", "created_at",
					}).
						AddRow(
							"1", "Test", "Desc", "123",
							100, 1, 0, true,
							4.5, 2, now,
						))
			},
			expected: Announcement{
				ID:           "1",
				Name:         "Test",
				Description:  "Desc",
				UserSellerID: "123",
				Price:        100,
				Category:     1,
				Discount:     0,
				IsActive:     true,
				Rating:       4.5,
				RatingCount:  2,
				CreatedAt:    now,
			},
			expectError: nil,
		},
		{
			name: "not found",
			id:   "999",
			mock: func() {
				mock.ExpectQuery(regexp.QuoteMeta(`
					SELECT * 
					FROM announcement 
					WHERE id = $1
				`)).
					WithArgs("999").
					WillReturnError(sql.ErrNoRows)
			},
			expected:    Announcement{},
			expectError: customErrors.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()

			result, err := repo.GetByID(tt.id)
			assert.Equal(t, tt.expected, result)
			assert.Equal(t, tt.expectError, err)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAnnouncementDBRepository_UpdateRating(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := zaptest.NewLogger(t).Sugar()
	repo := NewAnnouncementDBRepository(db, logger)

	tests := []struct {
		name        string
		id          string
		rate        int
		mock        func()
		expectError error
	}{
		{
			name: "first rating",
			id:   "1",
			rate: 5,
			mock: func() {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(`
					SELECT rating, rating_count 
					FROM announcement 
					WHERE id = $1 FOR UPDATE
				`)).
					WithArgs("1").
					WillReturnRows(sqlmock.NewRows([]string{"rating", "rating_count"}).AddRow(0.0, 0))

				mock.ExpectExec(regexp.QuoteMeta(`
					UPDATE announcement 
					SET rating = $1, rating_count = rating_count + 1 
					WHERE id = $2
				`)).
					WithArgs(5.0, "1").
					WillReturnResult(sqlmock.NewResult(0, 1))

				mock.ExpectCommit()
			},
			expectError: nil,
		},
		{
			name: "update existing rating",
			id:   "2",
			rate: 4,
			mock: func() {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(`
					SELECT rating, rating_count 
					FROM announcement 
					WHERE id = $1 FOR UPDATE
				`)).
					WithArgs("2").
					WillReturnRows(sqlmock.NewRows([]string{"rating", "rating_count"}).AddRow(4.5, 2))

				mock.ExpectExec(regexp.QuoteMeta(`
					UPDATE announcement 
					SET rating = $1, rating_count = rating_count + 1 
					WHERE id = $2
				`)).
					WithArgs((4.5+4.0)/2, "2").
					WillReturnResult(sqlmock.NewResult(0, 1))

				mock.ExpectCommit()
			},
			expectError: nil,
		},
		{
			name: "transaction error",
			id:   "3",
			rate: 5,
			mock: func() {
				mock.ExpectBegin().WillReturnError(errors.New("tx error"))
			},
			expectError: customErrors.ErrDBInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()

			err := repo.UpdateRating(tt.id, tt.rate)
			assert.Equal(t, tt.expectError, err)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
