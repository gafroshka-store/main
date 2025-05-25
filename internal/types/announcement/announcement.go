package announcement

// CreateAnnouncement - форма для создания объявления
type CreateAnnouncement struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	UserSellerID string `json:"user_seller_id"`
	Price        int64  `json:"price"`
	Category     int    `json:"category"`
	Discount     int    `json:"discount"`
}

// InfoForSC - форма для получения информации для вывода в корзине
type InfoForSC struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Price    int64   `json:"price"`
	Discount int     `json:"discount"`
	IsActive bool    `json:"is_active"`
	Rating   float64 `json:"rating"`
}
