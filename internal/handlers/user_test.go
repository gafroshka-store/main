package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"gafroshka-main/internal/mocks"
	myErr "gafroshka-main/internal/types/errors"
	types "gafroshka-main/internal/types/user"
	"gafroshka-main/internal/user"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-playground/assert"
	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

func TestUserHandler_Info(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	logger := zap.NewNop().Sugar()

	handler := NewUserHandler(logger, mockRepo)

	tests := []struct {
		name           string
		userID         string
		mockBehavior   func()
		expectedStatus int
	}{
		{
			name:   "Success",
			userID: "123",
			mockBehavior: func() {
				mockRepo.EXPECT().
					Info("123").
					Return(&user.User{ID: "123", Name: "Test"}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "User Not Found",
			userID: "notfound",
			mockBehavior: func() {
				mockRepo.EXPECT().
					Info("notfound").
					Return(nil, myErr.ErrNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "Internal Error",
			userID: "123",
			mockBehavior: func() {
				mockRepo.EXPECT().
					Info("123").
					Return(nil, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockBehavior()

			req := httptest.NewRequest(http.MethodGet, "/users/"+tt.userID, nil)
			rr := httptest.NewRecorder()
			r := mux.NewRouter()
			r.HandleFunc("/users/{id}", handler.Info)

			r.ServeHTTP(rr, req)
			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestUserHandler_ChangeProfile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	logger := zap.NewNop().Sugar()

	handler := NewUserHandler(logger, mockRepo)

	tests := []struct {
		name           string
		userID         string
		body           types.ChangeUser
		mockBehavior   func()
		expectedStatus int
	}{
		{
			name:   "Success",
			userID: "123",
			body:   types.ChangeUser{Name: "Updated"},
			mockBehavior: func() {
				mockRepo.EXPECT().
					ChangeProfile("123", types.ChangeUser{Name: "Updated"}).
					Return(&user.User{ID: "123", Name: "Updated"}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "User Not Found",
			userID: "456",
			body:   types.ChangeUser{Email: "notfound@example.com"},
			mockBehavior: func() {
				mockRepo.EXPECT().
					ChangeProfile("456", types.ChangeUser{Email: "notfound@example.com"}).
					Return(nil, myErr.ErrNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Invalid JSON",
			userID:         "789",
			body:           types.ChangeUser{}, // won't be used, we override req body
			mockBehavior:   func() {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockBehavior()

			var reqBody io.Reader
			if tt.name == "Invalid JSON" {
				reqBody = strings.NewReader("{invalid-json}")
			} else {
				jsonBody, _ := json.Marshal(tt.body)
				reqBody = bytes.NewReader(jsonBody)
			}

			req := httptest.NewRequest(http.MethodPut, "/users/"+tt.userID, reqBody)
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			r := mux.NewRouter()
			r.HandleFunc("/users/{id}", handler.ChangeProfile).Methods("PUT")

			r.ServeHTTP(rr, req)
			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}
