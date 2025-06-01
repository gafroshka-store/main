package handlers

import (
	"encoding/json"
	"errors"
	"gafroshka-main/internal/user"
	"math"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"gafroshka-main/internal/announcement"
	"gafroshka-main/internal/shopping_cart"
	typesAnn "gafroshka-main/internal/types/announcement"
	myErr "gafroshka-main/internal/types/errors"
)

// ShoppingCartHandler ручки для репозитория корзины
type ShoppingCartHandler struct {
	Logger           *zap.SugaredLogger
	CartRepo         shopping_cart.ShoppingCartRepo
	AnnouncementRepo announcement.AnnouncementRepo
	UserRepo         user.UserRepo
}

// NewShoppingCartHandler конструктор
func NewShoppingCartHandler(
	log *zap.SugaredLogger,
	cr shopping_cart.ShoppingCartRepo,
	ar announcement.AnnouncementRepo,
	ur user.UserRepo,
) *ShoppingCartHandler {
	return &ShoppingCartHandler{
		Logger: log, CartRepo: cr,
		AnnouncementRepo: ar,
		UserRepo:         ur,
	}
}

// AddToShoppingCart - POST /cart/{userID}/item/{annID}
func (h *ShoppingCartHandler) AddToShoppingCart(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userID"]
	annID := vars["annID"]

	if _, err := uuid.Parse(userID); err != nil {
		myErr.SendErrorTo(w, myErr.ErrBadID, http.StatusBadRequest, h.Logger)
		return
	}
	if _, err := uuid.Parse(annID); err != nil {
		myErr.SendErrorTo(w, myErr.ErrBadID, http.StatusBadRequest, h.Logger)
		return
	}

	err := h.CartRepo.AddAnnouncement(userID, annID)
	if err != nil {
		myErr.SendErrorTo(w, err, http.StatusInternalServerError, h.Logger)
		return
	}

	w.WriteHeader(http.StatusCreated)
	h.Logger.Infof("added announcement %s to user %s shopping cart", annID, userID)
}

// DeleteFromShoppingCart - DELETE /cart/{userID}/item/{annID}
func (h *ShoppingCartHandler) DeleteFromShoppingCart(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userID"]
	annID := vars["annID"]

	if _, err := uuid.Parse(userID); err != nil {
		myErr.SendErrorTo(w, myErr.ErrBadID, http.StatusBadRequest, h.Logger)
		return
	}
	if _, err := uuid.Parse(annID); err != nil {
		myErr.SendErrorTo(w, myErr.ErrBadID, http.StatusBadRequest, h.Logger)
		return
	}

	err := h.CartRepo.DeleteAnnouncement(userID, annID)
	if err != nil {
		if errors.Is(err, myErr.ErrNotFound) {
			myErr.SendErrorTo(w, err, http.StatusNotFound, h.Logger)
			return
		}
		myErr.SendErrorTo(w, err, http.StatusInternalServerError, h.Logger)
		return
	}

	w.WriteHeader(http.StatusOK)
	h.Logger.Infof("deleted announcement %s from user %s shopping cart", annID, userID)
}

// GetCart - GET /cart/{userID}
func (h *ShoppingCartHandler) GetCart(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userID"]

	if _, err := uuid.Parse(userID); err != nil {
		myErr.SendErrorTo(w, myErr.ErrBadID, http.StatusBadRequest, h.Logger)
		return
	}

	ids, err := h.CartRepo.GetByUserID(userID)
	if err != nil {
		if errors.Is(err, myErr.ErrNotFound) {
			// Если пусто, то возвращаем ноу контент
			resp := []typesAnn.InfoForSC{}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNoContent)
			err = json.NewEncoder(w).Encode(resp)
			if err != nil {
				h.Logger.Warnw("error writing response", "err", err)
				return
			}
			return
		}
		myErr.SendErrorTo(w, err, http.StatusInternalServerError, h.Logger)
		return
	}

	infos, err := h.AnnouncementRepo.GetInfoForShoppingCart(ids)
	if err != nil {
		myErr.SendErrorTo(w, err, http.StatusInternalServerError, h.Logger)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(infos)
	if err != nil {
		h.Logger.Warnw("error writing response", "err", err)
		return
	}
}

// PurchaseFromCart - POST /cart/{userID}/purchase
// Принимает в теле запроса массив в джейсоне айдишников товаров:
// [
//
//	"id1",
//	"id2"
//
// ]
// Ожидаю, что фронт, после успешной оплаты сделает вывод, что вы успешно купили эти товары
func (h *ShoppingCartHandler) PurchaseFromCart(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["userID"]

	if _, err := uuid.Parse(userID); err != nil {
		myErr.SendErrorTo(w, myErr.ErrBadID, http.StatusBadRequest, h.Logger)
		return
	}

	// Декодируем список ID объявлений из тела запроса
	var requestedIDs []string
	if err := json.NewDecoder(r.Body).Decode(&requestedIDs); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "invalid request body",
		})
		return
	}
	if len(requestedIDs) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "empty announcement list",
		})
		return
	}

	// Получаем текущую корзину пользователя
	cartIDs, err := h.CartRepo.GetByUserID(userID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "failed to get cart",
		})
		return
	}

	// Проверяем, что все переданные товары действительно есть в корзине
	validItems := map[string]bool{}
	for _, id := range cartIDs {
		validItems[id] = true
	}
	for _, reqID := range requestedIDs {
		if !validItems[reqID] {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "one or more items not in cart",
			})
			return
		}
	}

	// Получаем информацию о товарах для расчета суммы
	infos, err := h.AnnouncementRepo.GetInfoForShoppingCart(requestedIDs)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "failed to get announcement info",
		})
		return
	}

	var total int64 = 0
	for _, item := range infos {
		discountedPrice := float64(item.Price*(int64(100)-int64(item.Discount))) / 100.0
		total += int64(math.Ceil(discountedPrice))
	}

	// Получаем баланс пользователя
	balance, err := h.UserRepo.GetBalanceByUserID(userID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "failed to get user balance",
		})
		return
	}

	if balance < total {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusPaymentRequired)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "insufficient funds",
		})
		return
	}

	// Списываем деньги у пользователя
	_, err = h.UserRepo.TopUpBalance(userID, -total)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "failed to charge user",
		})
		return
	}

	// Удаляем купленные товары из корзины
	for _, id := range requestedIDs {
		err = h.CartRepo.DeleteAnnouncement(userID, id)
		if err != nil {
			h.Logger.Warnw("failed to delete item from cart after purchase", "userID", userID, "annID", id, "err", err)
			// продолжаем, но логируем
		}
	}

	// Отправляем подтверждение
	h.Logger.Infof("user %s purchased items %v for total %d", userID, requestedIDs, total)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"total":  total,
	})
}
