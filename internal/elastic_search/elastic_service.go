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

// IndexAnnouncement - записывает один элемент в индекс
// Принимает ElasticDoc, возвращает error
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

// BulkIndex - записывает butch данных в индекс
// Принимает массив подготовленных к загрузке в ES документов ElasticDoc, возвращает error
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

// SearchByName - ищет товар по имени с использованием полнотекствоого поиска
// Принимает запрос, возвращает массив подходящих документов ElasticDoc и error
func (s *ElasticService) SearchByName(ctx context.Context, query string) ([]esDoc.ElasticDoc, error) {
	searchQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"match": map[string]interface{}{
				"name": map[string]interface{}{
					"query":     query,
					"fuzziness": "AUTO",
				},
			},
		},
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(searchQuery); err != nil {
		s.Logger.Errorw("Failed to encode search query", zap.Error(err))
		return nil, err
	}

	res, err := s.Client.Search(
		s.Client.Search.WithContext(ctx),
		s.Client.Search.WithIndex(s.Index),
		s.Client.Search.WithBody(&buf),
		s.Client.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		s.Logger.Errorw("Failed to perform search query", zap.Error(err))
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		s.Logger.Errorw("Elasticsearch search error", zap.String("response", res.String()))
		return nil, myErr.ErrSearch
	}

	var esResp struct {
		Hits struct {
			Hits []struct {
				Source esDoc.ElasticDoc `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err = json.NewDecoder(res.Body).Decode(&esResp); err != nil {
		s.Logger.Errorw("Failed to decode search response", zap.Error(err))
		return nil, err
	}

	results := make([]esDoc.ElasticDoc, 0, len(esResp.Hits.Hits))
	for _, hit := range esResp.Hits.Hits {
		results = append(results, hit.Source)
	}

	return results, nil
}

// EnsureIndex - проверяет, существует ли индекс с нужными настройками, если нет - создает его
// Возвращает error
func (s *ElasticService) EnsureIndex(ctx context.Context) error {
	res, err := s.Client.Indices.Exists([]string{s.Index}, s.Client.Indices.Exists.WithContext(ctx))
	if err != nil {
		s.Logger.Errorw("Failed to check if index exists", zap.Error(err))
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == 200 {
		s.Logger.Infof("Index '%s' already exists", s.Index)
		return nil
	}

	settings := map[string]interface{}{
		"settings": map[string]interface{}{
			"analysis": map[string]interface{}{
				"filter": map[string]interface{}{
					"autocomplete_filter": map[string]interface{}{
						"type":     "edge_ngram",
						"min_gram": 2,
						"max_gram": 20,
					},
				},
				"analyzer": map[string]interface{}{
					"autocomplete": map[string]interface{}{
						"type":      "custom",
						"tokenizer": "standard",
						"filter":    []string{"lowercase", "autocomplete_filter"},
					},
				},
			},
		},
		"mappings": map[string]interface{}{
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":            "text",
					"analyzer":        "autocomplete",
					"search_analyzer": "standard",
				},
				"description": map[string]interface{}{
					"type": "text",
				},
				"category": map[string]interface{}{
					"type": "integer",
				},
			},
		},
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(settings); err != nil {
		s.Logger.Errorw("Failed to encode index settings", zap.Error(err))
		return err
	}

	createRes, err := s.Client.Indices.Create(s.Index,
		s.Client.Indices.Create.WithContext(ctx),
		s.Client.Indices.Create.WithBody(&buf),
	)
	if err != nil {
		s.Logger.Errorw("Failed to create index", zap.Error(err))
		return err
	}
	defer createRes.Body.Close()

	if createRes.IsError() {
		s.Logger.Errorw("Elasticsearch index creation error", zap.String("response", createRes.String()))
		return myErr.ErrIndexing
	}

	s.Logger.Infof("Index '%s' created successfully", s.Index)
	return nil
}
