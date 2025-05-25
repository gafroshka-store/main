package shopping_cart

// ShoppingCart структура корзины пользователей
type ShoppingCart struct {
	UserID         string `json:"user_id"`
	AnnouncementID string `json:"announcement_id"`
}

// ShoppingCartRepo интерфейс для работы репозитория корзины покупок
//
//go:generate mockgen -source=shopping_cart.go -destination=../mocks/mock_shopping_cart_repo.go -package=mocks
type ShoppingCartRepo interface {
	// AddAnnouncement добавляет пользователю в корзину товар
	AddAnnouncement(userID string, announcementID string) error
	// DeleteAnnouncement удаляет из корзины покупки
	DeleteAnnouncement(userID string, announcementID string) error
	// GetByUserID получает корзину пользователя (список id объявлений)
	GetByUserID(userID string) ([]string, error)
}
