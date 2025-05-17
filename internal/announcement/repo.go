package announcement

import (
	"database/sql"
	"strings"

	types "gafroshka-main/internal/types/announcement"
	"gafroshka-main/internal/types/errors"

	"go.uber.org/zap"
)

type AnnouncementDBRepository struct {
	DB     *sql.DB
	Logger *zap.SugaredLogger
}

func NewAnnouncementDBRepository(db *sql.DB, l *zap.SugaredLogger) *AnnouncementDBRepository {
	return &AnnouncementDBRepository{
		DB:     db,
		Logger: l,
	}
}

func (ar *AnnouncementDBRepository) Create(a types.CreateAnnouncement) (*Announcement, error) {
	var newAnn Announcement

	query := `
	INSERT INTO announcement (
		name, 
		description, 
		user_seller_id, 
		price, 
		category, 
		discount
	) VALUES ($1, $2, $3, $4, $5, $6)
	RETURNING id, name, description, user_seller_id, price, category, discount, is_active, rating, rating_count, created_at
	`

	err := ar.DB.QueryRow(
		query,
		a.Name,
		a.Description,
		a.UserSellerID,
		a.Price,
		a.Category,
		a.Discount,
	).Scan(
		&newAnn.ID,
		&newAnn.Name,
		&newAnn.Description,
		&newAnn.UserSellerID,
		&newAnn.Price,
		&newAnn.Category,
		&newAnn.Discount,
		&newAnn.IsActive,
		&newAnn.Rating,
		&newAnn.RatingCount,
		&newAnn.CreatedAt,
	)

	if err != nil {
		ar.Logger.Errorf("Error creating announcement: %v", err)
		return nil, errors.ErrDBInternal
	}

	return &newAnn, nil
}

func (ar *AnnouncementDBRepository) GetTopN(limit int) ([]Announcement, error) {
	query := `
	SELECT id, name, description, user_seller_id, price, category, discount, is_active, rating, rating_count, created_at 
	FROM announcement 
	WHERE is_active = TRUE 
	ORDER BY rating DESC 
	LIMIT $1
	`

	rows, err := ar.DB.Query(query, limit)
	if err != nil {
		ar.Logger.Errorf("Error getting top %d announcements: %v", limit, err)
		return nil, errors.ErrDBInternal
	}
	defer rows.Close()

	var announcements []Announcement
	for rows.Next() {
		var a Announcement
		err := rows.Scan(
			&a.ID,
			&a.Name,
			&a.Description,
			&a.UserSellerID,
			&a.Price,
			&a.Category,
			&a.Discount,
			&a.IsActive,
			&a.Rating,
			&a.RatingCount,
			&a.CreatedAt,
		)
		if err != nil {
			return nil, errors.ErrDBInternal
		}
		announcements = append(announcements, a)
	}

	return announcements, nil
}

func (ar *AnnouncementDBRepository) Search(query string) ([]Announcement, error) {
	query = strings.ToLower(query)
	sqlQuery := `
	SELECT id, name, description, user_seller_id, price, category, discount, is_active, rating, rating_count, created_at, 
		(LENGTH(name) - LENGTH(REPLACE(LOWER(name), $1, ''))) AS score
	FROM announcement 
	WHERE is_active = TRUE
	ORDER BY score DESC 
	LIMIT 10
	`

	rows, err := ar.DB.Query(sqlQuery, query)
	if err != nil {
		ar.Logger.Errorf("Error searching announcements: %v", err)
		return nil, errors.ErrDBInternal
	}
	defer rows.Close()

	var announcements []Announcement
	for rows.Next() {
		var a Announcement
		var score int
		err := rows.Scan(
			&a.ID,
			&a.Name,
			&a.Description,
			&a.UserSellerID,
			&a.Price,
			&a.Category,
			&a.Discount,
			&a.IsActive,
			&a.Rating,
			&a.RatingCount,
			&a.CreatedAt,
			&score,
		)
		if err != nil {
			return nil, errors.ErrDBInternal
		}
		announcements = append(announcements, a)
	}

	return announcements, nil
}

func (ar *AnnouncementDBRepository) GetByID(id string) (*Announcement, error) {
	var a Announcement

	query := `
	SELECT id, name, description, user_seller_id, price, category, discount, is_active, rating, rating_count, created_at 
	FROM announcement 
	WHERE id = $1
	`

	err := ar.DB.QueryRow(query, id).Scan(
		&a.ID,
		&a.Name,
		&a.Description,
		&a.UserSellerID,
		&a.Price,
		&a.Category,
		&a.Discount,
		&a.IsActive,
		&a.Rating,
		&a.RatingCount,
		&a.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.ErrNotFound
		}
		ar.Logger.Errorf("Error getting announcement by ID: %v", err)
		return nil, errors.ErrDBInternal
	}

	return &a, nil
}

func (ar *AnnouncementDBRepository) UpdateRating(id string, rate int) (*Announcement, error) {
	tx, err := ar.DB.Begin()
	if err != nil {
		return nil, errors.ErrDBInternal
	}
	defer tx.Rollback() // nolint:errcheck

	var currentRating float64
	var ratingCount int

	err = tx.QueryRow(
		"SELECT rating, rating_count FROM announcement WHERE id = $1 FOR UPDATE",
		id,
	).Scan(&currentRating, &ratingCount)

	if err != nil {
		return nil, errors.ErrDBInternal
	}

	newRating := float64(rate)
	if ratingCount > 0 {
		newRating = (currentRating + float64(rate)) / 2
	}

	_, err = tx.Exec(
		"UPDATE announcement SET rating = $1, rating_count = rating_count + 1 WHERE id = $2",
		newRating,
		id,
	)

	if err != nil {
		return nil, errors.ErrDBInternal
	}

	var updated Announcement
	err = tx.QueryRow(`
		SELECT id, name, description, user_seller_id, price, category, discount, is_active, rating, rating_count, created_at 
		FROM announcement 
		WHERE id = $1
	`, id).Scan(
		&updated.ID,
		&updated.Name,
		&updated.Description,
		&updated.UserSellerID,
		&updated.Price,
		&updated.Category,
		&updated.Discount,
		&updated.IsActive,
		&updated.Rating,
		&updated.RatingCount,
		&updated.CreatedAt,
	)
	if err != nil {
		return nil, errors.ErrDBInternal
	}

	if err = tx.Commit(); err != nil {
		return nil, errors.ErrDBInternal
	}

	return &updated, nil
}
