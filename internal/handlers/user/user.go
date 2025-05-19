package user

import (
	"context"
	"encoding/json"
	"errors"
	"gafroshka-main/internal/session"
	myErr "gafroshka-main/internal/types/errors"
	types "gafroshka-main/internal/types/user"
	"gafroshka-main/internal/user"
	"github.com/google/uuid"
	"net/http"
	"net/mail"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type UserHandler struct {
	Logger         *zap.SugaredLogger
	UserRepository user.UserRepo
	SessionManger  session.SessionRepo
}

func NewUserHandler(l *zap.SugaredLogger, ur user.UserRepo, sr session.SessionRepo) *UserHandler {
	return &UserHandler{
		Logger:         l,
		UserRepository: ur,
		SessionManger:  sr,
	}
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var form types.CreateUser
	if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
		myErr.SendErrorTo(w, err, http.StatusBadRequest, h.Logger)
		return
	}
	// Проверим на валидность переданной почты
	_, err := mail.ParseAddress(form.Email)
	if err != nil {
		myErr.SendErrorTo(w, err, http.StatusBadRequest, h.Logger)
		return
	}
	// Создаем пользователя
	u, err := h.UserRepository.CreateUser(form)
	if err != nil {
		if errors.Is(err, myErr.ErrAlreadyExists) {
			myErr.SendErrorTo(w, err, http.StatusUnprocessableEntity, h.Logger)
			return
		}

		myErr.SendErrorTo(w, err, http.StatusInternalServerError, h.Logger)
		return
	}

	// Создаем для него сессию
	sess, err := h.SessionManger.CreateSession(context.Background(), w, u.ID, u.Email)
	if err != nil {
		myErr.SendErrorTo(w, err, http.StatusInternalServerError, h.Logger)
		return
	}

	w.Header().Set("Access-control-allow-", "*")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusCreated)

	h.Logger.Infof("created session for %v", sess.ID)
}

type RequestRegisterForm struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var form RequestRegisterForm
	if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
		myErr.SendErrorTo(w, err, http.StatusBadRequest, h.Logger)
		return
	}

	u, err := h.UserRepository.CheckUser(form.Email, form.Password)
	if err != nil {
		if errors.Is(err, myErr.ErrNotFound) {
			myErr.SendErrorTo(w, myErr.ErrNotFound, http.StatusNotFound, h.Logger)
			return
		}

		if errors.Is(err, myErr.ErrBadPassword) {
			myErr.SendErrorTo(w, err, http.StatusUnauthorized, h.Logger)
			return
		}

		myErr.SendErrorTo(w, err, http.StatusInternalServerError, h.Logger)
		return
	}

	// Создаем для него сессию
	sess, err := h.SessionManger.CreateSession(context.Background(), w, u.ID, u.Email)
	if err != nil {
		myErr.SendErrorTo(w, err, http.StatusInternalServerError, h.Logger)
		return
	}

	w.WriteHeader(http.StatusOK)
	h.Logger.Infof("created session for %v", sess.ID)
}

func (h *UserHandler) Info(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	_, err := uuid.Parse(id)
	if err != nil {
		myErr.SendErrorTo(w, myErr.ErrBadID, http.StatusBadRequest, h.Logger)
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

	_, err := uuid.Parse(userID)
	if err != nil {
		myErr.SendErrorTo(w, myErr.ErrBadID, http.StatusBadRequest, h.Logger)
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
