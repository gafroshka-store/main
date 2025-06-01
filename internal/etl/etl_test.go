package etl_test

import (
	"context"
	"errors"
	"gafroshka-main/internal/announcement"
	"gafroshka-main/internal/types/elastic"
	"regexp"
	"testing"
	"time"

	"gafroshka-main/internal/etl"
	"github.com/DATA-DOG/go-sqlmock"
	"go.uber.org/zap"
)

func TestPostgresExtractor_ExtractNew(t *testing.T) {
	logger := zap.NewNop().Sugar()

	tests := []struct {
		name          string
		mockQuery     func(mock sqlmock.Sqlmock)
		expectedError bool
		expectedCount int
	}{
		{
			name: "success with two rows",
			mockQuery: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "name", "description", "category", "user_seller_id", "created_at"}).
					AddRow("id1", "name1", "desc1", 1, "seller1", time.Now()).
					AddRow("id2", "name2", "desc2", 2, "seller2", time.Now())
				mock.ExpectQuery(regexp.QuoteMeta(`
					SELECT id, name, description, category, user_seller_id, created_at
					FROM announcement
					WHERE searching = FALSE AND is_active = TRUE
				`)).WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name: "query error",
			mockQuery: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`
					SELECT id, name, description, category, user_seller_id, created_at
					FROM announcement
					WHERE searching = FALSE AND is_active = TRUE
				`)).WillReturnError(errors.New("query failed"))
			},
			expectedError: true,
		},
		{
			name: "rows iteration error",
			mockQuery: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "name", "description", "category", "user_seller_id", "created_at"}).
					AddRow("id1", "name1", "desc1", 1, "seller1", time.Now())
				mock.ExpectQuery(regexp.QuoteMeta(`
					SELECT id, name, description, category, user_seller_id, created_at
					FROM announcement
					WHERE searching = FALSE AND is_active = TRUE
				`)).WillReturnRows(rows).RowsWillBeClosed()
			},
			expectedError: false,
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to open sqlmock: %v", err)
			}
			defer db.Close()

			tt.mockQuery(mock)

			extractor := etl.NewPostgresExtractor(db, logger)
			ctx := context.Background()

			results, err := extractor.ExtractNew(ctx)

			if tt.expectedError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if len(results) != tt.expectedCount {
				t.Errorf("expected %d results, got %d", tt.expectedCount, len(results))
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

func TestTransformer_Transform(t *testing.T) {
	logger := zap.NewNop().Sugar()

	tests := []struct {
		name   string
		input  []announcement.Announcement
		expect []elastic.ElasticDoc
	}{
		{
			name:   "empty input",
			input:  []announcement.Announcement{},
			expect: []elastic.ElasticDoc{},
		},
		{
			name: "single announcement",
			input: []announcement.Announcement{
				{
					ID:          "1",
					Name:        "Title",
					Description: "Desc",
					Category:    1,
				},
			},
			expect: []elastic.ElasticDoc{
				{
					ID:          "1",
					Name:        "Title",
					Description: "Desc",
					Category:    1,
				},
			},
		},
		{
			name: "multiple announcements",
			input: []announcement.Announcement{
				{ID: "1", Name: "A1", Description: "D1", Category: 1},
				{ID: "2", Name: "A2", Description: "D2", Category: 2},
			},
			expect: []elastic.ElasticDoc{
				{ID: "1", Name: "A1", Description: "D1", Category: 1},
				{ID: "2", Name: "A2", Description: "D2", Category: 2},
			},
		},
	}

	transformer := etl.NewTransformer(logger)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := transformer.Transform(tt.input)
			if len(got) != len(tt.expect) {
				t.Fatalf("expected %d results, got %d", len(tt.expect), len(got))
			}

			for i := range got {
				if got[i] != tt.expect[i] {
					t.Errorf("expected %v, got %v", tt.expect[i], got[i])
				}
			}
		})
	}
}
