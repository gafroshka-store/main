package elastic

import (
	"bytes"
	"context"
	"encoding/json"
	esDoc "gafroshka-main/internal/types/elastic"
	myErr "gafroshka-main/internal/types/errors"
	"github.com/elastic/go-elasticsearch/v8"
	"go.uber.org/zap"
)

type ElasticService struct {
	Client *elasticsearch.Client
	Logger *zap.SugaredLogger
	Index  string
}

func NewService(client *elasticsearch.Client, logger *zap.SugaredLogger, index string) *ElasticService {
	return &ElasticService{
		Client: client,
		Logger: logger,
		Index:  index,
	}
}

func (s *ElasticService) IndexAnnouncement(ctx context.Context, doc esDoc.ElasticDoc) error {
	id := doc.ID
	body, err := json.Marshal(doc)
	if err != nil {
		s.Logger.Errorw("Failed to marshal document", zap.Error(err))

		return err
	}

	res, err := s.Client.Index(
		s.Index,
		bytes.NewReader(body),
		s.Client.Index.WithContext(ctx),
		s.Client.Index.WithDocumentID(id),
		s.Client.Index.WithRefresh("false"),
	)
	if err != nil {
		s.Logger.Errorw("Failed to index document", zap.Error(err))

		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		s.Logger.Errorf("Indexing error: %s", res.String())

		return myErr.ErrIndexing
	}

	return nil
}

func (s *ElasticService) BulkIndex(ctx context.Context, docs []esDoc.ElasticDoc) error {
	if len(docs) == 0 {
		return nil
	}

	var buf bytes.Buffer

	for _, doc := range docs {
		meta := map[string]map[string]string{
			"index": {
				"_index": s.Index,
				"_id":    doc.ID,
			},
		}
		metaLine, err := json.Marshal(meta)
		if err != nil {
			s.Logger.Errorw("Failed to marshal bulk meta", zap.Error(err))
			return err
		}

		docLine, err := json.Marshal(doc)
		if err != nil {
			s.Logger.Errorw("Failed to marshal doc", zap.Error(err), "doc_id", doc.ID)
			return err
		}

		buf.Write(metaLine)
		buf.WriteByte('\n')
		buf.Write(docLine)
		buf.WriteByte('\n')
	}

	res, err := s.Client.Bulk(bytes.NewReader(buf.Bytes()), s.Client.Bulk.WithContext(ctx))
	if err != nil {
		s.Logger.Errorw("Bulk request failed", zap.Error(err))

		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		s.Logger.Errorw("Bulk indexing returned error", zap.String("response", res.String()))

		return myErr.ErrIndexing
	}

	return nil
}
