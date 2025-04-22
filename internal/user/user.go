package user

import (
	types "gafroshka-main/internal/types/user"
	"time"
)

// User структура пользователя
type User struct {
	ID               string    `json:"user_id"` // uuid
	Name             string    `json:"name"`
	Surname          string    `json:"surname"`
	RegistrationDate time.Time `json:"registration_data"`
	Email            string    `json:"email"`
	PhoneNumber      string    `json:"phone_number"`
	PasswordHash     string    `json:"password_hash"`
	Balance          int64     `json:"balance"`
	DealsCount       int       `json:"deals_count"` // Кол-во сделок
}

// UserRepo интерфейс удовлетворяющий методам сущности пользователя
type UserRepo interface {
	// Authorize регистрирует/авторизует пользователя пользователя
	Authorize(login, password string) (User, error)
	// Info возвращает информацию о пользователи
	Info(userID string) (User, error)
	// ChangeProfile меняет поля пользователя с userID по updateUser
	ChangeProfile(userID string, updateUser types.ChangeUser) (User, error)
}
