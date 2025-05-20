package user_feedback

// UpdateUserFeedback - структура отзыва на продавца с полями для изменения
type UpdateUserFeedback struct {
	Comment string `json:"comment"`
	Rating  int    `json:"rating"`
}
