package errors

import "errors"

var (
	ErrDBInternal = errors.New("ошибка внутри базы")
	ErrNotFound   = errors.New("не найдено")
)
