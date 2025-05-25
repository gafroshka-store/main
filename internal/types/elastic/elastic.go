package elastic

// ElasticDoc - структура документа для хранения в ES
type ElasticDoc struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    int    `json:"category,omitempty"`
}
