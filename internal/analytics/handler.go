package analytics

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"net/http"
	"strconv"
)

type Handler struct {
	service *Service
	logger  *zap.SugaredLogger
}

func NewHandler(service *Service, logger *zap.SugaredLogger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

func (h *Handler) GetUserPreferences(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["user_id"]
	if userID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	topN := 3 // По умолчанию
	if topParam := r.URL.Query().Get("top"); topParam != "" {
		if n, err := strconv.Atoi(topParam); err == nil && n > 0 {
			topN = n
		}
	}

	categories, err := h.service.GetTopCategories(r.Context(), userID, topN)
	if err != nil {
		h.logger.Errorf("Failed to get user preferences: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if len(categories) == 0 {
		categories = []int{} // Пустой массив вместо null
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(categories); err != nil {
		h.logger.Errorf("Failed to encode response: %v", err)
	}
}
