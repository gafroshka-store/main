package handlers

import (
	"encoding/json"
	"errors"
	myErr "gafroshka-main/internal/types/errors"
	types "gafroshka-main/internal/types/user"
	"gafroshka-main/internal/user"
	"net/http"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type UserHandler struct {
	Logger         *zap.SugaredLogger
	UserRepository user.UserRepo
}

func NewUserHandler(l *zap.SugaredLogger, ur user.UserRepo) *UserHandler {
	return &UserHandler{
		Logger:         l,
		UserRepository: ur,
	}
}

func (h *UserHandler) Info(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if id == "" {
		myErr.SendErrorTo(w, errors.New("missing user id"), http.StatusBadRequest, h.Logger)
		return
	}

	userInfo, err := h.UserRepository.Info(id)
	if err != nil {
		if errors.Is(err, myErr.ErrNotFound) {
			myErr.SendErrorTo(w, err, http.StatusNotFound, h.Logger)
			return
		}
		myErr.SendErrorTo(w, err, http.StatusInternalServerError, h.Logger)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(userInfo); err != nil {
		myErr.SendErrorTo(w, err, http.StatusInternalServerError, h.Logger)
		return
	}

	h.Logger.Infof("get info by user: %s", id)
}

func (h *UserHandler) ChangeProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	if userID == "" {
		myErr.SendErrorTo(w, errors.New("missing user id"), http.StatusBadRequest, h.Logger)
		return
	}

	var updateData types.ChangeUser
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		myErr.SendErrorTo(w, errors.New("invalid JSON payload"), http.StatusBadRequest, h.Logger)
		return
	}

	user, err := h.UserRepository.ChangeProfile(userID, updateData)
	if err != nil {
		if errors.Is(err, myErr.ErrNotFound) {
			myErr.SendErrorTo(w, err, http.StatusNotFound, h.Logger)
			return
		}
		myErr.SendErrorTo(w, err, http.StatusInternalServerError, h.Logger)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(user); err != nil {
		myErr.SendErrorTo(w, err, http.StatusInternalServerError, h.Logger)
		return
	}

	h.Logger.Infof("user profile updated successfully: %s", userID)
}
