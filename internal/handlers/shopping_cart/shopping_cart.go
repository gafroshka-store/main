package handlers

import (
	"encoding/json"
	"errors"
	"gafroshka-main/internal/kafka"
	"gafroshka-main/internal/user"
	"math"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"time"

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
	EventProducer    kafka.EventProducer
}

// NewShoppingCartHandler конструктор
func NewShoppingCartHandler(
	log *zap.SugaredLogger,
	cr shopping_cart.ShoppingCartRepo,
	ar announcement.AnnouncementRepo,
	ur user.UserRepo,
	ep kafka.EventProducer,
) *ShoppingCartHandler {
	return &ShoppingCartHandler{
		Logger:           log,
		CartRepo:         cr,
		AnnouncementRepo: ar,
		UserRepo:         ur,
		EventProducer:    ep,
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

	// После успешного добавления — отправляем событие "view" (просмотр карточки объявления) в Kafka
	ann, err := h.AnnouncementRepo.GetByID(annID)
	if err != nil {
		h.Logger.Warnf("failed to fetch announcement %s for analytics: %v", annID, err)
	} else {
		event := kafka.Event{
			UserID:     userID,
			Type:       kafka.EventTypeView,
			Categories: []int{ann.Category},
			Timestamp:  time.Now(),
		}
		if err := h.EventProducer.SendEvent(r.Context(), event); err != nil {
			h.Logger.Warnf("failed to send view event on AddToShoppingCart: %v", err)
		}
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
			// Если пусто, то возвращаем no content
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
// Принимает в теле запроса массив в JSON айдишников товаров:
// [
//
//	"id1",
//	"id2"
//
// ]
// После успешной оплаты возвращаем {"status": "success", "total": <сумма>}
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
		myErr.SendErrorTo(w, err, http.StatusBadRequest, h.Logger)
		return
	}
	if len(requestedIDs) == 0 {
		myErr.SendErrorTo(w, errors.New("empty announcement list"), http.StatusBadRequest, h.Logger)
		return
	}

	// Получаем текущую корзину пользователя
	cartIDs, err := h.CartRepo.GetByUserID(userID)
	if err != nil {
		myErr.SendErrorTo(w, err, http.StatusInternalServerError, h.Logger)
		return
	}

	// Проверяем, что все переданные товары действительно есть в корзине
	validItems := map[string]bool{}
	for _, id := range cartIDs {
		validItems[id] = true
	}
	for _, reqID := range requestedIDs {
		if !validItems[reqID] {
			myErr.SendErrorTo(w, errors.New("one or more items not in cart"), http.StatusBadRequest, h.Logger)
			return
		}
	}

	// Получаем информацию о товарах для расчета суммы
	infos, err := h.AnnouncementRepo.GetInfoForShoppingCart(requestedIDs)
	if err != nil {
		myErr.SendErrorTo(w, err, http.StatusInternalServerError, h.Logger)
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
		myErr.SendErrorTo(w, err, http.StatusInternalServerError, h.Logger)
		return
	}

	if balance < total {
		myErr.SendErrorTo(w, errors.New("insufficient funds"), http.StatusPaymentRequired, h.Logger)
		return
	}

	// Списываем деньги у пользователя
	_, err = h.UserRepo.TopUpBalance(userID, -total)
	if err != nil {
		myErr.SendErrorTo(w, err, http.StatusInternalServerError, h.Logger)
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

	// После успешной покупки — отправляем событие "purchase" в Kafka
	var categories []int
	catSet := make(map[int]struct{})
	for _, annID := range requestedIDs {
		ann, err := h.AnnouncementRepo.GetByID(annID)
		if err != nil {
			h.Logger.Warnf("failed to fetch announcement %s for analytics: %v", annID, err)
			continue
		}
		if _, exists := catSet[ann.Category]; !exists {
			catSet[ann.Category] = struct{}{}
			categories = append(categories, ann.Category)
		}
	}
	if len(categories) > 0 {
		event := kafka.Event{
			UserID:     userID,
			Type:       kafka.EventTypePurchase,
			Categories: categories,
			Timestamp:  time.Now(),
		}
		if err := h.EventProducer.SendEvent(r.Context(), event); err != nil {
			h.Logger.Warnf("failed to send purchase event: %v", err)
		}
	} else {
		h.Logger.Infof("no valid categories found for PurchaseFromCart, skipping analytics")
	}

	// Отправляем подтверждение
	h.Logger.Infof("user %s purchased items %v for total %d", userID, requestedIDs, total)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"total":  total,
	})
}
