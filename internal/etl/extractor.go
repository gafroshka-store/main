package etl

import (
	"database/sql"
	"gafroshka-main/internal/announcement"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"time"
)

type PostgresExtractor struct {
	DB     *sql.DB
	Logger *zap.SugaredLogger
}

func NewPostgresExtractor(db *sql.DB, logger *zap.SugaredLogger) *PostgresExtractor {
	return &PostgresExtractor{
		DB:     db,
		Logger: logger,
	}
}

func (e *PostgresExtractor) ExtractNew(ctx context.Context, since time.Time) ([]announcement.Announcement, error) {
	query :=
		`
		SELECT id, name, description, category, user_seller_id, created_at
		FROM announcements
		WHERE created_at >= $1 AND is_active = true
		`

	rows, err := e.DB.QueryContext(ctx, query, since)
	if err != nil {
		e.Logger.Error("Failed to executing query", zap.Error(err))

		return nil, err
	}
	defer rows.Close()

	var result []announcement.Announcement

	for rows.Next() {
		var a announcement.Announcement
		err := rows.Scan(&a.ID, &a.Name, &a.Description, &a.Category, &a.UserSellerID, &a.CreatedAt)
		if err != nil {
			e.Logger.Error("Failed to scan rows", zap.Error(err))

			return nil, err
		}
		result = append(result, a)
	}

	if err := rows.Err(); err != nil {
		e.Logger.Error("Error during rows iteration", zap.Error(err))
		return nil, err
	}

	return result, nil
}
