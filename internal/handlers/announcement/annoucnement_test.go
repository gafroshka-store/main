package announcement

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	repoAnn "gafroshka-main/internal/announcement"
	"gafroshka-main/internal/kafka"
	"gafroshka-main/internal/middleware"
	"gafroshka-main/internal/session"
	typesAnn "gafroshka-main/internal/types/announcement"
	myErr "gafroshka-main/internal/types/errors"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// ----------------------------
// Вспомогательные «фейковые» реализации
// ----------------------------

// fakeRepo реализует интерфейс repoAnn.AnnouncementRepo.
type fakeRepo struct {
	// Для Create
	lastCreateInput typesAnn.CreateAnnouncement
	returnCreateAnn *repoAnn.Announcement
	returnCreateErr error

	// Для GetByID
	lastGetByIDInput string
	returnGetByIDAnn *repoAnn.Announcement
	returnGetByIDErr error

	// Для GetTopN
	lastGetTopNLimit      int
	lastGetTopNCategories []int
	returnGetTopNAnns     []repoAnn.Announcement
	returnGetTopNErr      error

	// Для Search
	lastSearchQuery  string
	returnSearchAnns []repoAnn.Announcement
	returnSearchErr  error

	// Для UpdateRating
	lastUpdateRatingID    string
	lastUpdateRatingValue int
	returnUpdateRatingAnn *repoAnn.Announcement
	returnUpdateRatingErr error
}

func (f *fakeRepo) Create(a typesAnn.CreateAnnouncement) (*repoAnn.Announcement, error) {
	f.lastCreateInput = a
	return f.returnCreateAnn, f.returnCreateErr
}

func (f *fakeRepo) GetByID(id string) (*repoAnn.Announcement, error) {
	f.lastGetByIDInput = id
	return f.returnGetByIDAnn, f.returnGetByIDErr
}

func (f *fakeRepo) GetTopN(limit int, categories []int) ([]repoAnn.Announcement, error) {
	f.lastGetTopNLimit = limit
	f.lastGetTopNCategories = append([]int(nil), categories...)
	return f.returnGetTopNAnns, f.returnGetTopNErr
}

func (f *fakeRepo) Search(query string) ([]repoAnn.Announcement, error) {
	f.lastSearchQuery = query
	return f.returnSearchAnns, f.returnSearchErr
}

func (f *fakeRepo) UpdateRating(id string, rate int) (*repoAnn.Announcement, error) {
	f.lastUpdateRatingID = id
	f.lastUpdateRatingValue = rate
	return f.returnUpdateRatingAnn, f.returnUpdateRatingErr
}

// fakeProducer реализует интерфейс kafka.EventProducer
type fakeProducer struct {
	calledEvents []kafka.Event
	returnError  error
}

func (f *fakeProducer) SendEvent(ctx context.Context, event kafka.Event) error {
	f.calledEvents = append(f.calledEvents, event)
	return f.returnError
}

func (f *fakeProducer) Close() error {
	return nil
}

// zapTestLogger создаёт «тихий» SugaredLogger для тестов
func zapTestLogger(t *testing.T) *zap.SugaredLogger {
	t.Helper()
	logger, err := zap.NewDevelopmentConfig().Build(zap.AddCallerSkip(1))
	if err != nil {
		t.Fatalf("failed to create zap logger: %v", err)
	}
	return logger.Sugar()
}

// ----------------------------
// Тесты для метода Create
// ----------------------------

