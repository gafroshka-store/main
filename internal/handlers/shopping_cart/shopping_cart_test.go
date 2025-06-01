package handlers

import (
	"bytes"
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
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCartRepo := mocks.NewMockShoppingCardRepo(ctrl)
	mockAnnRepo := mocks.NewMockAnnouncementRepo(ctrl)
	logger := zaptest.NewLogger(t).Sugar()
	handler := NewShoppingCartHandler(logger, mockCartRepo, mockAnnRepo, nil)

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
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
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
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCartRepo := mocks.NewMockShoppingCartRepo(ctrl)
	mockAnnRepo := mocks.NewMockAnnouncementRepo(ctrl)
	logger := zaptest.NewLogger(t).Sugar()
	handler := NewShoppingCartHandler(logger, mockCartRepo, mockAnnRepo, nil)

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
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
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
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCartRepo := mocks.NewMockShoppingCartRepo(ctrl)
	mockAnnRepo := mocks.NewMockAnnouncementRepo(ctrl)
	logger := zaptest.NewLogger(t).Sugar()
	handler := NewShoppingCartHandler(logger, mockCartRepo, mockAnnRepo, nil)

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
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
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

func TestShoppingCartHandler_PurchaseFromCart(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCartRepo := mocks.NewMockShoppingCartRepo(ctrl)
	mockAnnRepo := mocks.NewMockAnnouncementRepo(ctrl)
	mockUserRepo := mocks.NewMockUserRepo(ctrl)
	logger := zaptest.NewLogger(t).Sugar()

	handler := NewShoppingCartHandler(logger, mockCartRepo, mockAnnRepo, mockUserRepo)

	validUserID := uuid.New().String()
	itemID1 := uuid.New().String()
	itemID2 := uuid.New().String()

	requestedIDs := []string{itemID1, itemID2}
	cartItems := []string{itemID1, itemID2, uuid.New().String()}
	infos := []typesAnn.InfoForSC{
		{ID: itemID1, Price: 1000, Discount: 10},
		{ID: itemID2, Price: 2000, Discount: 10},
	}
	total := int64(2700)

	tests := []struct {
		name           string
		userID         string
		requestedIDs   []string
		mockBehavior   func()
		expectedStatus int
	}{
		{
			name:         "success",
			userID:       validUserID,
			requestedIDs: requestedIDs,
			mockBehavior: func() {
				mockCartRepo.EXPECT().GetByUserID(validUserID).Return(cartItems, nil)
				mockAnnRepo.EXPECT().GetInfoForShoppingCart(requestedIDs).Return(infos, nil)
				mockUserRepo.EXPECT().GetBalanceByUserID(validUserID).Return(int64(5000), nil)
				mockUserRepo.EXPECT().TopUpBalance(validUserID, -total).Return(int64(2000), nil)
				mockCartRepo.EXPECT().DeleteAnnouncement(validUserID, itemID1).Return(nil)
				mockCartRepo.EXPECT().DeleteAnnouncement(validUserID, itemID2).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid UUID",
			userID:         "bad-id",
			requestedIDs:   requestedIDs,
			mockBehavior:   func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "item not in cart",
			userID:       validUserID,
			requestedIDs: []string{uuid.New().String()},
			mockBehavior: func() {
				mockCartRepo.EXPECT().GetByUserID(validUserID).Return(cartItems, nil)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "insufficient funds",
			userID:       validUserID,
			requestedIDs: requestedIDs,
			mockBehavior: func() {
				mockCartRepo.EXPECT().GetByUserID(validUserID).Return(cartItems, nil)
				mockAnnRepo.EXPECT().GetInfoForShoppingCart(requestedIDs).Return(infos, nil)
				mockUserRepo.EXPECT().GetBalanceByUserID(validUserID).Return(int64(1000), nil)
			},
			expectedStatus: http.StatusPaymentRequired,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.mockBehavior()

			body, _ := json.Marshal(tc.requestedIDs)
			url := fmt.Sprintf("/cart/%s/purchase", tc.userID)
			req := httptest.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
			req = mux.SetURLVars(req, map[string]string{
				"userID": tc.userID,
			})
			w := httptest.NewRecorder()

			handler.PurchaseFromCart(w, req)

			resp := w.Result()
			if resp.StatusCode != tc.expectedStatus {
				t.Errorf("expected status %d, got %d", tc.expectedStatus, resp.StatusCode)
			}

			if tc.expectedStatus == http.StatusOK {
				var respBody map[string]interface{}
				if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
					t.Errorf("failed to decode response body: %v", err)
				}
				if respBody["status"] != "success" {
					t.Errorf("expected success status, got %v", respBody["status"])
				}
				res, ok := respBody["total"].(float64)
				if !ok {
					t.Errorf("expected total to be float64, got %v", respBody["total"])
				}
				if int64(res) != total {
					t.Errorf("expected total %d, got %v", total, respBody["total"])
				}
			}
		})
	}
}
