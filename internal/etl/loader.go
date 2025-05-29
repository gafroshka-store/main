package etl

import (
	"context"
	"database/sql"
	"fmt"
	elasticService "gafroshka-main/internal/elastic_search"
	"gafroshka-main/internal/types/elastic"
	myErr "gafroshka-main/internal/types/errors"
	"go.uber.org/zap"
	"strings"
)

type ElasticLoader struct {
	Service *elasticService.ElasticService
	Logger  *zap.SugaredLogger
	DB      *sql.DB
}

func NewElasticLoader(service *elasticService.ElasticService, logger *zap.SugaredLogger, db *sql.DB) *ElasticLoader {
	return &ElasticLoader{
		Service: service,
		Logger:  logger,
		DB:      db,
	}
}

// Load - загружает подготовленные ElasticDoc в индекс ElasticSearch
// Принимает массив ElasticDoc, возвращает error
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

	// Сбор id
	ids := make([]interface{}, len(docs))
	for i, doc := range docs {
		ids[i] = doc.ID
	}

	// Динамическая генерация плейсхолдеров: $1, $2, ...
	placeholders := make([]string, len(ids))
	for i := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	query := fmt.Sprintf(
		"UPDATE announcement SET searching = TRUE WHERE id IN (%s)",
		strings.Join(placeholders, ", "),
	)

	_, err = l.DB.ExecContext(ctx, query, ids...)
	if err != nil {
		l.Logger.Errorw("Failed to update documents in PostgreSQL", zap.Error(err))
		return myErr.ErrDBInternal
	}

	return nil
}
