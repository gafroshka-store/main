package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-playground/assert"
	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"gafroshka-main/internal/announcement"
	"gafroshka-main/internal/mocks"
	typesAnn "gafroshka-main/internal/types/announcement"
	myErr "gafroshka-main/internal/types/errors"
)

func TestAnnouncementHandler_Create(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockAnnouncementRepo(ctrl)
	logger := zap.NewNop().Sugar()
	handler := NewAnnouncementHandler(logger, mockRepo)

	input := typesAnn.CreateAnnouncement{Name: "Name", Description: "Desc", UserSellerID: "u1", Price: 100, Category: 1, Discount: 0}
	ann := &announcement.Announcement{ID: "a1", Name: input.Name}

	mockRepo.EXPECT().Create(input).Return(ann, nil)

	body, _ := json.Marshal(input)
	req := httptest.NewRequest(http.MethodPost, "/announcement", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h := mux.NewRouter()
	h.HandleFunc("/announcement", handler.Create).Methods("POST")
	h.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusCreated, rr.Code)
}

func TestAnnouncementHandler_GetByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockAnnouncementRepo(ctrl)
	logger := zap.NewNop().Sugar()
	h := NewAnnouncementHandler(logger, mockRepo)

	tests := []struct {
		name       string
		id         string
		behavior   func()
		statusCode int
	}{
		{
			name: "Success",
			id:   "a1",
			behavior: func() {
				mockRepo.EXPECT().GetByID("a1").Return(&announcement.Announcement{ID: "a1"}, nil)
			},
			statusCode: http.StatusOK,
		},
		{
			name: "NotFound",
			id:   "na",
			behavior: func() {
				mockRepo.EXPECT().GetByID("na").Return(nil, myErr.ErrNotFound)
			},
			statusCode: http.StatusNotFound,
		},
		{
			name: "InternalError",
			id:   "a1",
			behavior: func() {
				mockRepo.EXPECT().GetByID("a1").Return(nil, errors.New("db"))
			},
			statusCode: http.StatusInternalServerError,
		},
	}

	r := mux.NewRouter()
	r.HandleFunc("/announcement/{id}", h.GetByID)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.behavior()

			req := httptest.NewRequest(http.MethodGet, "/announcement/"+tt.id, nil)
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)
			assert.Equal(t, tt.statusCode, rr.Code)
		})
	}
}

func TestAnnouncementHandler_GetTopN(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockAnnouncementRepo(ctrl)
	logger := zap.NewNop().Sugar()
	h := NewAnnouncementHandler(logger, mockRepo)

	r := mux.NewRouter()
	r.HandleFunc("/announcements/top/{limit}", h.GetTopN)

	// Success
	mockRepo.EXPECT().GetTopN(5).Return([]announcement.Announcement{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/announcements/top/5", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// Invalid limit
	req = httptest.NewRequest(http.MethodGet, "/announcements/top/zero", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestAnnouncementHandler_Search(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockAnnouncementRepo(ctrl)
	logger := zap.NewNop().Sugar()
	h := NewAnnouncementHandler(logger, mockRepo)

	r := mux.NewRouter()
	r.HandleFunc("/announcements/search", h.Search)

	// Missing query
	req := httptest.NewRequest(http.MethodGet, "/announcements/search", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)

	// Success
	mockRepo.EXPECT().Search("foo").Return([]announcement.Announcement{}, nil)
	req = httptest.NewRequest(http.MethodGet, "/announcements/search?q=foo", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAnnouncementHandler_UpdateRating(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockAnnouncementRepo(ctrl)
	logger := zap.NewNop().Sugar()
	h := NewAnnouncementHandler(logger, mockRepo)

	r := mux.NewRouter()
	r.HandleFunc("/announcement/{id}/rating", h.UpdateRating)

	// Invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/announcement/a1/rating", strings.NewReader("{bad}"))
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)

	// NotFound
	mockRepo.EXPECT().UpdateRating("a1", 5).Return(nil, myErr.ErrNotFound)
	req = httptest.NewRequest(http.MethodPost, "/announcement/a1/rating", strings.NewReader("{"+`"rating":5`+"}"))
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)

	// Success
	mockRepo.EXPECT().UpdateRating("a1", 4).Return(&announcement.Announcement{ID: "a1"}, nil)
	req = httptest.NewRequest(http.MethodPost, "/announcement/a1/rating", strings.NewReader("{"+`"rating":4`+"}"))
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}
