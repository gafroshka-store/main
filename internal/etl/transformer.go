package etl

import (
	"gafroshka-main/internal/announcement"
	"gafroshka-main/internal/types/elastic"
	"go.uber.org/zap"
)

type Transformer struct {
	Logger *zap.SugaredLogger
}

func NewTransformer(logger *zap.SugaredLogger) *Transformer {
	return &Transformer{
		Logger: logger,
	}
}

func (t *Transformer) Transform(input []announcement.Announcement) []elastic.ElasticDoc {
	docs := make([]elastic.ElasticDoc, 0, len(input))
	for _, a := range input {
		docs = append(docs, elastic.ElasticDoc{
			ID:          a.ID,
			Title:       a.Name,
			Description: a.Description,
			Category:    a.Category,
		})
	}

	t.Logger.Infof("Transformed %d docs succesfully", len(input))

	return docs
}
