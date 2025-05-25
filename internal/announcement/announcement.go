package announcement

import (
	"time"

	types "gafroshka-main/internal/types/announcement"
)

type Announcement struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	UserSellerID string    `json:"user_seller_id"`
	Price        int64     `json:"price"`
	Category     int       `json:"category"`
	Discount     int       `json:"discount"`
	IsActive     bool      `json:"is_active"`
	Rating       float64   `json:"rating"`
	RatingCount  int       `json:"rating_count"`
	CreatedAt    time.Time `json:"created_at"`
}

//go:generate mockgen -source=announcement.go -destination=../mocks/mock_announcement_repo.go -package=mocks
type AnnouncementRepo interface {
	Create(a types.CreateAnnouncement) (*Announcement, error)
	GetTopN(limit int) ([]Announcement, error)
	Search(query string) ([]Announcement, error)
	GetByID(id string) (*Announcement, error)
	UpdateRating(id string, rate int) (*Announcement, error)
	GetInfoForShoppingCart(ids []string) ([]types.InfoForSC, error)
}
