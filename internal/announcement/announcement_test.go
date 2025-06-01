package announcement

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	elastic "gafroshka-main/internal/elastic_search"
	"github.com/elastic/go-elasticsearch/v8"
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

	elasticClient, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{
			"http://elasticsearch:9200",
		},
	})
	if err != nil {
		logger.Errorf("failed to create elastic client: %v", err)
	}

	_, err = elasticClient.Ping()
	if err != nil {
		logger.Warnf("failed to ping Elasticsearch: %v", err)
	}

	elasicService := elastic.NewService(elasticClient, logger, "announcement")

	repo := NewAnnouncementDBRepository(db, logger, elasicService)

	tests := []struct {
		name        string
		input       types.CreateAnnouncement
		mock        func()
		expected    *Announcement
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
					RETURNING id, 
					name, 
					description, 
					user_seller_id, 
					price, category, 
					discount, 
					is_active, 
					rating, rating_count, created_at
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
							0.0, 0, time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
						))
			},
			expected: &Announcement{
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
					RETURNING id, 
					name, 
					description, 
					user_seller_id, 
					price, category, 
					discount, 
					is_active, 
					rating, rating_count, created_at				`)).
					WithArgs("Invalid Item", "", "", int64(0), 0, 0).
					WillReturnError(errors.New("database error"))
			},
			expected:    nil,
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

	elasticClient, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{
			"http://elasticsearch:9200",
		},
	})
	if err != nil {
		logger.Errorf("failed to create elastic client: %v", err)
	}

	_, err = elasticClient.Ping()
	if err != nil {
		logger.Warnf("failed to ping Elasticsearch: %v", err)
	}

	elasicService := elastic.NewService(elasticClient, logger, "announcement")

	repo := NewAnnouncementDBRepository(db, logger, elasicService)

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
					SELECT id, 
					name, 
					description, 
					user_seller_id, 
					price, 
					
					category, 
					discount, 
					is_active, rating, rating_count, created_at 
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

	elasticClient, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{
			"http://elasticsearch:9200",
		},
	})
	if err != nil {
		logger.Errorf("failed to create elastic client: %v", err)
	}

	_, err = elasticClient.Ping()
	if err != nil {
		logger.Warnf("failed to ping Elasticsearch: %v", err)
	}

	elasicService := elastic.NewService(elasticClient, logger, "announcement")

	repo := NewAnnouncementDBRepository(db, logger, elasicService)

	now := time.Now()

	tests := []struct {
		name        string
		id          string
		mock        func()
		expected    *Announcement
		expectError error
	}{
		{
			name: "existing announcement",
			id:   "1",
			mock: func() {
				mock.ExpectQuery(regexp.QuoteMeta(`
					SELECT id, 
					name, 
					description, 
					user_seller_id, 
					price, 
					category, discount, 
					is_active, 
					rating, rating_count, created_at 
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
			expected: &Announcement{
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
					SELECT id, 
					name, 
					description, 
					user_seller_id, 
					price, 
					category, discount, 
					is_active, 
					rating, rating_count, created_at 
					FROM announcement 
					WHERE id = $1
				`)).
					WithArgs("999").
					WillReturnError(sql.ErrNoRows)
			},
			expected:    nil,
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

func TestAnnouncementDBRepository_GetInfoForShoppingCart(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := zaptest.NewLogger(t).Sugar()

	elasticClient, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{
			"http://elasticsearch:9200",
		},
	})
	if err != nil {
		logger.Errorf("failed to create elastic client: %v", err)
	}

	_, err = elasticClient.Ping()
	if err != nil {
		logger.Warnf("failed to ping Elasticsearch: %v", err)
	}

	elasicService := elastic.NewService(elasticClient, logger, "announcement")

	repo := NewAnnouncementDBRepository(db, logger, elasicService)

	tests := []struct {
		name        string
		ids         []string
		mock        func()
		expected    []types.InfoForSC
		expectError error
	}{
		{
			name: "success",
			ids:  []string{"1", "2"},
			mock: func() {
				values := [][]driver.Value{
					{"1", "test1", 10, 0, true, 5.0},
					{"2", "test2", 100, 10, true, 5.0},
				}

				mock.ExpectQuery(regexp.QuoteMeta(
					"SELECT id, name, price, discount, is_active, rating FROM announcement WHERE id IN ($1,$2)",
				)).
					WithArgs("1", "2").
					WillReturnRows(sqlmock.NewRows([]string{
						"id", "name", "price", "discount", "is_active", "rating",
					}).AddRows(values...))
			},
			expected: []types.InfoForSC{
				{ID: "1", Name: "test1", Price: 10, Discount: 0, IsActive: true, Rating: 5.0},
				{ID: "2", Name: "test2", Price: 100, Discount: 10, IsActive: true, Rating: 5.0},
			},
			expectError: nil,
		},
		{
			name: "empty ids returns empty slice without query",
			ids:  []string{},
			mock: func() {
				// Не ожидаем никаких запросов
			},
			expected:    []types.InfoForSC{},
			expectError: nil,
		},
		{
			name: "db query error",
			ids:  []string{"1"},
			mock: func() {
				mock.ExpectQuery(regexp.QuoteMeta(
					"SELECT id, name, price, discount, is_active, rating FROM announcement WHERE id IN ($1)",
				)).
					WithArgs("1").
					WillReturnError(fmt.Errorf("some db error"))
			},
			expected:    nil,
			expectError: customErrors.ErrDBInternal,
		},
		{
			name: "scan error",
			ids:  []string{"1"},
			mock: func() {
				rows := sqlmock.NewRows([]string{"id", "name", "price", "discount", "is_active", "rating"}).
					AddRow("1", "test1", "WRONG_TYPE_INSTEAD_OF_INT", 0, true, 5.0)
				mock.ExpectQuery(regexp.QuoteMeta(
					"SELECT id, name, price, discount, is_active, rating FROM announcement WHERE id IN ($1)",
				)).
					WithArgs("1").
					WillReturnRows(rows)
			},
			expected:    nil,
			expectError: customErrors.ErrDBInternal,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()

			result, err := repo.GetInfoForShoppingCart(tt.ids)
			assert.Equal(t, tt.expected, result)
			assert.Equal(t, tt.expectError, err)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
