package middleware

import (
	"context"
	"gafroshka-main/internal/session"
	"net/http"
)

type SessKey string

var sessKey SessKey = "sessionKey"

func Auth(sm *session.SessionRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Проверка сессии пользователя
			sess, err := sm.CheckSession(r)
			if err != nil {
				http.Redirect(w, r, "/", http.StatusFound)
				return
			}

			// Добавляем сессию в контекст и передаем дальше
			ctx := ContextWithSession(r.Context(), sess)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ContextWithSession(ctx context.Context, s *session.Session) context.Context {
	// создаем новый контекст с нашим ключом и сессией
	return context.WithValue(ctx, sessKey, s)
}
