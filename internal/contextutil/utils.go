package contextutil

import (
	"context"
	"gafroshka-main/internal/middleware"
)

// GetUserIDFromContext извлекает userID из контекста
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	sess, ok := middleware.GetSessionFromContext(ctx)
	if !ok || sess == nil {
		return "", false
	}
	return sess.UserID, true
}
