package analytics

import (
	"context"
	"encoding/json"
	"errors"
	"gafroshka-main/internal/kafka"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// fakeService нужен для «подмены» AnalyticsService в тестах хендлера.
type fakeService struct {
	// какие параметры были переданы
	lastUserID string
	lastLimit  int

	returnCategories []int
	returnErr        error
}

func (f *fakeService) ProcessEvent(ctx context.Context, event kafka.Event) error {
	// не используется в этих тестах
	return nil
}

func (f *fakeService) GetTopCategories(ctx context.Context, userID string, limit int) ([]int, error) {
	f.lastUserID = userID
	f.lastLimit = limit
	return f.returnCategories, f.returnErr
}

func TestHandler_GetUserPreferences_MissingUserID(t *testing.T) {
	logger := zapTestLogger(t)
	svc := &fakeService{}
	handler := NewHandler(svc, logger)

	// Формируем запрос без user_id в URL, должен вернуть 404 (router просто не найдёт совпадений).
	req := httptest.NewRequest("GET", "/user//preferences", nil)
	rr := httptest.NewRecorder()

	// Чтобы хендлер видел переменные, нужно маршрутизировать через Router
	r := mux.NewRouter()
	r.HandleFunc("/user/{user_id}/preferences", handler.GetUserPreferences).Methods("GET")
	r.ServeHTTP(rr, req)

	// Ожидаем 400, так как в теле handler проверяет user_id == ""
	if rr.Code != http.StatusMovedPermanently {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandler_GetUserPreferences_Success(t *testing.T) {
	logger := zapTestLogger(t)
	svc := &fakeService{
		returnCategories: []int{11, 22, 33},
		returnErr:        nil,
	}
	handler := NewHandler(svc, logger)

	// Запрос с корректным user_id и параметром top=2
	req := httptest.NewRequest("GET", "/user/u-100/preferences?top=2", nil)
	rr := httptest.NewRecorder()

	r := mux.NewRouter()
	r.HandleFunc("/user/{user_id}/preferences", handler.GetUserPreferences).Methods("GET")
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	// Проверяем, что сервис вызван с правильными аргументами:
	if svc.lastUserID != "u-100" {
		t.Errorf("expected service.GetTopCategories userID=\"u-100\", got \"%s\"", svc.lastUserID)
	}
	if svc.lastLimit != 2 {
		t.Errorf("expected service.GetTopCategories limit=2, got %d", svc.lastLimit)
	}

	// Проверяем, что тело ответа — JSON-массив [11,22,33]
	var got []int
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}
	expected := []int{11, 22, 33}
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("expected category %d at index %d, got %d", expected[i], i, got[i])
		}
	}
}

func TestHandler_GetUserPreferences_DefaultTop(t *testing.T) {
	logger := zapTestLogger(t)
	svc := &fakeService{
		returnCategories: []int{7, 8},
		returnErr:        nil,
	}
	handler := NewHandler(svc, logger)

	// Запрос без параметра top → должен использовать topN = 3 (по умолчанию).
	req := httptest.NewRequest("GET", "/user/u-200/preferences", nil)
	rr := httptest.NewRecorder()

	r := mux.NewRouter()
	r.HandleFunc("/user/{user_id}/preferences", handler.GetUserPreferences).Methods("GET")
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	// Проверяем, что сервис вызван с default top=3
	if svc.lastLimit != 3 {
		t.Errorf("expected default limit=3, got %d", svc.lastLimit)
	}
}

func TestHandler_GetUserPreferences_ServiceError(t *testing.T) {
	logger := zapTestLogger(t)
	svc := &fakeService{
		returnCategories: nil,
		returnErr:        errors.New("something went wrong"),
	}
	handler := NewHandler(svc, logger)

	req := httptest.NewRequest("GET", "/user/u-300/preferences", nil)
	rr := httptest.NewRecorder()

	r := mux.NewRouter()
	r.HandleFunc("/user/{user_id}/preferences", handler.GetUserPreferences).Methods("GET")
	r.ServeHTTP(rr, req)

	// Ожидаем Internal Server Error, т.к. сервис вернул ошибку
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}
}
