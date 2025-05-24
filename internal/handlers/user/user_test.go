package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"gafroshka-main/internal/mocks"
	"gafroshka-main/internal/session"
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

const (
	invalidJSON = "Invalid JSON"
)

func TestUserHandler_Login(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mocks.NewMockUserRepo(ctrl)
	mockSessionRepo := mocks.NewMockSessionRepo(ctrl)
	logger := zap.NewNop().Sugar()
	handler := &UserHandler{
		Logger:         logger,
		UserRepository: mockUserRepo,
		SessionManger:  mockSessionRepo,
	}

	tests := []struct {
		name           string
		body           RequestRegisterForm
		mockBehavior   func()
		expectedStatus int
	}{
		{
			name: "Success",
			body: RequestRegisterForm{
				Email:    "test@example.com",
				Password: "123456",
			},
			mockBehavior: func() {
				mockUserRepo.EXPECT().
					CheckUser("test@example.com", "123456").
					Return(&user.User{ID: "1", Email: "test@example.com"}, nil)

				mockSessionRepo.EXPECT().
					CreateSession(gomock.Any(), gomock.Any(), "1", "test@example.com").
					Return(&session.Session{ID: "sess-123"}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "User Not Found",
			body: RequestRegisterForm{
				Email:    "notfound@example.com",
				Password: "123456",
			},
			mockBehavior: func() {
				mockUserRepo.EXPECT().
					CheckUser("notfound@example.com", "123456").
					Return(nil, myErr.ErrNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "Wrong Password",
			body: RequestRegisterForm{
				Email:    "test@example.com",
				Password: "wrongpass",
			},
			mockBehavior: func() {
				mockUserRepo.EXPECT().
					CheckUser("test@example.com", "wrongpass").
					Return(nil, myErr.ErrBadPassword)
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Internal Error",
			body: RequestRegisterForm{
				Email:    "test@example.com",
				Password: "123456",
			},
			mockBehavior: func() {
				mockUserRepo.EXPECT().
					CheckUser("test@example.com", "123456").
					Return(nil, errors.New("db failure"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           invalidJSON,
			body:           RequestRegisterForm{}, // ignored
			mockBehavior:   func() {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockBehavior()

			var body io.Reader
			if tt.name == invalidJSON {
				body = strings.NewReader("{invalid-json}")
			} else {
				bodyBytes, _ := json.Marshal(tt.body) // nolint:errcheck
				body = bytes.NewReader(bodyBytes)
			}

			req := httptest.NewRequest(http.MethodPost, "/login", body)
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()

			handler.Login(rr, req)
			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestUserHandler_Register(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mocks.NewMockUserRepo(ctrl)
	mockSessionRepo := mocks.NewMockSessionRepo(ctrl)
	logger := zap.NewNop().Sugar()
	handler := &UserHandler{
		Logger:         logger,
		UserRepository: mockUserRepo,
		SessionManger:  mockSessionRepo,
	}

	tests := []struct {
		name           string
		body           types.CreateUser
		mockBehavior   func()
		expectedStatus int
	}{
		{
			name: "Success",
			body: types.CreateUser{
				Email:    "test@example.com",
				Password: "123456",
			},
			mockBehavior: func() {
				mockUserRepo.EXPECT().
					CreateUser(types.CreateUser{
						Email:    "test@example.com",
						Password: "123456",
					}).
					Return(&user.User{ID: "1", Email: "test@example.com"}, nil)

				mockSessionRepo.EXPECT().
					CreateSession(gomock.Any(), gomock.AssignableToTypeOf(httptest.NewRecorder()), "1", "test@example.com").
					Return(&session.Session{ID: "sess-123"}, nil)

			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Invalid Email Format",
			body: types.CreateUser{
				Email:    "invalid-email",
				Password: "123456",
			},
			mockBehavior:   func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "User Already Exists",
			body: types.CreateUser{
				Email:    "exists@example.com",
				Password: "123456",
			},
			mockBehavior: func() {
				mockUserRepo.EXPECT().
					CreateUser(gomock.Any()).
					Return(nil, myErr.ErrAlreadyExists)
			},
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "Internal Error",
			body: types.CreateUser{
				Email:    "test@example.com",
				Password: "123456",
			},
			mockBehavior: func() {
				mockUserRepo.EXPECT().
					CreateUser(gomock.Any()).
					Return(nil, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockBehavior()

			bodyBytes, _ := json.Marshal(tt.body) // nolint:errcheck
			req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()

			handler.Register(rr, req)
			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestUserHandler_Info(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockUserRepo(ctrl)
	logger := zap.NewNop().Sugar()
	mockeSessionRepo := mocks.NewMockSessionRepo(ctrl)

	handler := NewUserHandler(logger, mockRepo, mockeSessionRepo)

	tests := []struct {
		name           string
		userID         string
		mockBehavior   func()
		expectedStatus int
	}{
		{
			name:   "Success",
			userID: "da19a8d6-4b6c-48a8-b888-fdc6b9deef4a",
			mockBehavior: func() {
				mockRepo.EXPECT().
					Info("da19a8d6-4b6c-48a8-b888-fdc6b9deef4a").
					Return(&user.User{ID: "da19a8d6-4b6c-48a8-b888-fdc6b9deef4a", Name: "Test"}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "Invalid ID",
			userID: "da19a8d6",
			mockBehavior: func() {

			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "User Not Found",
			userID: "da19a8d6-4b6c-48a8-b888-fdc6b9deef4a",
			mockBehavior: func() {
				mockRepo.EXPECT().
					Info("da19a8d6-4b6c-48a8-b888-fdc6b9deef4a").
					Return(nil, myErr.ErrNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "Internal Error",
			userID: "da19a8d6-4b6c-48a8-b888-fdc6b9deef4a",
			mockBehavior: func() {
				mockRepo.EXPECT().
					Info("da19a8d6-4b6c-48a8-b888-fdc6b9deef4a").
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
	mockeSessionRepo := mocks.NewMockSessionRepo(ctrl)

	handler := NewUserHandler(logger, mockRepo, mockeSessionRepo)

	tests := []struct {
		name           string
		userID         string
		body           types.ChangeUser
		mockBehavior   func()
		expectedStatus int
	}{
		{
			name:   "Success",
			userID: "da19a8d6-4b6c-48a8-b888-fdc6b9deef4a",
			body:   types.ChangeUser{Name: "Updated"},
			mockBehavior: func() {
				mockRepo.EXPECT().
					ChangeProfile("da19a8d6-4b6c-48a8-b888-fdc6b9deef4a", types.ChangeUser{Name: "Updated"}).
					Return(&user.User{ID: "da19a8d6-4b6c-48a8-b888-fdc6b9deef4a", Name: "Updated"}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "User Not Found",
			userID: "da19a8d6-4b6c-48a8-b888-fdc6b9deef4a",
			body:   types.ChangeUser{Email: "notfound@example.com"},
			mockBehavior: func() {
				mockRepo.EXPECT().
					ChangeProfile("da19a8d6-4b6c-48a8-b888-fdc6b9deef4a", types.ChangeUser{Email: "notfound@example.com"}).
					Return(nil, myErr.ErrNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "Invalid ID",
			userID: "invalid-id",
			mockBehavior: func() {

			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           invalidJSON,
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
			if tt.name == invalidJSON {
				reqBody = strings.NewReader("{invalid-json}")
			} else {
				jsonBody, err := json.Marshal(tt.body)
				assert.Equal(t, nil, err)
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
