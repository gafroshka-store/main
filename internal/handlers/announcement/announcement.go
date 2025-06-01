package announcement

import (
	"encoding/json"
	"errors"
	"fmt"
	"gafroshka-main/internal/contextutil"
	"gafroshka-main/internal/kafka"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"gafroshka-main/internal/announcement"
	typesAnn "gafroshka-main/internal/types/announcement"
	myErr "gafroshka-main/internal/types/errors"
)

// AnnouncementHandler работает с AnnouncementRepo и EventProducer интерфейсами.
type AnnouncementHandler struct {
	Logger           *zap.SugaredLogger
	AnnouncementRepo announcement.AnnouncementRepo
	EventProducer    kafka.EventProducer
}

func NewAnnouncementHandler(
	l *zap.SugaredLogger,
	ar announcement.AnnouncementRepo,
	kp kafka.EventProducer,
) *AnnouncementHandler {
	return &AnnouncementHandler{
		Logger:           l,
		AnnouncementRepo: ar,
		EventProducer:    kp,
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

	// Отправка события просмотра в Kafka
	if userID, ok := contextutil.GetUserIDFromContext(r.Context()); ok {
		event := kafka.Event{
			UserID:     userID,
			Type:       kafka.EventTypeView,
			Categories: []int{ann.Category},
			Timestamp:  time.Now(),
		}

		if err := h.EventProducer.SendEvent(r.Context(), event); err != nil {
			h.Logger.Warnf("Failed to send view event: %v", err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(ann); err != nil {
		myErr.SendErrorTo(w, err, http.StatusInternalServerError, h.Logger)
		return
	}

	h.Logger.Infof("announcement created: %s", ann.ID)
}

// остальные методы без изменений
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

func (h *AnnouncementHandler) GetTopN(w http.ResponseWriter, r *http.Request) {
	var input struct {
		UserID string `json:"user_id"`
		Limit  int    `json:"limit"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		myErr.SendErrorTo(w, errors.New("invalid JSON payload"), http.StatusBadRequest, h.Logger)
		return
	}

	if input.Limit <= 0 {
		myErr.SendErrorTo(w, errors.New("limit must be positive number"), http.StatusBadRequest, h.Logger)
		return
	}

	var categories []int
	if input.UserID != "" {
		// Запрос к сервису аналитики
		url := fmt.Sprintf("http://localhost:8082/user/%s/preferences?top=%d", input.UserID, input.Limit)
		resp, err := http.Get(url)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				if err := json.NewDecoder(resp.Body).Decode(&categories); err != nil {
					h.Logger.Warnf("Failed to decode user preferences: %v", err)
				}
			}
		} else {
			h.Logger.Warnf("Failed to get user preferences: %v", err)
		}
	}

	anns, err := h.AnnouncementRepo.GetTopN(input.Limit, categories)
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

	h.Logger.Infof("fetched top %d announcements for user %s, categories %v", input.Limit, input.UserID, categories)
}

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

	if userID, ok := contextutil.GetUserIDFromContext(r.Context()); ok {
		categories := make([]int, 0, len(anns))
		for _, a := range anns {
			categories = append(categories, a.Category)
		}

		event := kafka.Event{
			UserID:     userID,
			Type:       kafka.EventTypeSearch,
			Categories: categories,
			Timestamp:  time.Now(),
		}

		if err := h.EventProducer.SendEvent(r.Context(), event); err != nil {
			h.Logger.Warnf("Failed to send search event: %v", err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(anns); err != nil {
		myErr.SendErrorTo(w, err, http.StatusInternalServerError, h.Logger)
		return
	}

	h.Logger.Infof("searched announcements with query: %s", q)
}
