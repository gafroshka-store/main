package user

import (
	"database/sql"
	"errors"
	"fmt"
	myErr "gafroshka-main/internal/types/errors"
	types "gafroshka-main/internal/types/user"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

type UserDBRepository struct {
	DB     *sql.DB
	Logger *zap.SugaredLogger
}

func NewUserDBRepository(db *sql.DB, l *zap.SugaredLogger) *UserDBRepository {
	return &UserDBRepository{
		DB:     db,
		Logger: l,
	}
}

func (ur *UserDBRepository) CreateUser(u types.CreateUser) (*User, error) {
	if u, _ := ur.CheckUser(u.Email, u.Password); u != nil { // nolint:errcheck
		return nil, myErr.ErrAlreadyExists
	}

	query := `
	INSERT INTO users (
	   id, 
	   name, 
	   surname, 
	   day_of_birth,
	   sex, 
	   registration_date,
	   email,
	   phone_number, 
	   password_hash,
	   balance,
	   deals_count,
	   rating,
	   rating_count
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
`
	userID := uuid.NewString()
	registrationDate := time.Now()
	hashP, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	_, err = ur.DB.Exec(
		query,
		userID, u.Name, u.Surname,
		u.DateOfBirth, u.Sex, registrationDate, u.Email,
		u.PhoneNumber, string(hashP), 0, 0, 0, 0,
	)
	if err != nil {
		ur.Logger.Warnf("Ошибка при создании информации пользователя: %v", err)
		return nil, fmt.Errorf("%w: %w", myErr.ErrDBInternal, err)
	}

	return &User{
		ID:               userID,
		Name:             u.Name,
		Surname:          u.Surname,
		DayOfBirth:       u.DateOfBirth,
		Sex:              u.Sex,
		RegistrationDate: registrationDate,
		Email:            u.Email,
		PhoneNumber:      u.PhoneNumber,
		PasswordHash:     string(hashP),
		Balance:          0,
		DealsCount:       0,
		Rating:           0,
		RatingCount:      0,
	}, nil
}

func (ur *UserDBRepository) CheckUser(email, password string) (*User, error) {
	query := `
		SELECT id, 
		   name,
		   surname,
		   day_of_birth,
		   sex,
		   registration_date,
		   email,
		   phone_number,
		   password_hash,
		   balance,
		   deals_count,
		   rating,
		   rating_count
	FROM users
	WHERE email = $1 
	`
	var checkPassword string
	var u User
	err := ur.DB.QueryRow(query, email).Scan(
		&u.ID, &u.Name, &u.Surname, &u.DayOfBirth,
		&u.Sex, &u.RegistrationDate, &u.Email,
		&u.PhoneNumber, &checkPassword, &u.Balance,
		&u.DealsCount, &u.Rating, &u.RatingCount,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, myErr.ErrNotFound
		}
		return nil, fmt.Errorf("%w: %w", myErr.ErrDBInternal, err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(checkPassword), []byte(password)); err != nil {
		return nil, myErr.ErrBadPassword
	}

	return &u, nil
}

func (ur *UserDBRepository) Info(userID string) (*User, error) {
	query := `
	SELECT id, 
		   name,
		   surname,
		   day_of_birth,
		   sex,
		   registration_date,
		   email,
		   phone_number,
		   balance,
		   deals_count,
		   rating,
		   rating_count
	FROM users
	WHERE id = $1
	`
	u := &User{}
	err := ur.DB.QueryRow(query, userID).
		Scan(
			&u.ID, &u.Name, &u.Surname, &u.DayOfBirth,
			&u.Sex, &u.RegistrationDate, &u.Email,
			&u.PhoneNumber, &u.Balance, &u.DealsCount,
			&u.Rating, &u.RatingCount,
		)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, myErr.ErrNotFound
		}
		ur.Logger.Warnf("Ошибка при получения информации о пользователе: %v", err)
		return nil, myErr.ErrDBInternal
	}

	return u, nil
}

func (ur *UserDBRepository) ChangeProfile(userID string, updateUser types.ChangeUser) (*User, error) {
	fields := []string{}
	args := []interface{}{}
	argID := 1

	// Динамически добавляем поля в обновление
	if updateUser.Name != "" {
		fields = append(fields, "name = $"+strconv.Itoa(argID))
		args = append(args, updateUser.Name)
		argID++
	}
	if updateUser.Surname != "" {
		fields = append(fields, "surname = $"+strconv.Itoa(argID))
		args = append(args, updateUser.Surname)
		argID++
	}
	if updateUser.Email != "" {
		fields = append(fields, "email = $"+strconv.Itoa(argID))
		args = append(args, updateUser.Email)
		argID++
	}
	if updateUser.PhoneNumber != "" {
		fields = append(fields, "phone_number = $"+strconv.Itoa(argID))
		args = append(args, updateUser.PhoneNumber)
		argID++
	}

	if len(fields) == 0 {
		return ur.Info(userID) // Если ничего не обновляется, просто вернуть текущие данные
	}

	query := "UPDATE users SET " + strings.Join(fields, ", ") + " WHERE id = $" + strconv.Itoa(argID) // nolint:gosec
	args = append(args, userID)

	res, err := ur.DB.Exec(query, args...)
	if err != nil {
		ur.Logger.Warnf("Ошибка при обновлении профиля: %v", err)
		return nil, myErr.ErrDBInternal
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		ur.Logger.Warnf("Не удалось получить количество обновлённых строк: %v", err)
		return nil, myErr.ErrDBInternal
	}

	if rowsAffected == 0 {
		return nil, myErr.ErrNotFound
	}

	return ur.Info(userID) // Возвращаем обновлённые данные пользователя
}
