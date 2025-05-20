package handlers

import (
	"encoding/json"
	"errors"
	myErr "gafroshka-main/internal/types/errors"
	types "gafroshka-main/internal/types/user_feedback"
	userFeedback "gafroshka-main/internal/user_feedback"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"net/http"
)

const (
	minRating        = 1
	maxRating        = 5
	maxCommentLength = 1000
)

type UserFeedbackHandler struct {
	Logger                 *zap.SugaredLogger
	UserFeedbackRepository userFeedback.UserFeedbackRepo
}

func NewUserFeedbackHandler(l *zap.SugaredLogger, repo userFeedback.UserFeedbackRepo) *UserFeedbackHandler {
	return &UserFeedbackHandler{
		Logger:                 l,
		UserFeedbackRepository: repo,
	}
}

func (h *UserFeedbackHandler) Create(w http.ResponseWriter, r *http.Request) {
	var feedback userFeedback.UserFeedback
	if err := json.NewDecoder(r.Body).Decode(&feedback); err != nil {
		myErr.SendErrorTo(w, myErr.ErrInvalidJSONPayload, http.StatusBadRequest, h.Logger)

		return
	}

	if feedback.Rating < minRating || feedback.Rating > maxRating {
		myErr.SendErrorTo(w, myErr.ErrRatingIsInvalid, http.StatusBadRequest, h.Logger)
		return
	}
	if len(feedback.Comment) > maxCommentLength {
		myErr.SendErrorTo(w, myErr.ErrCommentIsTooLong, http.StatusBadRequest, h.Logger)
		return
	}

	createdFeedback, err := h.UserFeedbackRepository.Create(r.Context(), &feedback)
	if err != nil {
		myErr.SendErrorTo(w, err, http.StatusInternalServerError, h.Logger)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(createdFeedback); err != nil {
		myErr.SendErrorTo(w, err, http.StatusInternalServerError, h.Logger)

		return
	}

	h.Logger.Infof("Created user feedback with id: %s", createdFeedback.ID)
}

func (h *UserFeedbackHandler) GetByUserID(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["id"]
	if userID == "" {
		myErr.SendErrorTo(w, myErr.ErrMissingFeedbackID, http.StatusBadRequest, h.Logger)

		return
	}

	feedbacks, err := h.UserFeedbackRepository.GetByUserID(r.Context(), userID)
	if err != nil {
		myErr.SendErrorTo(w, err, http.StatusInternalServerError, h.Logger)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(feedbacks); err != nil {
		myErr.SendErrorTo(w, err, http.StatusInternalServerError, h.Logger)

		return
	}

	h.Logger.Infof("Retrieved feedbacks for user id: %s", userID)
}

func (h *UserFeedbackHandler) Update(w http.ResponseWriter, r *http.Request) {
	feedbackID := mux.Vars(r)["id"]
	if feedbackID == "" {
		myErr.SendErrorTo(w, myErr.ErrMissingFeedbackID, http.StatusBadRequest, h.Logger)

		return
	}

	var updateData types.UpdateUserFeedback
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		myErr.SendErrorTo(w, myErr.ErrInvalidJSONPayload, http.StatusBadRequest, h.Logger)

		return
	}

	if len(updateData.Comment) > maxCommentLength {
		myErr.SendErrorTo(w, myErr.ErrCommentIsTooLong, http.StatusBadRequest, h.Logger)

		return
	}

	updatedFeedback, err := h.UserFeedbackRepository.Update(r.Context(), feedbackID, updateData)
	if err != nil {
		if errors.Is(err, myErr.ErrNotFoundUserFeedback) {
			myErr.SendErrorTo(w, err, http.StatusNotFound, h.Logger)
		} else {
			myErr.SendErrorTo(w, err, http.StatusInternalServerError, h.Logger)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(updatedFeedback); err != nil {
		myErr.SendErrorTo(w, err, http.StatusInternalServerError, h.Logger)
		return
	}
	w.WriteHeader(http.StatusOK)

	h.Logger.Infof("Updated user feedback with id: %s", feedbackID)
}

func (h *UserFeedbackHandler) Delete(w http.ResponseWriter, r *http.Request) {
	feedbackID := mux.Vars(r)["id"]
	if feedbackID == "" {
		myErr.SendErrorTo(w, myErr.ErrMissingFeedbackID, http.StatusBadRequest, h.Logger)

		return
	}

	err := h.UserFeedbackRepository.Delete(r.Context(), feedbackID)
	if err != nil {
		if errors.Is(err, myErr.ErrNotFoundUserFeedback) {
			myErr.SendErrorTo(w, err, http.StatusNotFound, h.Logger)
		} else {
			myErr.SendErrorTo(w, err, http.StatusInternalServerError, h.Logger)
		}

		return
	}

	w.WriteHeader(http.StatusOK)
	h.Logger.Infof("Deleted user feedback with id: %s", feedbackID)
}
