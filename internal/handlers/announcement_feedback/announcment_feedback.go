package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	annfb "gafroshka-main/internal/announcment_feedback"
	myErr "gafroshka-main/internal/types/errors"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type AnnouncementFeedbackHandler struct {
	Logger       *zap.SugaredLogger
	FeedbackRepo annfb.FeedbackRepo
}

func NewAnnouncementFeedbackHandler(logger *zap.SugaredLogger, repo annfb.FeedbackRepo) *AnnouncementFeedbackHandler {
	return &AnnouncementFeedbackHandler{
		Logger:       logger,
		FeedbackRepo: repo,
	}
}

func (h *AnnouncementFeedbackHandler) Create(w http.ResponseWriter, r *http.Request) {
	var f annfb.Feedback
	if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
		myErr.SendErrorTo(w, errors.New("invalid JSON payload"), http.StatusBadRequest, h.Logger)
		return
	}

	created, err := h.FeedbackRepo.Create(f)
	if err != nil {
		myErr.SendErrorTo(w, err, http.StatusInternalServerError, h.Logger)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(created); err != nil {
		myErr.SendErrorTo(w, err, http.StatusInternalServerError, h.Logger)
		return
	}
}

// Delete handles DELETE /feedback/{id}
func (h *AnnouncementFeedbackHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if id == "" {
		myErr.SendErrorTo(w, errors.New("missing feedback id"), http.StatusBadRequest, h.Logger)
		return
	}

	err := h.FeedbackRepo.Delete(id)
	if err != nil {
		if errors.Is(err, myErr.ErrNotFoundFeedback) {
			myErr.SendErrorTo(w, err, http.StatusNotFound, h.Logger)
		} else {
			myErr.SendErrorTo(w, err, http.StatusInternalServerError, h.Logger)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetByAnnouncementID handles GET /feedback/announcement/{id}
func (h *AnnouncementFeedbackHandler) GetByAnnouncementID(w http.ResponseWriter, r *http.Request) {
	announcementID := mux.Vars(r)["id"]
	if announcementID == "" {
		myErr.SendErrorTo(w, errors.New("missing announcement id"), http.StatusBadRequest, h.Logger)
		return
	}

	feedbacks, err := h.FeedbackRepo.GetByAnnouncementID(announcementID)
	if err != nil {
		myErr.SendErrorTo(w, err, http.StatusInternalServerError, h.Logger)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(feedbacks); err != nil {
		myErr.SendErrorTo(w, err, http.StatusInternalServerError, h.Logger)
		return
	}
}
