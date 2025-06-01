package handlers

import (
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"net/http"

	"gafroshka-main/internal/announcement"
	typesAnn "gafroshka-main/internal/types/announcement"
	myErr "gafroshka-main/internal/types/errors"
)

type AnnouncementHandler struct {
	Logger           *zap.SugaredLogger
	AnnouncementRepo announcement.AnnouncementRepo
}

func NewAnnouncementHandler(l *zap.SugaredLogger, ar announcement.AnnouncementRepo) *AnnouncementHandler {
	return &AnnouncementHandler{
		Logger:           l,
		AnnouncementRepo: ar,
	}
}

// Create handles POST /announcement
func (h *AnnouncementHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input typesAnn.CreateAnnouncement
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		myErr.SendErrorTo(w, errors.New("invalid JSON payload"), http.StatusBadRequest, h.Logger)
		return
	}

	ann, err := h.AnnouncementRepo.Create(input)
	if err != nil {
		myErr.SendErrorTo(w, err, http.StatusInternalServerError, h.Logger)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(ann); err != nil {
		myErr.SendErrorTo(w, err, http.StatusInternalServerError, h.Logger)
		return
	}

	h.Logger.Infof("announcement created: %s", ann.ID)
}

// GetByID handles GET /announcement/{id}
func (h *AnnouncementHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		myErr.SendErrorTo(w, errors.New("missing announcement id"), http.StatusBadRequest, h.Logger)
		return
	}

	ann, err := h.AnnouncementRepo.GetByID(id)
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
	if err := json.NewEncoder(w).Encode(ann); err != nil {
		myErr.SendErrorTo(w, err, http.StatusInternalServerError, h.Logger)
		return
	}

	h.Logger.Infof("fetched announcement by id: %s", id)
}

// GetTopN handles POST /announcements/top
func (h *AnnouncementHandler) GetTopN(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Limit int `json:"limit"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		myErr.SendErrorTo(w, errors.New("invalid JSON payload"), http.StatusBadRequest, h.Logger)
		return
	}

	if input.Limit <= 0 {
		myErr.SendErrorTo(w, errors.New("limit must be positive number"), http.StatusBadRequest, h.Logger)
		return
	}

	anns, err := h.AnnouncementRepo.GetTopN(input.Limit)
	if err != nil {
		myErr.SendErrorTo(w, err, http.StatusInternalServerError, h.Logger)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(anns); err != nil {
		myErr.SendErrorTo(w, err, http.StatusInternalServerError, h.Logger)
		return
	}

	h.Logger.Infof("fetched top %d announcements", input.Limit)
}

// Search handles GET /announcements/search?q={query}
func (h *AnnouncementHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		myErr.SendErrorTo(w, errors.New("missing query parameter"), http.StatusBadRequest, h.Logger)
		return
	}

	anns, err := h.AnnouncementRepo.Search(q)
	if err != nil {
		myErr.SendErrorTo(w, err, http.StatusInternalServerError, h.Logger)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(anns); err != nil {
		myErr.SendErrorTo(w, err, http.StatusInternalServerError, h.Logger)
		return
	}

	h.Logger.Infof("searched announcements with query: %s", q)
}
