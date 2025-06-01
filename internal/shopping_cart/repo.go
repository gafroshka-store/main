package shopping_cart

import (
	"errors"
	myErr "gafroshka-main/internal/types/errors"

	"database/sql"

	"go.uber.org/zap"
)

type ShoppingCartRepository struct {
	DB     *sql.DB
	Logger *zap.SugaredLogger
}

func NewShoppingCartRepository(db *sql.DB, logger *zap.SugaredLogger) *ShoppingCartRepository {
	return &ShoppingCartRepository{
		DB:     db,
		Logger: logger,
	}
}

// AddAnnouncement добавляет пользователю в корзину товар
func (scr *ShoppingCartRepository) AddAnnouncement(userID string, announcementID string) error {
	query := `
	INSERT INTO shopping_cart(user_id, announcement_id) 
	VALUES ($1, $2) ON CONFLICT (user_id, announcement_id)
	DO NOTHING
`
	_, err := scr.DB.Exec(query, userID, announcementID)
	if err != nil {
		scr.Logger.Errorf("Ошибка при добавлении объявления: %v", err)
		return myErr.ErrDBInternal
	}

	return nil
}

// DeleteAnnouncement удаляет из корзины покупки
func (scr *ShoppingCartRepository) DeleteAnnouncement(userID string, announcementID string) error {
	query := `
	DELETE FROM shopping_cart
	WHERE user_id = $1 AND announcement_id = $2
`
	_, err := scr.DB.Exec(query, userID, announcementID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return myErr.ErrNotFound
		}

		scr.Logger.Errorf("Ошибка при удалении из корзины: %v", err)
		return myErr.ErrDBInternal
	}

	return nil
}

// GetByUserID получает корзину пользователя (список id объявлений)
func (scr *ShoppingCartRepository) GetByUserID(userID string) ([]string, error) {
	query := `
	SELECT announcement_id FROM shopping_cart
	WHERE user_id = $1
`
	rows, err := scr.DB.Query(query, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, myErr.ErrNotFound
		}

		scr.Logger.Errorf("Ошибка при получении корзины клиента %v: %v", userID, err)
		return nil, myErr.ErrDBInternal
	}
	defer rows.Close()

	var announcementIDs []string
	for rows.Next() {
		var announcementID string
		if err := rows.Scan(&announcementID); err != nil {
			return nil, myErr.ErrDBInternal
		}

		announcementIDs = append(announcementIDs, announcementID)
	}

	return announcementIDs, nil
}
