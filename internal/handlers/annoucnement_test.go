package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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

	tests := []struct {
		name       string
		input      interface{}
		mockSetup  func()
		statusCode int
	}{
		{
			name: "Success",
			input: typesAnn.CreateAnnouncement{
				Name:         "Test",
				Description:  "Desc",
				UserSellerID: "u1",
				Price:        100,
				Category:     1,
			},
			mockSetup: func() {
				mockRepo.EXPECT().Create(
					typesAnn.CreateAnnouncement{
						Name:         "Test",
						Description:  "Desc",
						UserSellerID: "u1",
						Price:        100,
						Category:     1,
					},
				).Return(&announcement.Announcement{ID: "a1"}, nil)
			},
			statusCode: http.StatusCreated,
		},
		{
			name:       "InvalidJSON",
			input:      "{bad}",
			mockSetup:  func() {},
			statusCode: http.StatusBadRequest,
		},
		{
			name: "InternalError",
			input: typesAnn.CreateAnnouncement{
				Name:         "Test",
				Description:  "Desc",
				UserSellerID: "u1",
				Price:        100,
				Category:     1,
			},
			mockSetup: func() {
				mockRepo.EXPECT().Create(gomock.Any()).Return(nil, errors.New("db error"))
			},
			statusCode: http.StatusInternalServerError,
		},
	}

	r := mux.NewRouter()
	r.HandleFunc("/announcement", handler.Create).Methods("POST")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			body, _ := json.Marshal(tt.input)
			req := httptest.NewRequest(http.MethodPost, "/announcement", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)
			assert.Equal(t, tt.statusCode, rr.Code)
		})
	}
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
		mockSetup  func()
		statusCode int
	}{
		{
			name: "Success",
			id:   "a1",
			mockSetup: func() {
				mockRepo.EXPECT().GetByID("a1").Return(&announcement.Announcement{ID: "a1"}, nil)
			},
			statusCode: http.StatusOK,
		},
		{
			name:       "InvalidID",
			id:         "",
			mockSetup:  func() {},
			statusCode: http.StatusNotFound,
		},
		{
			name: "NotFound",
			id:   "na",
			mockSetup: func() {
				mockRepo.EXPECT().GetByID("na").Return(nil, myErr.ErrNotFound)
			},
			statusCode: http.StatusNotFound,
		},
		{
			name: "InternalError",
			id:   "a1",
			mockSetup: func() {
				mockRepo.EXPECT().GetByID("a1").Return(nil, errors.New("db"))
			},
			statusCode: http.StatusInternalServerError,
		},
	}

	r := mux.NewRouter()
	r.HandleFunc("/announcement/{id}", h.GetByID)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

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
	r.HandleFunc("/announcements/top", h.GetTopN).Methods("POST")

	tests := []struct {
		name       string
		input      interface{}
		mockSetup  func()
		statusCode int
	}{
		{
			name:  "Success",
			input: map[string]int{"limit": 5},
			mockSetup: func() {
				mockRepo.EXPECT().GetTopN(5).Return([]announcement.Announcement{}, nil)
			},
			statusCode: http.StatusOK,
		},
		{
			name:       "InvalidLimitZero",
			input:      map[string]int{"limit": 0},
			mockSetup:  func() {},
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "InvalidLimitNegative",
			input:      map[string]int{"limit": -5},
			mockSetup:  func() {},
			statusCode: http.StatusBadRequest,
		},
		{
			name:  "ValidLimit",
			input: map[string]int{"limit": 5},
			mockSetup: func() {
				mockRepo.EXPECT().GetTopN(5).
					Return([]announcement.Announcement{}, nil)
			},
			statusCode: http.StatusOK,
		},
		{
			name:       "InvalidLimit",
			input:      map[string]int{"limit": -1},
			mockSetup:  func() {},
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "InvalidJSON",
			input:      "{bad}",
			mockSetup:  func() {},
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "MissingLimitField",
			input:      map[string]int{},
			mockSetup:  func() {},
			statusCode: http.StatusBadRequest,
		},
		{
			name:  "InternalError",
			input: map[string]int{"limit": 5},
			mockSetup: func() {
				mockRepo.EXPECT().GetTopN(5).Return(nil, errors.New("db error"))
			},
			statusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			var body bytes.Buffer
			switch v := tt.input.(type) {
			case string:
				body.WriteString(v)
			default:
				json.NewEncoder(&body).Encode(v)
			}

			req := httptest.NewRequest(http.MethodPost, "/announcements/top", &body)
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)
			assert.Equal(t, tt.statusCode, rr.Code)
		})
	}
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

	tests := []struct {
		name       string
		id         string
		body       interface{}
		mockSetup  func()
		statusCode int
	}{
		{
			name: "InvalidJSON",
			id:   "a1",
			body: "{bad}",
			mockSetup: func() {
				mockRepo.EXPECT().UpdateRating(gomock.Any(), gomock.Any()).Times(0)
			},
			statusCode: http.StatusBadRequest,
		},
		{
			name: "NotFound",
			id:   "a1",
			body: map[string]int{"rating": 5},
			mockSetup: func() {
				mockRepo.EXPECT().UpdateRating("a1", 5).
					Return(nil, myErr.ErrNotFound)
			},
			statusCode: http.StatusNotFound,
		},
		{
			name: "Success",
			id:   "a1",
			body: map[string]int{"rating": 4},
			mockSetup: func() {
				mockRepo.EXPECT().UpdateRating("a1", 4).
					Return(&announcement.Announcement{ID: "a1"}, nil)
			},
			statusCode: http.StatusOK,
		},
		{
			name: "MissingID",
			id:   "",
			body: map[string]int{"rating": 5},
			mockSetup: func() {
				mockRepo.EXPECT().UpdateRating(gomock.Any(), gomock.Any()).Times(0)
			},
			statusCode: http.StatusMovedPermanently,
		},
		{
			name: "NegativeRating",
			id:   "a1",
			body: map[string]int{"rating": -5},
			mockSetup: func() {
				mockRepo.EXPECT().UpdateRating(gomock.Any(), gomock.Any()).Times(0)
			},
			statusCode: http.StatusBadRequest,
		},
		{
			name: "RatingTooHigh",
			id:   "a1",
			body: map[string]int{"rating": 6},
			mockSetup: func() {
				mockRepo.EXPECT().UpdateRating(gomock.Any(), gomock.Any()).Times(0)
			},
			statusCode: http.StatusBadRequest,
		},
	}

	r := mux.NewRouter()
	r.HandleFunc("/announcement/{id}/rating", h.UpdateRating)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			var body bytes.Buffer
			switch v := tt.body.(type) {
			case string:
				body.WriteString(v)
			default:
				json.NewEncoder(&body).Encode(v)
			}

			req := httptest.NewRequest(
				http.MethodPost,
				"/announcement/"+tt.id+"/rating",
				&body,
			)
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			assert.Equal(t, tt.statusCode, rr.Code)
		})
	}
}
