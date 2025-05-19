package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"net/http"
	"strings"
	"time"

	errorspkg "gafroshka-main/internal/types/errors"
)

type SessionRepository struct {
	RedisClient  *redis.Client
	Logger       *zap.SugaredLogger
	tokenSecret  string
	baseDuration time.Duration
}

func NewSessionRepository(
	redisClient *redis.Client,
	logger *zap.SugaredLogger,
	tokenSecret string,
	baseDuration time.Duration,
) *SessionRepository {
	return &SessionRepository{
		RedisClient:  redisClient,
		Logger:       logger,
		tokenSecret:  tokenSecret,
		baseDuration: baseDuration,
	}
}

func (sessionRepository *SessionRepository) CreateSession(
	ctx context.Context,
	w http.ResponseWriter,
	userID string,
	email string,
) (*Session, error) {
	now := time.Now()

	// Создаём новую сессию
	sessionID := uuid.New().String()
	session := &Session{
		ID:        sessionID,
		UserID:    userID,
		StartTime: now,
		EndTime:   now.Add(sessionRepository.baseDuration),
	}

	// Сохраняем сессию в Redis
	if err := sessionRepository.saveSessionToRedis(ctx, session); err != nil {
		// Логируется внутри saveSessionToRedis
		return nil, err
	}

	// Генерируем JWT токен
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email":      email,
		"id":         userID,
		"iat":        session.StartTime.Unix(),
		"exp":        session.EndTime.Unix(),
		"session_id": session.ID,
	})

	tokenStr, err := token.SignedString([]byte(sessionRepository.tokenSecret))
	if err != nil {
		sessionRepository.Logger.Error("Failed to sign JWT token", zap.Error(err))
		return nil, fmt.Errorf("error signing token: %w", err)
	}

	// Формируем JSON-ответ
	response := struct {
		Token string `json:"token"`
	}{
		Token: tokenStr,
	}
	respJSON, err := json.Marshal(response)
	if err != nil {
		sessionRepository.Logger.Error("Failed to marshal JSON response", zap.Error(err))
		return nil, fmt.Errorf("error marshaling response: %w", err)
	}

	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(respJSON); err != nil {
		sessionRepository.Logger.Error("Failed to write response", zap.Error(err))
		return nil, fmt.Errorf("error writing response: %w", err)
	}

	sessionRepository.Logger.Infof("Session %s created and JWT sent for user: %s", session.ID, email)
	return session, nil
}

func (sessionRepository *SessionRepository) CheckSession(r *http.Request) (*Session, error) { // nolint:gocyclo
	const bearerPrefix = "Bearer "

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, errorspkg.ErrNoAuth
	}

	if !strings.HasPrefix(authHeader, bearerPrefix) {
		return nil, errorspkg.ErrNoAuth
	}

	tokenStr := strings.TrimPrefix(authHeader, bearerPrefix)

	// Разбор токена
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			sessionRepository.Logger.Warnf("Unexpected signing method: %v", token.Header["alg"])
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(sessionRepository.tokenSecret), nil
	})
	if err != nil || !token.Valid {
		sessionRepository.Logger.Warnf("Invalid JWT token: %v", err)
		return nil, errorspkg.ErrNoAuth
	}

	// Извлечение claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["session_id"] == nil {
		sessionRepository.Logger.Warn("Missing session_id claim in JWT")
		return nil, errorspkg.ErrNoAuth
	}

	sessionID, ok := claims["session_id"].(string)
	if !ok {
		sessionRepository.Logger.Warn("session_id claim is not a string")
		return nil, errorspkg.ErrNoAuth
	}

	// Поиск сессии по ID
	ctx := context.Background()
	session, err := sessionRepository.getSessionFromRedis(ctx, sessionID)
	if err != nil {
		return nil, err // уже логируется внутри
	}

	if time.Now().After(session.EndTime) {
		_ = sessionRepository.RedisClient.Del(ctx, sessionID).Err() // nolint:errcheck
		return nil, errorspkg.ErrSessionIsExpired
	}

	return session, nil
}

func (sessionRepository *SessionRepository) ExtendSession(
	ctx context.Context,
	sessionID string,
) error {
	session, err := sessionRepository.getSessionFromRedis(ctx, sessionID)
	if err != nil {
		// Все логирование происходит внутри getSessionFromRedis
		return err
	}

	session.EndTime = time.Now().Add(sessionRepository.baseDuration)

	if err = sessionRepository.saveSessionToRedis(ctx, session); err != nil {
		sessionRepository.Logger.Error(
			"Failed update session end time",
			zap.Error(err),
			zap.String("sessionID", sessionID),
		)

		return err
	}

	return nil
}

func (sessionRepository *SessionRepository) saveSessionToRedis(
	ctx context.Context,
	session *Session,
) error {
	sessionDataJSON, err := json.Marshal(session)
	if err != nil {
		sessionRepository.Logger.Error(
			"Failed encode session to JSON",
			zap.Error(err),
			zap.String("sessionID", session.ID),
		)

		return err
	}

	err = sessionRepository.RedisClient.Set(ctx, session.ID, sessionDataJSON, sessionRepository.baseDuration).Err()
	if err != nil {
		sessionRepository.Logger.Error(
			"Failed save session to Redis",
			zap.Error(err),
			zap.String("sessionID", session.ID),
		)

		return err
	}
	sessionRepository.Logger.Info(
		fmt.Sprintf("Session %s saved to Redis successfully", session.ID),
	)

	return nil
}

func (sessionRepository *SessionRepository) getSessionFromRedis(
	ctx context.Context,
	sessionID string,
) (*Session, error) {
	sessionDataJSON, err := sessionRepository.RedisClient.Get(ctx, sessionID).Bytes()
	if err != nil {
		sessionRepository.Logger.Error(
			"Failed get session from Redis",
			zap.Error(err),
			zap.String("sessionID", sessionID),
		)

		if errors.Is(err, redis.Nil) {
			sessionRepository.Logger.Error(
				fmt.Sprintf("Session %s not found in Redis", sessionID),
			)

			return nil, errorspkg.ErrSessionNotFound
		}

		return nil, err
	}
	sessionRepository.Logger.Info(
		fmt.Sprintf("Session %s got from Redis successfully", sessionID),
	)

	var session Session
	if err = json.Unmarshal(sessionDataJSON, &session); err != nil {
		sessionRepository.Logger.Error(
			"Failed decode session from JSON",
			zap.Error(err),
			zap.String("sessionID", sessionID),
		)

		return nil, err
	}

	return &session, nil
}
