package announcement

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	elastic "gafroshka-main/internal/elastic_search"

	"github.com/lib/pq"

	types "gafroshka-main/internal/types/announcement"
	"gafroshka-main/internal/types/errors"

	"go.uber.org/zap"
)

type AnnouncementDBRepository struct {
	DB             *sql.DB
	Logger         *zap.SugaredLogger
	ElasticService *elastic.ElasticService
}

func NewAnnouncementDBRepository(db *sql.DB, l *zap.SugaredLogger, es *elastic.ElasticService) *AnnouncementDBRepository {
	return &AnnouncementDBRepository{
		DB:             db,
		Logger:         l,
		ElasticService: es,
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

func (ar *AnnouncementDBRepository) GetTopN(limit int, categories []int) ([]Announcement, error) {
	var (
		query string
		args  []interface{}
	)

	if len(categories) > 0 {
		query = `
			SELECT id, name, description, user_seller_id, price, category, discount, is_active, rating, rating_count, created_at
			FROM announcement
			WHERE is_active = TRUE AND category = ANY($1)
			ORDER BY rating DESC, rating_count DESC
			LIMIT $2
		`
		args = append(args, pq.Array(categories), limit)
	} else {
		query = `
			SELECT id, name, description, user_seller_id, price, category, discount, is_active, rating, rating_count, created_at
			FROM announcement
			WHERE is_active = TRUE
			ORDER BY rating DESC, rating_count DESC
			LIMIT $1
		`
		args = append(args, limit)
	}

	rows, err := ar.DB.Query(query, args...)
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
	docs, err := ar.ElasticService.SearchByName(context.Background(), query)
	if err != nil {
		ar.Logger.Errorf("Elastic search error: %v", err)
		return nil, errors.ErrSearch
	}

	if len(docs) == 0 {
		return []Announcement{}, nil
	}

	ids := make([]string, len(docs))
	for i, doc := range docs {
		ids[i] = doc.ID
	}

	// Формируем placeholders: $1, $2, ...
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	queryStr := fmt.Sprintf(`
		SELECT 
		    id, 
		    name, 
		    description, 
		    user_seller_id, 
		    price, 
		    category, 
		    discount, 
		    is_active, 
		    rating, 
		    rating_count, 
		    created_at
		FROM announcement
		WHERE id IN (%s)
	`,
		strings.Join(placeholders, ","),
	)

	rows, err := ar.DB.Query(queryStr, args...)
	if err != nil {
		ar.Logger.Errorf("PostgreSQL search query failed: %v", err)
		return nil, errors.ErrDBInternal
	}
	defer rows.Close()

	// Сохраняем по ID, чтобы восстановить порядок
	annByID := make(map[string]Announcement)
	for rows.Next() {
		var a Announcement
		if err := rows.Scan(
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
		); err != nil {
			ar.Logger.Errorf("Row scan failed: %v", err)
			return nil, errors.ErrDBInternal
		}
		annByID[a.ID] = a
	}

	if err := rows.Err(); err != nil {
		ar.Logger.Errorf("Rows iteration error: %v", err)
		return nil, errors.ErrDBInternal
	}

	var result []Announcement
	for _, id := range ids {
		if ann, ok := annByID[id]; ok {
			result = append(result, ann)
		}
	}

	return result, nil
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

func (ar *AnnouncementDBRepository) GetInfoForShoppingCart(ids []string) ([]types.InfoForSC, error) {
	if len(ids) == 0 {
		// Если нет id, сразу возвращаем пустой слайс
		return []types.InfoForSC{}, nil
	}

	// Формируем плейсхолдеры для IN ($1, $2, ..., $n)
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "$" + fmt.Sprint(i+1)
		args[i] = id
	}

	query := `
	SELECT id, name, price, discount, is_active, rating
	FROM announcement
	WHERE id IN (` + strings.Join(placeholders, ",") + `)
	`

	rows, err := ar.DB.Query(query, args...)
	if err != nil {
		ar.Logger.Errorf("Error getting info for shopping card: %v", err)
		return nil, errors.ErrDBInternal
	}
	defer rows.Close()

	var infos []types.InfoForSC
	for rows.Next() {
		var info types.InfoForSC
		if err := rows.Scan(
			&info.ID,
			&info.Name,
			&info.Price,
			&info.Discount,
			&info.IsActive,
			&info.Rating,
		); err != nil {
			ar.Logger.Errorf("Error scanning row in GetInfoForShoppingCard: %v", err)
			return nil, errors.ErrDBInternal
		}
		infos = append(infos, info)
	}

	return infos, nil
}
