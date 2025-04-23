package user

import (
	"database/sql"
	"gafroshka-main/internal/types/errors"
	types "gafroshka-main/internal/types/user"
	wrappers "gafroshka-main/internal/wrappers/auth_wrappers"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

type UserDBRepository struct {
	DB          *sql.DB
	Logger      *zap.SugaredLogger
	AuthWrapper wrappers.AuthWrapperRepo
}

func NewUserDBRepository(db *sql.DB, l *zap.SugaredLogger, aw wrappers.AuthWrapperRepo) *UserDBRepository {
	return &UserDBRepository{
		DB:          db,
		Logger:      l,
		AuthWrapper: aw,
	}
}

func (ur *UserDBRepository) Authorize(login, password string) (User, error) {
	var u User

	return u, nil
}

func (ur *UserDBRepository) Info(userID string) (User, error) {
	var u User

	query := `
	SELECT user_id, 
		   name,
		   surname,
		   day_of_birth,
		   sex,
		   registration_data,
		   email,
		   phone_number,
		   balance,
		   deals_count,
		   rating,
		   rating_count
	FROM users
	WHERE user_id = $1
	`

	err := ur.DB.QueryRow(query, userID).
		Scan(
			&u.ID, &u.Name, &u.Surname, &u.DayOfBirth,
			&u.Sex, &u.RegistrationDate, &u.Email,
			&u.PhoneNumber, &u.Balance, &u.DealsCount,
			&u.Rating, &u.RatingCount,
		)
	if err != nil {
		// Нужно проверить на NoRows
		ur.Logger.Warnf("Ошибка при получения информации о пользователе: %v", err)
		return u, errors.ErrDBInternal
	}

	return u, nil
}

func (ur *UserDBRepository) ChangeProfile(userID string, updateUser types.ChangeUser) (User, error) {
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

	query := "UPDATE users SET " + strings.Join(fields, ", ") + " WHERE user_id = $" + strconv.Itoa(argID) // nolint:gosec
	args = append(args, userID)

	_, err := ur.DB.Exec(query, args...)
	if err != nil {
		ur.Logger.Warnf("Ошибка при обновлении профиля: %v", err)
		return User{}, errors.ErrDBInternal
	}

	return ur.Info(userID) // Возвращаем обновлённые данные пользователя
}
