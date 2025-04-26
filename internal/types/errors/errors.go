package errors

import "errors"

var (
	ErrDBInternal       = errors.New("ошибка внутри базы")
	ErrNotFoundFeedback = errors.New("отзыв с таким id не найден")
)
