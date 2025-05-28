package elastic

import (
	"context"
	"errors"
	esDoc "gafroshka-main/internal/types/elastic"
	myErr "gafroshka-main/internal/types/errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

type mockTransport struct {
	Response    *http.Response
	RoundTripFn func(req *http.Request) (*http.Response, error)
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.RoundTripFn(req)
}

func setupTestService(t *testing.T, transport http.RoundTripper) *ElasticService {
	t.Helper()
	client, err := elasticsearch.NewClient(elasticsearch.Config{
		Transport: transport,
	})
	assert.NoError(t, err)

	logger := zaptest.NewLogger(t).Sugar()

	return NewService(client, logger, "test-index")
}

func elasticOKResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"X-Elastic-Product": []string{"Elasticsearch"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestIndexAnnouncement(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		doc         esDoc.ElasticDoc
		mockFn      func(req *http.Request) (*http.Response, error)
		expectedErr error
	}{
		{
			name: "successful indexing",
			doc: esDoc.ElasticDoc{
				ID:          "test-id",
				Title:       "test-title",
				Description: "test-description",
				Category:    1,
			},
			mockFn: func(req *http.Request) (*http.Response, error) {
				if req.Method == "GET" && req.URL.Path == "/_cluster/health" {
					return &http.Response{
						StatusCode: 200,
						Header:     http.Header{"X-elastic-product": []string{"Elasticsearch"}},
						Body:       io.NopCloser(strings.NewReader(`{"status":"green"}`)),
					}, nil
				}

				return elasticOKResponse(`{}`), nil
			},
			expectedErr: nil,
		},
		{
			name: "elasticsearch error",
			doc: esDoc.ElasticDoc{
				ID:          "test-id",
				Title:       "test-title",
				Description: "test-description",
				Category:    1,
			},
			mockFn: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"error": "server error"}`)),
				}, nil
			},
			expectedErr: myErr.ErrIndexing,
		},
		{
			name: "request error",
			doc: esDoc.ElasticDoc{
				ID:          "test-id",
				Title:       "test-title",
				Description: "test-description",
				Category:    1,
			},
			mockFn: func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("connection error")
			},
			expectedErr: errors.New("connection error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := &mockTransport{
				RoundTripFn: tt.mockFn,
			}

			service := setupTestService(t, transport)
			err := service.IndexAnnouncement(context.Background(), tt.doc)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBulkIndex(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		docs        []esDoc.ElasticDoc
		mockFn      func(req *http.Request) (*http.Response, error)
		expectedErr error
	}{
		{
			name: "successful bulk indexing",
			docs: []esDoc.ElasticDoc{
				{
					ID:          "test-id-1",
					Title:       "test-title-1",
					Description: "test-description-1",
					Category:    1,
				},
				{
					ID:          "test-id-2",
					Title:       "test-title-2",
					Description: "test-description-2",
					Category:    2,
				},
			},
			mockFn: func(req *http.Request) (*http.Response, error) {
				if req.Method == "GET" && req.URL.Path == "/_cluster/health" {

					return &http.Response{
						StatusCode: 200,
						Header:     http.Header{"X-elastic-product": []string{"Elasticsearch"}},
						Body:       io.NopCloser(strings.NewReader(`{"status":"green"}`)),
					}, nil
				}

				body, err := io.ReadAll(req.Body)
				assert.NoError(t, err)
				assert.Contains(t, string(body), `"_id":"test-id-1"`)
				assert.Contains(t, string(body), `"_id":"test-id-2"`)
				return elasticOKResponse(`{}`), nil
			},
			expectedErr: nil,
		},
		{
			name: "empty docs array",
			docs: []esDoc.ElasticDoc{},
			mockFn: func(req *http.Request) (*http.Response, error) {
				t.Error("Request should not be made for empty docs")
				return nil, nil
			},
			expectedErr: nil,
		},
		{
			name: "bulk request error",
			docs: []esDoc.ElasticDoc{
				{
					ID:          "test-id-1",
					Title:       "test-title-1",
					Description: "test-description-1",
					Category:    1,
				},
			},
			mockFn: func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("bulk request failed")
			},
			expectedErr: errors.New("bulk request failed"),
		},
		{
			name: "bulk response error",
			docs: []esDoc.ElasticDoc{
				{
					ID:          "test-id-1",
					Title:       "test-title-1",
					Description: "test-description-1",
					Category:    1,
				},
			},
			mockFn: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"error": "bulk error"}`)),
				}, nil
			},
			expectedErr: myErr.ErrIndexing,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := &mockTransport{
				RoundTripFn: tt.mockFn,
			}

			service := setupTestService(t, transport)
			err := service.BulkIndex(context.Background(), tt.docs)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
