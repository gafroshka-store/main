package user

// ChangeUser структура пользователя с полями для изменения
type ChangeUser struct {
	Name        string `json:"name"`
	Surname     string `json:"surname"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phone_number"`
}
