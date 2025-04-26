package errors

import "errors"

var (
	ErrDBInternal       = errors.New("ошибка внутри базы")
	ErrNotFoundFeedback = errors.New("Отзыв с таким id не найден")
)
