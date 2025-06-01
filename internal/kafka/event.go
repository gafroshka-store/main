package kafka

import "time"

type EventType string

const (
	EventTypeSearch   EventType = "search"
	EventTypeView     EventType = "view"
	EventTypePurchase EventType = "purchase"
)

type Event struct {
	UserID     string    `json:"user_id"`
	Type       EventType `json:"type"`
	Categories []int     `json:"categories,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}
