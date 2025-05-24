package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"gafroshka-main/internal/mocks"
	typesAnn "gafroshka-main/internal/types/announcement"
	myErr "gafroshka-main/internal/types/errors"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.uber.org/zap/zaptest"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestShoppingCartHandler_AddToShoppingCart(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCartRepo := mocks.NewMockShoppingCardRepo(ctrl)
	mockAnnRepo := mocks.NewMockAnnouncementRepo(ctrl)
	logger := zaptest.NewLogger(t).Sugar()
	handler := NewShoppingCartHandler(logger, mockCartRepo, mockAnnRepo)

	validUserID := uuid.New().String()
	validAnnID := uuid.New().String()
	tests := []struct {
		name           string
		userID         string
		annID          string
		mockBehavior   func()
		expectedStatus int
	}{
		{
			name:   "success",
			userID: validUserID,
			annID:  validAnnID,
			mockBehavior: func() {
				mockCartRepo.EXPECT().AddAnnouncement(validUserID, validAnnID).Return(nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "bad userID",
			userID:         "invalid",
			annID:          validAnnID,
			mockBehavior:   func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "repo error",
			userID: validUserID,
			annID:  validAnnID,
			mockBehavior: func() {
				mockCartRepo.EXPECT().AddAnnouncement(validUserID, validAnnID).Return(errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.mockBehavior()
			url := fmt.Sprintf("/cart/%s/item/%s", tc.userID, tc.annID)
			req := httptest.NewRequest(http.MethodPost, url, nil)
			req = mux.SetURLVars(req, map[string]string{
				"userID": tc.userID,
				"annID":  tc.annID,
			})
			w := httptest.NewRecorder()

			handler.AddToShoppingCart(w, req)

			resp := w.Result()
			if resp.StatusCode != tc.expectedStatus {
				t.Errorf("expected %d, got %d", tc.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func TestShoppingCartHandler_DeleteFromShoppingCart(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCartRepo := mocks.NewMockShoppingCartRepo(ctrl)
	mockAnnRepo := mocks.NewMockAnnouncementRepo(ctrl)
	logger := zaptest.NewLogger(t).Sugar()
	handler := NewShoppingCartHandler(logger, mockCartRepo, mockAnnRepo)

	validUserID := uuid.New().String()
	validAnnID := uuid.New().String()

	tests := []struct {
		name           string
		userID         string
		annID          string
		mockBehavior   func()
		expectedStatus int
	}{
		{
			name:   "success",
			userID: validUserID,
			annID:  validAnnID,
			mockBehavior: func() {
				mockCartRepo.EXPECT().DeleteAnnouncement(validUserID, validAnnID).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "not found",
			userID: validUserID,
			annID:  validAnnID,
			mockBehavior: func() {
				mockCartRepo.EXPECT().DeleteAnnouncement(validUserID, validAnnID).Return(myErr.ErrNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid annID",
			userID:         validUserID,
			annID:          "bad-id",
			mockBehavior:   func() {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.mockBehavior()

			url := fmt.Sprintf("/cart/%s/item/%s", tc.userID, tc.annID)
			req := httptest.NewRequest(http.MethodDelete, url, nil)
			req = mux.SetURLVars(req, map[string]string{
				"userID": tc.userID,
				"annID":  tc.annID,
			})
			w := httptest.NewRecorder()

			handler.DeleteFromShoppingCart(w, req)

			resp := w.Result()
			if resp.StatusCode != tc.expectedStatus {
				t.Errorf("expected %d, got %d", tc.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func TestShoppingCartHandler_GetCart(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCartRepo := mocks.NewMockShoppingCartRepo(ctrl)
	mockAnnRepo := mocks.NewMockAnnouncementRepo(ctrl)
	logger := zaptest.NewLogger(t).Sugar()
	handler := NewShoppingCartHandler(logger, mockCartRepo, mockAnnRepo)

	validUserID := uuid.New().String()
	announcementIDs := []string{uuid.New().String(), uuid.New().String()}
	infos := []typesAnn.InfoForSC{
		{ID: announcementIDs[0], Name: "Item 1"},
		{ID: announcementIDs[1], Name: "Item 2"},
	}

	tests := []struct {
		name           string
		userID         string
		mockBehavior   func()
		expectedStatus int
		expectEmpty    bool
	}{
		{
			name:   "success",
			userID: validUserID,
			mockBehavior: func() {
				mockCartRepo.EXPECT().GetByUserID(validUserID).Return(announcementIDs, nil)
				mockAnnRepo.EXPECT().GetInfoForShoppingCart(announcementIDs).Return(infos, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "empty cart",
			userID: validUserID,
			mockBehavior: func() {
				mockCartRepo.EXPECT().GetByUserID(validUserID).Return(nil, myErr.ErrNotFound)
			},
			expectedStatus: http.StatusNoContent,
			expectEmpty:    true,
		},
		{
			name:           "bad uuid",
			userID:         "not-a-uuid",
			mockBehavior:   func() {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.mockBehavior()

			url := fmt.Sprintf("/cart/%s", tc.userID)
			req := httptest.NewRequest(http.MethodGet, url, nil)
			req = mux.SetURLVars(req, map[string]string{
				"userID": tc.userID,
			})
			w := httptest.NewRecorder()

			handler.GetCart(w, req)

			resp := w.Result()
			if resp.StatusCode != tc.expectedStatus {
				t.Errorf("expected %d, got %d", tc.expectedStatus, resp.StatusCode)
			}

			if !tc.expectEmpty && tc.expectedStatus == http.StatusOK {
				var got []typesAnn.InfoForSC
				err := json.NewDecoder(resp.Body).Decode(&got)
				if err != nil {
					t.Errorf("failed to decode response: %v", err)
				}
				if len(got) != len(infos) {
					t.Errorf("expected %d items, got %d", len(infos), len(got))
				}
			}
		})
	}
}
