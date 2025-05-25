package session

import (
	"context"
	"encoding/json"
	"gafroshka-main/internal/types/errors"
	"github.com/alicebob/miniredis/v2"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
	"net/http"
	"net/http/httptest"

	"testing"
	"time"
)

func setupTestRepo(t *testing.T) (*SessionRepository, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	assert.NoError(t, err)

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	logger := zaptest.NewLogger(t).Sugar()
	repo := NewSessionRepository(rdb, logger, "secret", 15*time.Minute)

	return repo, mr
}

func TestCreateSession(t *testing.T) {
	repo, mr := setupTestRepo(t)
	defer mr.Close()

	w := httptest.NewRecorder()
	ctx := context.Background()

	userID := "user-123"
	email := "user@example.com"

	sess, err := repo.CreateSession(ctx, w, userID, email)
	assert.NoError(t, err)
	assert.NotNil(t, sess)

	// Проверка записи в Redis
	val, err := mr.Get(sess.ID)
	assert.NoError(t, err)
	assert.NotEmpty(t, val)

	// Проверка ответа содержит токен
	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response struct {
		Token string `json:"token"`
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response.Token)
}

func TestCheckSession_Success(t *testing.T) {
	repo, mr := setupTestRepo(t)
	defer mr.Close()

	// Создание сессии
	sessionData := Session{
		ID:        "session-1",
		UserID:    "user-id",
		StartTime: time.Now().Add(-5 * time.Minute),
		EndTime:   time.Now().Add(10 * time.Minute),
	}
	data, _ := json.Marshal(sessionData) // nolint:errcheck
	mr.Set("session-1", string(data))    // nolint:errcheck

	// Генерация токена
	tokenStr := generateJWT(t, "secret", sessionData.ID, sessionData.UserID)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	result, err := repo.CheckSession(req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, sessionData.ID, result.ID)
}

func TestCheckSession_MissingAuthHeader(t *testing.T) {
	repo, _ := setupTestRepo(t)

	req := httptest.NewRequest("GET", "/", nil)

	sess, err := repo.CheckSession(req)
	assert.Nil(t, sess)
	assert.ErrorIs(t, err, errors.ErrNoAuth)
}

func TestCheckSession_InvalidToken(t *testing.T) {
	repo, _ := setupTestRepo(t)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.value")

	sess, err := repo.CheckSession(req)
	assert.Nil(t, sess)
	assert.ErrorIs(t, err, errors.ErrNoAuth)
}

func TestCheckSession_SessionExpired(t *testing.T) {
	repo, mr := setupTestRepo(t)
	defer mr.Close()

	sessionData := Session{
		ID:        "expired-session",
		UserID:    "user-id",
		StartTime: time.Now().Add(-30 * time.Minute),
		EndTime:   time.Now().Add(-10 * time.Minute),
	}
	data, _ := json.Marshal(sessionData)    // nolint:errcheck
	mr.Set("expired-session", string(data)) // nolint:errcheck

	tokenStr := generateJWT(t, "secret", sessionData.ID, sessionData.UserID)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	sess, err := repo.CheckSession(req)
	assert.Nil(t, sess)
	assert.ErrorIs(t, err, errors.ErrSessionIsExpired)

	exists := mr.Exists("expired-session")
	assert.False(t, exists)
}

func generateJWT(t *testing.T, secret, sessionID, userID string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"session_id": sessionID,
		"id":         userID,
		"iat":        time.Now().Unix(),
		"exp":        time.Now().Add(15 * time.Minute).Unix(),
	})
	tokenStr, err := token.SignedString([]byte(secret))
	assert.NoError(t, err)
	return tokenStr
}
