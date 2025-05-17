package session

import (
	"context"
	"net/http"
	"time"
)

// Session - структура сессии
type Session struct {
	ID        string
	UserID    string
	StartTime time.Time
	EndTime   time.Time
}

// SessionRepo - репозиторий для работы с сессиями
//
//go:generate mockgen -source=internal/session/session.go -destination=internal/mocks/mock_session_repo.go -package=mocks
type SessionRepo interface {
	// CreateSession - создает новую сессию для уникального пользователя и кладет ее в Redis
	// Возвращает Session
	CreateSession(ctx context.Context, w http.ResponseWriter, userID string, email string) (*Session, error)
	// CheckSession - проверяет существование сессии в Redis и не истекла ли она
	// Возвращает *Session в случае успеха, иначе nil
	CheckSession(r *http.Request) (*Session, error)

	// ExtendSession - продлевает сессию на 15 минут, если пользователь активно пользуется сервисом
	// Возвращает error
	ExtendSession(ctx context.Context, sessionID string) error
}
