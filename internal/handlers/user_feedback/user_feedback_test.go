package handlers

import (
	"bytes"
	"encoding/json"
	"gafroshka-main/internal/mocks"
	myErr "gafroshka-main/internal/types/errors"
	types "gafroshka-main/internal/types/user_feedback"
	userFeedback "gafroshka-main/internal/user_feedback"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"go.uber.org/zap/zaptest"

	"github.com/golang/mock/gomock"
)

func TestUserFeedbackHandler_Create(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserFeedbackRepo(ctrl)
	logger := zaptest.NewLogger(t).Sugar()
	handler := NewUserFeedbackHandler(logger, mockRepo)

	tests := []struct {
		name           string
		payload        userFeedback.UserFeedback
		mockBehavior   func()
		expectedStatus int
	}{
		{
			name: "success",
			payload: userFeedback.UserFeedback{
				UserRecipientID: "user1",
				UserWriterID:    "user2",
				Comment:         "Great!",
				Rating:          5,
			},
			mockBehavior: func() {
				mockRepo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(&userFeedback.UserFeedback{
						ID:              "123",
						UserRecipientID: "user1",
						UserWriterID:    "user2",
						Comment:         "Great!",
						Rating:          5,
					}, nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "invalid json",
			payload: userFeedback.UserFeedback{
				UserRecipientID: "",
			},
			mockBehavior:   func() {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.mockBehavior()

			var body *bytes.Buffer
			if tc.name == "invalid json" {
				body = bytes.NewBuffer([]byte("{invalid_json}"))
			} else {
				data, err := json.Marshal(tc.payload)
				if err != nil {
					t.Fatal(err)
				}

				body = bytes.NewBuffer(data)
			}

			req := httptest.NewRequest(http.MethodPost, "/user/feedback", body)
			w := httptest.NewRecorder()

			handler.Create(w, req)

			resp := w.Result()
			if resp.StatusCode != tc.expectedStatus {
				t.Errorf("expected %d, got %d", tc.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func TestUserFeedbackHandler_GetByUserID(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserFeedbackRepo(ctrl)
	logger := zaptest.NewLogger(t).Sugar()
	handler := NewUserFeedbackHandler(logger, mockRepo)

	tests := []struct {
		name           string
		userID         string
		mockBehavior   func()
		expectedStatus int
	}{
		{
			name:   "success",
			userID: "user1",
			mockBehavior: func() {
				mockRepo.EXPECT().
					GetByUserID(gomock.Any(), "user1").
					Return([]*userFeedback.UserFeedback{}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "missing id",
			userID: "",
			mockBehavior: func() {
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.mockBehavior()

			req := httptest.NewRequest(http.MethodGet, "/user/feedback/user/"+tc.userID, nil)
			req = mux.SetURLVars(req, map[string]string{"id": tc.userID})
			w := httptest.NewRecorder()

			handler.GetByUserID(w, req)

			resp := w.Result()
			if resp.StatusCode != tc.expectedStatus {
				t.Errorf("expected %d, got %d", tc.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func TestUserFeedbackHandler_Update(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserFeedbackRepo(ctrl)
	logger := zaptest.NewLogger(t).Sugar()
	handler := NewUserFeedbackHandler(logger, mockRepo)

	tests := []struct {
		name           string
		feedbackID     string
		body           types.UpdateUserFeedback
		mockBehavior   func()
		expectedStatus int
	}{
		{
			name:       "success",
			feedbackID: "id1",
			body: types.UpdateUserFeedback{
				Comment: "Updated!",
				Rating:  4,
			},
			mockBehavior: func() {
				mockRepo.EXPECT().
					Update(gomock.Any(), "id1", types.UpdateUserFeedback{
						Comment: "Updated!",
						Rating:  4,
					}).
					Return(&userFeedback.UserFeedback{
						ID:              "id1",
						UserRecipientID: "userX",
						UserWriterID:    "userY",
						Comment:         "Updated!",
						Rating:          4,
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:       "not found",
			feedbackID: "id2",
			body: types.UpdateUserFeedback{
				Comment: "Nope",
			},
			mockBehavior: func() {
				mockRepo.EXPECT().
					Update(gomock.Any(), "id2", types.UpdateUserFeedback{
						Comment: "Nope",
						Rating:  0,
					}).
					Return(nil, myErr.ErrNotFoundUserFeedback)
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.mockBehavior()

			bodyBytes, err := json.Marshal(tc.body)
			if err != nil {
				t.Fatal(err)
			}

			req := httptest.NewRequest(http.MethodPut, "/user/feedback/"+tc.feedbackID, bytes.NewReader(bodyBytes))
			req = mux.SetURLVars(req, map[string]string{"id": tc.feedbackID})
			w := httptest.NewRecorder()

			handler.Update(w, req)

			resp := w.Result()
			if resp.StatusCode != tc.expectedStatus {
				t.Errorf("expected %d, got %d", tc.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func TestUserFeedbackHandler_Delete(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserFeedbackRepo(ctrl)
	logger := zaptest.NewLogger(t).Sugar()
	handler := NewUserFeedbackHandler(logger, mockRepo)

	tests := []struct {
		name           string
		feedbackID     string
		mockBehavior   func()
		expectedStatus int
	}{
		{
			name:       "success",
			feedbackID: "id1",
			mockBehavior: func() {
				mockRepo.EXPECT().
					Delete(gomock.Any(), "id1").
					Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:       "not found",
			feedbackID: "id2",
			mockBehavior: func() {
				mockRepo.EXPECT().
					Delete(gomock.Any(), "id2").
					Return(myErr.ErrNotFoundUserFeedback)
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.mockBehavior()

			req := httptest.NewRequest(http.MethodDelete, "/user/feedback/"+tc.feedbackID, nil)
			req = mux.SetURLVars(req, map[string]string{"id": tc.feedbackID})
			w := httptest.NewRecorder()

			handler.Delete(w, req)

			resp := w.Result()
			if resp.StatusCode != tc.expectedStatus {
				t.Errorf("expected %d, got %d", tc.expectedStatus, resp.StatusCode)
			}
		})
	}
}