func TestCreate_InvalidJSON(t *testing.T) {
	logger := zapTestLogger(t)
	repo := &fakeRepo{}
	prod := &fakeProducer{}
	handler := NewAnnouncementHandler(logger, repo, prod)

	req := httptest.NewRequest(http.MethodPost, "/announcement", bytes.NewBufferString(`{bad json`))
	rr := httptest.NewRecorder()

	handler.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestCreate_RepoError(t *testing.T) {
	logger := zapTestLogger(t)
	repo := &fakeRepo{returnCreateErr: errors.New("db failure")}
	prod := &fakeProducer{}
	handler := NewAnnouncementHandler(logger, repo, prod)

	input := typesAnn.CreateAnnouncement{
		Name:         "Test",
		Description:  "Desc",
		UserSellerID: "user-1",
		Price:        100,
		Category:     5,
		Discount:     0,
	}
	body, _ := json.Marshal(input)
	req := httptest.NewRequest(http.MethodPost, "/announcement", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	handler.Create(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}
	if len(prod.calledEvents) != 0 {
		t.Errorf("expected SendEvent NOT to be called, but it was")
	}
}

func TestCreate_Success_NoUserInContext(t *testing.T) {
	logger := zapTestLogger(t)
	returnAnn := &repoAnn.Announcement{
		ID:           "ann-123",
		Name:         "My Announcement",
		Description:  "Some description",
		UserSellerID: "user-1",
		Price:        200,
		Category:     3,
		Discount:     10,
		IsActive:     true,
		Rating:       0,
		RatingCount:  0,
		CreatedAt:    time.Now(),
	}
	repo := &fakeRepo{returnCreateAnn: returnAnn, returnCreateErr: nil}
	prod := &fakeProducer{}
	handler := NewAnnouncementHandler(logger, repo, prod)

	input := typesAnn.CreateAnnouncement{
		Name:         returnAnn.Name,
		Description:  returnAnn.Description,
		UserSellerID: returnAnn.UserSellerID,
		Price:        returnAnn.Price,
		Category:     returnAnn.Category,
		Discount:     returnAnn.Discount,
	}
	body, _ := json.Marshal(input)
	req := httptest.NewRequest(http.MethodPost, "/announcement", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	handler.Create(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rr.Code)
	}
	if repo.lastCreateInput != input {
		t.Errorf("expected repo.Create to receive %+v, got %+v", input, repo.lastCreateInput)
	}
	if len(prod.calledEvents) != 0 {
		t.Errorf("expected SendEvent NOT to be called, but it was")
	}

	var got repoAnn.Announcement
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if got.ID != returnAnn.ID || got.Name != returnAnn.Name {
		t.Errorf("unexpected announcement in response: %+v", got)
	}
}

func TestCreate_Success_WithUserInContext(t *testing.T) {
	logger := zapTestLogger(t)
	returnAnn := &repoAnn.Announcement{
		ID:           "ann-456",
		Name:         "Another Ann",
		Description:  "Desc2",
		UserSellerID: "user-2",
		Price:        300,
		Category:     7,
		Discount:     5,
		IsActive:     true,
		Rating:       1.0,
		RatingCount:  1,
		CreatedAt:    time.Now(),
	}
	repo := &fakeRepo{returnCreateAnn: returnAnn, returnCreateErr: nil}
	prod := &fakeProducer{returnError: nil}
	handler := NewAnnouncementHandler(logger, repo, prod)

	input := typesAnn.CreateAnnouncement{
		Name:         returnAnn.Name,
		Description:  returnAnn.Description,
		UserSellerID: returnAnn.UserSellerID,
		Price:        returnAnn.Price,
		Category:     returnAnn.Category,
		Discount:     returnAnn.Discount,
	}
	body, _ := json.Marshal(input)
	req := httptest.NewRequest(http.MethodPost, "/announcement", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	sess := &session.Session{UserID: "user-2"}
	ctxWithSess := middleware.ContextWithSession(req.Context(), sess)
	req = req.WithContext(ctxWithSess)

	handler.Create(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rr.Code)
	}
	if len(prod.calledEvents) != 1 {
		t.Fatalf("expected SendEvent to be called once, but was called %d times", len(prod.calledEvents))
	}
	sent := prod.calledEvents[0]
	if sent.UserID != "user-2" {
		t.Errorf("expected event.UserID = \"user-2\", got %q", sent.UserID)
	}
	if sent.Type != kafka.EventTypeView {
		t.Errorf("expected event.Type = view, got %q", sent.Type)
	}
	if len(sent.Categories) != 1 || sent.Categories[0] != returnAnn.Category {
		t.Errorf("expected event.Categories = [%d], got %v", returnAnn.Category, sent.Categories)
	}
}

// ----------------------------
// Тесты для метода GetByID
// ----------------------------

func TestGetByID_MissingID(t *testing.T) {
	logger := zapTestLogger(t)
	repo := &fakeRepo{}
	prod := &fakeProducer{}
	handler := NewAnnouncementHandler(logger, repo, prod)

	req := httptest.NewRequest(http.MethodGet, "/announcement//", nil)
	rr := httptest.NewRecorder()

	r := mux.NewRouter()
	r.HandleFunc("/announcement/{id}", handler.GetByID).Methods(http.MethodGet)
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusMovedPermanently {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	logger := zapTestLogger(t)
	repo := &fakeRepo{returnGetByIDErr: myErr.ErrNotFound}
	prod := &fakeProducer{}
	handler := NewAnnouncementHandler(logger, repo, prod)

	req := httptest.NewRequest(http.MethodGet, "/announcement/nonexistent", nil)
	rr := httptest.NewRecorder()

	r := mux.NewRouter()
	r.HandleFunc("/announcement/{id}", handler.GetByID).Methods(http.MethodGet)
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
	if repo.lastGetByIDInput != "nonexistent" {
		t.Errorf("expected repo.GetByID to be called with \"nonexistent\", got %q", repo.lastGetByIDInput)
	}
}

func TestGetByID_RepoError(t *testing.T) {
	logger := zapTestLogger(t)
	repo := &fakeRepo{returnGetByIDErr: errors.New("db fail")}
	prod := &fakeProducer{}
	handler := NewAnnouncementHandler(logger, repo, prod)

	req := httptest.NewRequest(http.MethodGet, "/announcement/anyid", nil)
	rr := httptest.NewRecorder()

	r := mux.NewRouter()
	r.HandleFunc("/announcement/{id}", handler.GetByID).Methods(http.MethodGet)
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}
}

func TestGetByID_Success(t *testing.T) {
	logger := zapTestLogger(t)
	expectedAnn := &repoAnn.Announcement{
		ID:           "ann-789",
		Name:         "Found Ann",
		Description:  "Desc3",
		UserSellerID: "seller-1",
		Price:        150,
		Category:     9,
		Discount:     0,
		IsActive:     true,
		Rating:       2.5,
		RatingCount:  4,
		CreatedAt:    time.Now(),
	}
	repo := &fakeRepo{returnGetByIDAnn: expectedAnn, returnGetByIDErr: nil}
	prod := &fakeProducer{}
	handler := NewAnnouncementHandler(logger, repo, prod)

	req := httptest.NewRequest(http.MethodGet, "/announcement/ann-789", nil)
	rr := httptest.NewRecorder()

	r := mux.NewRouter()
	r.HandleFunc("/announcement/{id}", handler.GetByID).Methods(http.MethodGet)
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var got repoAnn.Announcement
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if got.ID != expectedAnn.ID || got.Name != expectedAnn.Name {
		t.Errorf("unexpected announcement in response: %+v", got)
	}
	if repo.lastGetByIDInput != "ann-789" {
		t.Errorf("expected repo.GetByID to be called with \"ann-789\", got %q", repo.lastGetByIDInput)
	}
}

// ----------------------------
// Тесты для метода GetTopN
// ----------------------------

func TestGetTopN_InvalidJSON(t *testing.T) {
	logger := zapTestLogger(t)
	repo := &fakeRepo{}
	prod := &fakeProducer{}
	handler := NewAnnouncementHandler(logger, repo, prod)

	req := httptest.NewRequest(http.MethodPost, "/announcements/top", bytes.NewBufferString(`{invalid}`))
	rr := httptest.NewRecorder()

	handler.GetTopN(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestGetTopN_InvalidLimit(t *testing.T) {
	logger := zapTestLogger(t)
	repo := &fakeRepo{}
	prod := &fakeProducer{}
	handler := NewAnnouncementHandler(logger, repo, prod)

	body, _ := json.Marshal(map[string]int{"limit": 0})
	req := httptest.NewRequest(http.MethodPost, "/announcements/top", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	handler.GetTopN(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestGetTopN_RepoError_NoUser(t *testing.T) {
	logger := zapTestLogger(t)
	repo := &fakeRepo{returnGetTopNErr: errors.New("db fail")}
	prod := &fakeProducer{}
	handler := NewAnnouncementHandler(logger, repo, prod)

	body, _ := json.Marshal(map[string]int{"limit": 2})
	req := httptest.NewRequest(http.MethodPost, "/announcements/top", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	handler.GetTopN(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}
	if repo.lastGetTopNLimit != 2 {
		t.Errorf("expected repo.GetTopN limit=2, got %d", repo.lastGetTopNLimit)
	}
	if len(repo.lastGetTopNCategories) != 0 {
		t.Errorf("expected repo.GetTopN categories empty, got %v", repo.lastGetTopNCategories)
	}
}

func TestGetTopN_Success_NoUser(t *testing.T) {
	logger := zapTestLogger(t)
	expectedAnns := []repoAnn.Announcement{
		{
			ID:           "ann-A",
			Name:         "Top A",
			Description:  "D1",
			UserSellerID: "sellerA",
			Price:        50,
			Category:     1,
			Discount:     0,
			IsActive:     true,
			Rating:       4.0,
			RatingCount:  2,
			CreatedAt:    time.Now(),
		},
		{
			ID:           "ann-B",
			Name:         "Top B",
			Description:  "D2",
			UserSellerID: "sellerB",
			Price:        80,
			Category:     2,
			Discount:     5,
			IsActive:     true,
			Rating:       3.7,
			RatingCount:  3,
			CreatedAt:    time.Now(),
		},
	}
	repo := &fakeRepo{returnGetTopNAnns: expectedAnns, returnGetTopNErr: nil}
	prod := &fakeProducer{}
	handler := NewAnnouncementHandler(logger, repo, prod)

	body, _ := json.Marshal(map[string]int{"limit": 2})
	req := httptest.NewRequest(http.MethodPost, "/announcements/top", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	handler.GetTopN(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if repo.lastGetTopNLimit != 2 {
		t.Errorf("expected repo.GetTopN limit=2, got %d", repo.lastGetTopNLimit)
	}
	if len(repo.lastGetTopNCategories) != 0 {
		t.Errorf("expected repo.GetTopN categories empty, got %v", repo.lastGetTopNCategories)
	}

	var got []repoAnn.Announcement
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(got) != len(expectedAnns) {
		t.Fatalf("expected %d announcements, got %d", len(expectedAnns), len(got))
	}
	if got[0].ID != expectedAnns[0].ID || got[1].ID != expectedAnns[1].ID {
		t.Errorf("unexpected announcements in response: %v", got)
	}
}

// ----------------------------
// Тесты для метода Search
// ----------------------------

func TestSearch_MissingQuery(t *testing.T) {
	logger := zapTestLogger(t)
	repo := &fakeRepo{}
	prod := &fakeProducer{}
	handler := NewAnnouncementHandler(logger, repo, prod)

	req := httptest.NewRequest(http.MethodGet, "/announcements/search", nil)
	rr := httptest.NewRecorder()

	r := mux.NewRouter()
	r.HandleFunc("/announcements/search", handler.Search).Methods(http.MethodGet)
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestSearch_RepoError(t *testing.T) {
	logger := zapTestLogger(t)
	repo := &fakeRepo{returnSearchErr: errors.New("db fail")}
	prod := &fakeProducer{}
	handler := NewAnnouncementHandler(logger, repo, prod)

	req := httptest.NewRequest(http.MethodGet, "/announcements/search?q=test", nil)
	rr := httptest.NewRecorder()

	r := mux.NewRouter()
	r.HandleFunc("/announcements/search", handler.Search).Methods(http.MethodGet)
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}
	if repo.lastSearchQuery != "test" {
		t.Errorf("expected repo.Search query=\"test\", got %q", repo.lastSearchQuery)
	}
}

func TestSearch_Success_NoUser(t *testing.T) {
	logger := zapTestLogger(t)
	expectedAnns := []repoAnn.Announcement{
		{
			ID:           "ann-X",
			Name:         "Search X",
			Description:  "DX",
			UserSellerID: "sellerX",
			Price:        40,
			Category:     3,
			Discount:     0,
			IsActive:     true,
			Rating:       4.5,
			RatingCount:  5,
			CreatedAt:    time.Now(),
		},
	}
	repo := &fakeRepo{returnSearchAnns: expectedAnns, returnSearchErr: nil}
	prod := &fakeProducer{}
	handler := NewAnnouncementHandler(logger, repo, prod)

	req := httptest.NewRequest(http.MethodGet, "/announcements/search?q=foo", nil)
	rr := httptest.NewRecorder()

	r := mux.NewRouter()
	r.HandleFunc("/announcements/search", handler.Search).Methods(http.MethodGet)
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if repo.lastSearchQuery != "foo" {
		t.Errorf("expected repo.Search query=\"foo\", got %q", repo.lastSearchQuery)
	}
	if len(prod.calledEvents) != 0 {
		t.Errorf("expected SendEvent NOT to be called, but it was")
	}

	var got []repoAnn.Announcement
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(got) != 1 || got[0].ID != expectedAnns[0].ID {
		t.Errorf("unexpected search results: %v", got)
	}
}

func TestSearch_Success_WithUser(t *testing.T) {
	logger := zapTestLogger(t)
	expectedAnns := []repoAnn.Announcement{
		{
			ID:           "ann-Y",
			Name:         "Search Y",
			Description:  "DY",
			UserSellerID: "sellerY",
			Price:        60,
			Category:     4,
			Discount:     0,
			IsActive:     true,
			Rating:       3.9,
			RatingCount:  2,
			CreatedAt:    time.Now(),
		},
	}
	repo := &fakeRepo{returnSearchAnns: expectedAnns, returnSearchErr: nil}
	prod := &fakeProducer{returnError: nil}
	handler := NewAnnouncementHandler(logger, repo, prod)

	req := httptest.NewRequest(http.MethodGet, "/announcements/search?q=bar", nil)
	rr := httptest.NewRecorder()
	sess := &session.Session{UserID: "user-xyz"}
	ctxWithSess := middleware.ContextWithSession(req.Context(), sess)
	req = req.WithContext(ctxWithSess)

	r := mux.NewRouter()
	r.HandleFunc("/announcements/search", handler.Search).Methods(http.MethodGet)
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if repo.lastSearchQuery != "bar" {
		t.Errorf("expected repo.Search query=\"bar\", got %q", repo.lastSearchQuery)
	}
	if len(prod.calledEvents) != 1 {
		t.Fatalf("expected SendEvent to be called once, but was called %d times", len(prod.calledEvents))
	}
	sent := prod.calledEvents[0]
	if sent.UserID != "user-xyz" {
		t.Errorf("expected event.UserID=\"user-xyz\", got %q", sent.UserID)
	}
	if sent.Type != kafka.EventTypeSearch {
		t.Errorf("expected event.Type=search, got %q", sent.Type)
	}
	if len(sent.Categories) != 1 || sent.Categories[0] != expectedAnns[0].Category {
		t.Errorf("expected event.Categories=[4], got %v", sent.Categories)
	}
}

// ----------------------------
// Тесты для метода UpdateRating
// ----------------------------

func TestUpdateRating_MissingID(t *testing.T) {
	logger := zapTestLogger(t)
	repo := &fakeRepo{}
	prod := &fakeProducer{}
	handler := NewAnnouncementHandler(logger, repo, prod)

	req := httptest.NewRequest(http.MethodPost, "/announcement//rating", bytes.NewBufferString(`{"rating":5}`))
	rr := httptest.NewRecorder()

	r := mux.NewRouter()
	r.HandleFunc("/announcement/{id}/rating", handler.UpdateRating).Methods(http.MethodPost)
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusMovedPermanently {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestUpdateRating_InvalidJSON(t *testing.T) {
	logger := zapTestLogger(t)
	repo := &fakeRepo{}
	prod := &fakeProducer{}
	handler := NewAnnouncementHandler(logger, repo, prod)

	req := httptest.NewRequest(http.MethodPost, "/announcement/ann-001/rating", bytes.NewBufferString(`{bad json}`))
	rr := httptest.NewRecorder()

	r := mux.NewRouter()
	r.HandleFunc("/announcement/{id}/rating", handler.UpdateRating).Methods(http.MethodPost)
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestUpdateRating_InvalidRatingValue(t *testing.T) {
	logger := zapTestLogger(t)
	repo := &fakeRepo{}
	prod := &fakeProducer{}
	handler := NewAnnouncementHandler(logger, repo, prod)

	body, _ := json.Marshal(map[string]int{"rating": 0})
	req := httptest.NewRequest(http.MethodPost, "/announcement/ann-002/rating", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	r := mux.NewRouter()
	r.HandleFunc("/announcement/{id}/rating", handler.UpdateRating).Methods(http.MethodPost)
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestUpdateRating_NotFound(t *testing.T) {
	logger := zapTestLogger(t)
	repo := &fakeRepo{returnUpdateRatingErr: myErr.ErrNotFound}
	prod := &fakeProducer{}
	handler := NewAnnouncementHandler(logger, repo, prod)

	body, _ := json.Marshal(map[string]int{"rating": 3})
	req := httptest.NewRequest(http.MethodPost, "/announcement/unknown/rating", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	r := mux.NewRouter()
	r.HandleFunc("/announcement/{id}/rating", handler.UpdateRating).Methods(http.MethodPost)
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
	if repo.lastUpdateRatingID != "unknown" {
		t.Errorf("expected UpdateRating called with id=\"unknown\", got %q", repo.lastUpdateRatingID)
	}
	if repo.lastUpdateRatingValue != 3 {
		t.Errorf("expected UpdateRating called with rate=3, got %d", repo.lastUpdateRatingValue)
	}
}

func TestUpdateRating_RepoError(t *testing.T) {
	logger := zapTestLogger(t)
	repo := &fakeRepo{returnUpdateRatingErr: errors.New("db fail")}
	prod := &fakeProducer{}
	handler := NewAnnouncementHandler(logger, repo, prod)

	body, _ := json.Marshal(map[string]int{"rating": 4})
	req := httptest.NewRequest(http.MethodPost, "/announcement/ann-003/rating", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	r := mux.NewRouter()
	r.HandleFunc("/announcement/{id}/rating", handler.UpdateRating).Methods(http.MethodPost)
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}
}

func TestUpdateRating_Success(t *testing.T) {
	logger := zapTestLogger(t)
	updatedAnn := &repoAnn.Announcement{
		ID:           "ann-004",
		Name:         "Rated Ann",
		Description:  "Desc4",
		UserSellerID: "seller4",
		Price:        120,
		Category:     2,
		Discount:     0,
		IsActive:     true,
		Rating:       4.5,
		RatingCount:  2,
		CreatedAt:    time.Now(),
	}
	repo := &fakeRepo{returnUpdateRatingAnn: updatedAnn, returnUpdateRatingErr: nil}
	prod := &fakeProducer{}
	handler := NewAnnouncementHandler(logger, repo, prod)

	body, _ := json.Marshal(map[string]int{"rating": 5})
	req := httptest.NewRequest(http.MethodPost, "/announcement/ann-004/rating", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	r := mux.NewRouter()
	r.HandleFunc("/announcement/{id}/rating", handler.UpdateRating).Methods(http.MethodPost)
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if repo.lastUpdateRatingID != "ann-004" {
		t.Errorf("expected UpdateRating called with id=\"ann-004\", got %q", repo.lastUpdateRatingID)
	}
	if repo.lastUpdateRatingValue != 5 {
		t.Errorf("expected UpdateRating called with rate=5, got %d", repo.lastUpdateRatingValue)
	}

	var got repoAnn.Announcement
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if got.ID != updatedAnn.ID || got.Rating != updatedAnn.Rating {
		t.Errorf("unexpected announcement in response: %+v", got)
	}
}
