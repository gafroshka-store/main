package kafka

import "time"

type EventType string

const (
	Search    EventType = "search"
	View      EventType = "view"
	AddToCart EventType = "addToCart"
	Purchase  EventType = "purchase"
)

type Event struct {
	UserID     string    `json:"user_id"`
	Type       EventType `json:"type"`
	Categories []int     `json:"categories,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}
