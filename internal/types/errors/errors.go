package errors

import (
	"encoding/json"
	"errors"
	"net/http"

	"go.uber.org/zap"
)

var (
	ErrDBInternal    = errors.New("database internal error")
	ErrNotFound      = errors.New("record not found")
	ErrAlreadyExists = errors.New("record already exists")
	ErrNotFoundFeedback = errors.New("отзыв с таким id не найден")
	ErrInvalidRating    = errors.New("рейтинг должен быть от 1 до 5")
	ErrSessionNotFound  = errors.New("session not found")
	ErrSessionIsExpired = errors.New("session is expired")
	ErrNoAuth           = errors.New("authorization required")

	ErrBadPassword = errors.New("bad password")
	ErrBadID       = errors.New("bad id")
)

type ErrorServer struct {
	Message string `json:"message"`
}

func (e *ErrorServer) Error() string {
	return e.Message
}

/*
NewErrorServer
Функция имеет возможность принимать "nil ошибку"
при получении nil наша функция понимает, что нам
просто надо отдать саксесс клиенту
*/
func NewErrorServer(err error) ErrorServer {
	if err == nil {
		return ErrorServer{
			Message: "success",
		}
	}

	return ErrorServer{
		Message: err.Error(),
	}
}

func SendErrorTo(w http.ResponseWriter, err error, statusCode int, logger *zap.SugaredLogger) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if errEncode := json.NewEncoder(w).Encode(NewErrorServer(err)); errEncode != nil {
		logger.Error(errEncode)
	}
}
