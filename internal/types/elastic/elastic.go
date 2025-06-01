package elastic

// ElasticDoc - структура документа для хранения в ES
type ElasticDoc struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    int    `json:"category,omitempty"`
}
