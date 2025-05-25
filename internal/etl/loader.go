package etl

import (
	"context"
	elasticService "gafroshka-main/internal/elastic_search"
	"gafroshka-main/internal/types/elastic"
	"go.uber.org/zap"
)

type ElasticLoader struct {
	Service *elasticService.ElasticService
	Logger  *zap.SugaredLogger
}

func NewElasticLoader(service *elasticService.ElasticService, logger *zap.SugaredLogger) *ElasticLoader {
	return &ElasticLoader{
		Service: service,
		Logger:  logger,
	}
}

func (l *ElasticLoader) Load(ctx context.Context, docs []elastic.ElasticDoc) error {
	if len(docs) == 0 {
		l.Logger.Infow("No documents to load")
		return nil
	}

	l.Logger.Infow("Loading documents to Elasticsearch", "count", len(docs))
	err := l.Service.BulkIndex(ctx, docs)
	if err != nil {
		l.Logger.Errorw("Failed to bulk index documents", zap.Error(err))
		return err
	}

	l.Logger.Infow("Successfully indexed documents", "count", len(docs))
	return nil
}
