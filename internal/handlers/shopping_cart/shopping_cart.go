package handlers

import (
	"encoding/json"
	"errors"
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
}

// NewShoppingCartHandler конструктор
func NewShoppingCartHandler(log *zap.SugaredLogger, cr shopping_cart.ShoppingCartRepo, ar announcement.AnnouncementRepo) *ShoppingCartHandler {
	return &ShoppingCartHandler{Logger: log, CartRepo: cr, AnnouncementRepo: ar}
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
