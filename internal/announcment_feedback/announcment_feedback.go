package announcmentfeedback

type Feedback struct {
	ID             string `json:"id"`
	AnnouncementID string `json:"announcement_id" binding:"required"`
	UserWriterID   string `json:"user_writer_id" binding:"required"`
	Comment        string `json:"comment"`
	Rating         int    `json:"rating" binding:"required,gte=0,lte=5"`
}

type FeedbackRepo interface {
	Create(feedback Feedback) (Feedback, error)
	Delete(feedbackID string) error
	GetByAnnouncementID(announcementID string) ([]Feedback, error)
	Update(feedbackID string, comment string, rating int) (Feedback, error)
}
