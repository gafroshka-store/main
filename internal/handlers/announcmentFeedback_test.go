package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"

	hand "gafroshka-main/internal/handlers"
	"gafroshka-main/internal/mocks"
	announcmentfeedback "gafroshka-main/internal/types/announcmentFeedback"
	myErr "gafroshka-main/internal/types/errors"

	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

func setupHandler(t *testing.T) (*hand.AnnouncementFeedbackHandler, *mocks.MockFeedbackRepo, func()) {
	ctrl := gomock.NewController(t)
	mockRepo := mocks.NewMockFeedbackRepo(ctrl)
	logger := zaptest.NewLogger(t).Sugar()
	handler := hand.NewAnnouncementFeedbackHandler(logger, mockRepo)
	return handler, mockRepo, func() { ctrl.Finish() }
}

func TestCreate_Success(t *testing.T) {
	h, mockRepo, teardown := setupHandler(t)
	defer teardown()

	input := announcmentfeedback.Feedback{
		AnnouncementID: "ann1",
		UserWriterID:   "user1",
		Comment:        "Nice!",
		Rating:         5,
	}
	body, _ := json.Marshal(input)

	mockRepo.EXPECT().Create(input).Return(input, nil)

	req := httptest.NewRequest(http.MethodPost, "/feedback", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.Create(w, req)

	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusCreated, res.StatusCode)
	var got announcmentfeedback.Feedback
	err := json.NewDecoder(res.Body).Decode(&got)
	assert.NoError(t, err)
	assert.Equal(t, input, got)
}

func TestCreate_InvalidJSON(t *testing.T) {
	h, _, teardown := setupHandler(t)
	defer teardown()

	req := httptest.NewRequest(http.MethodPost, "/feedback", bytes.NewReader([]byte("{invalid json")))
	w := httptest.NewRecorder()
	h.Create(w, req)

	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusBadRequest, res.StatusCode)
}

func TestDelete_Success(t *testing.T) {
	h, mockRepo, teardown := setupHandler(t)
	defer teardown()

	mockRepo.EXPECT().Delete("f1").Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/feedback/f1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "f1"})
	w := httptest.NewRecorder()
	h.Delete(w, req)

	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusNoContent, res.StatusCode)
}

func TestDelete_MissingID(t *testing.T) {
	h, _, teardown := setupHandler(t)
	defer teardown()

	req := httptest.NewRequest(http.MethodDelete, "/feedback/", nil)
	req = mux.SetURLVars(req, map[string]string{"id": ""})
	w := httptest.NewRecorder()
	h.Delete(w, req)

	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusBadRequest, res.StatusCode)
}

func TestDelete_NotFound(t *testing.T) {
	h, mockRepo, teardown := setupHandler(t)
	defer teardown()

	mockRepo.EXPECT().Delete("f2").Return(myErr.ErrNotFoundFeedback)

	req := httptest.NewRequest(http.MethodDelete, "/feedback/f2", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "f2"})
	w := httptest.NewRecorder()
	h.Delete(w, req)

	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusNotFound, res.StatusCode)
}

func TestDelete_DBError(t *testing.T) {
	h, mockRepo, teardown := setupHandler(t)
	defer teardown()

	mockRepo.EXPECT().Delete("f3").Return(errors.New("db error"))

	req := httptest.NewRequest(http.MethodDelete, "/feedback/f3", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "f3"})
	w := httptest.NewRecorder()
	h.Delete(w, req)

	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
}

func TestGetByAnnouncementID_Success(t *testing.T) {
	h, mockRepo, teardown := setupHandler(t)
	defer teardown()

	expected := []announcmentfeedback.Feedback{
		{ID: "1", AnnouncementID: "a1", UserWriterID: "u1", Comment: "Good", Rating: 4},
	}
	mockRepo.EXPECT().GetByAnnouncementID("a1").Return(expected, nil)

	req := httptest.NewRequest(http.MethodGet, "/feedback/announcement/a1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "a1"})
	w := httptest.NewRecorder()
	h.GetByAnnouncementID(w, req)

	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusOK, res.StatusCode)
	var got []announcmentfeedback.Feedback
	err := json.NewDecoder(res.Body).Decode(&got)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestGetByAnnouncementID_MissingID(t *testing.T) {
	h, _, teardown := setupHandler(t)
	defer teardown()

	req := httptest.NewRequest(http.MethodGet, "/feedback/announcement/", nil)
	req = mux.SetURLVars(req, map[string]string{"id": ""})
	w := httptest.NewRecorder()
	h.GetByAnnouncementID(w, req)

	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusBadRequest, res.StatusCode)
}

func TestGetByAnnouncementID_DBError(t *testing.T) {
	h, mockRepo, teardown := setupHandler(t)
	defer teardown()

	mockRepo.EXPECT().GetByAnnouncementID("a2").Return(nil, errors.New("db error"))

	req := httptest.NewRequest(http.MethodGet, "/feedback/announcement/a2", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "a2"})
	w := httptest.NewRecorder()
	h.GetByAnnouncementID(w, req)

	res := w.Result()
	defer res.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
}
