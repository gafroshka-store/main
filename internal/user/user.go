package user

import (
	types "gafroshka-main/internal/types/user"
	"time"
)

const (
	SexManT   = true
	SexWomenT = false
)

// User структура пользователя
type User struct {
	ID               string    `json:"user_id"` // uuid
	Name             string    `json:"name"`
	Surname          string    `json:"surname"`
	DayOfBirth       time.Time `json:"day_of_birth"`
	Sex              bool      `json:"sex"`
	RegistrationDate time.Time `json:"registration_date"`
	Email            string    `json:"email"`
	PhoneNumber      string    `json:"phone_number"`
	PasswordHash     string    `json:"password_hash"`
	Balance          int64     `json:"balance"`
	DealsCount       int       `json:"deals_count"` // Кол-во сделок
	Rating           float64   `json:"rating"`
	RatingCount      int       `json:"rating_count"`
}

// UserRepo интерфейс удовлетворяющий методам сущности пользователя
//
//go:generate mockgen -source=internal/user/user.go -destination=internal/mocks/mock_user_repo.go -package=mocks
type UserRepo interface {
	// CheckUser - проверяет пользователя по почте и паролю
	CheckUser(email, password string) (*User, error)
	// CreateUser создает пользователя
	CreateUser(u types.CreateUser) (*User, error)
	// Info возвращает информацию о пользователи
	Info(userID string) (*User, error)
	// ChangeProfile меняет поля пользователя с userID по updateUser
	ChangeProfile(userID string, updateUser types.ChangeUser) (*User, error)
}
