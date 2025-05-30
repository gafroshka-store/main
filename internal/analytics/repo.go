package analytics

import (
	"context"
	"database/sql"
	"go.uber.org/zap"
)

type Repository struct {
	db     *sql.DB
	logger *zap.SugaredLogger
}

func NewRepository(db *sql.DB, logger *zap.SugaredLogger) *Repository {
	return &Repository{
		db:     db,
		logger: logger,
	}
}

func (r *Repository) UpdatePreferences(ctx context.Context, userID string, weights map[int]int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for category, weight := range weights {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO user_preferences (user_id, category, weight)
			VALUES ($1, $2, $3)
			ON CONFLICT (user_id, category)
			DO UPDATE SET weight = user_preferences.weight + EXCLUDED.weight
		`, userID, category, weight)

		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *Repository) GetTopCategories(ctx context.Context, userID string, limit int) ([]int, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT category
		FROM user_preferences
		WHERE user_id = $1
		ORDER BY weight DESC
		LIMIT $2
	`, userID, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []int
	for rows.Next() {
		var category int
		if err := rows.Scan(&category); err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}

	return categories, nil
}
