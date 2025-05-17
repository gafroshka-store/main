package announcmentFeedback

type Feedback struct {
	ID             string `json:"id"`
	AnnouncementID string `json:"announcement_id" binding:"required"`
	UserWriterID   string `json:"user_writer_id" binding:"required"`
	Comment        string `json:"comment"`
	Rating         int    `json:"rating" binding:"required,gte=0,lte=5"`
}
