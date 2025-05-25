package etl

import (
	"context"
	"go.uber.org/zap"
	"time"
)

type Pipeline struct {
	extractor   *PostgresExtractor
	transformer *Transformer
	loader      *ElasticLoader
	logger      *zap.SugaredLogger
	interval    time.Duration
}

func NewPipeline(
	extractor *PostgresExtractor,
	transformer *Transformer,
	loader *ElasticLoader,
	logger *zap.SugaredLogger,
	interval time.Duration,
) *Pipeline {
	return &Pipeline{
		extractor:   extractor,
		transformer: transformer,
		loader:      loader,
		logger:      logger,
		interval:    interval,
	}
}

func (p *Pipeline) Run(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	p.logger.Infow("ETL pipeline started")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.logger.Infow("Running ETL pipeline iteration")

			now := time.Now()
			from := now.Add(-p.interval)

			// EXTRACT
			announcements, err := p.extractor.ExtractNew(ctx, from)
			if err != nil {
				p.logger.Errorw("Extracting failed", zap.Error(err))

				continue
			}
			if len(announcements) == 0 {
				p.logger.Infow("No new announcements to process")

				continue
			}

			// TRANSFORM
			docs := p.transformer.Transform(announcements)

			// LOAD
			err = p.loader.Load(ctx, docs)
			if err != nil {
				p.logger.Errorw("Error while loading docs to ES", zap.Error(err))
				continue
			}

			p.logger.Infof("ETL pipeline completed, successfully loaded %d docs", len(announcements))
		}
	}
}
