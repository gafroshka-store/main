package analytics

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"gafroshka-main/internal/kafka"
)

// fakeRepo нужен для «подмены» AnalyticsRepo в тестах.
type fakeRepo struct {
	called      bool
	lastUserID  string
	lastWeights map[int]int
	// можно добавлять флаги, чтобы «симулировать» ошибку
	returnErr error
}

func (f *fakeRepo) UpdatePreferences(ctx context.Context, userID string, weights map[int]int) error {
	f.called = true
	f.lastUserID = userID
	// копируем map, чтобы избежать мутирования извне
	f.lastWeights = make(map[int]int)
	for k, v := range weights {
		f.lastWeights[k] = v
	}
	return f.returnErr
}

func (f *fakeRepo) GetTopCategories(ctx context.Context, userID string, limit int) ([]int, error) {
	// не требуется для тестирования ProcessEvent
	return nil, nil
}

func TestService_ProcessEvent_EmptyUserID(t *testing.T) {
	repo := &fakeRepo{}
	logger := zapTestLogger(t)
	service := NewService(repo, logger)

	ctx := context.Background()
	evt := kafka.Event{
		UserID:     "", // пустой user
		Type:       kafka.EventTypeView,
		Categories: []int{1, 2},
	}

	err := service.ProcessEvent(ctx, evt)
	if err != nil {
		t.Errorf("expected no error when userID is empty, got %v", err)
	}
	if repo.called {
		t.Errorf("expected repo.UpdatePreferences NOT to be called when userID is empty")
	}
}

func TestService_ProcessEvent_SearchEvent(t *testing.T) {
	repo := &fakeRepo{}
	logger := zapTestLogger(t)
	service := NewService(repo, logger)

	ctx := context.Background()
	evt := kafka.Event{
		UserID:     "u-1",
		Type:       kafka.EventTypeSearch,
		Categories: []int{3, 3, 5},
	}

	err := service.ProcessEvent(ctx, evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.called {
		t.Fatalf("expected repo.UpdatePreferences to be called")
	}
	if repo.lastUserID != "u-1" {
		t.Errorf("expected userID \"u-1\", got %s", repo.lastUserID)
	}
	expectedWeights := map[int]int{
		3: 2, // две встречи категории 3 → 2 * 1
		5: 1, // одна встреча категории 5 → 1 * 1
	}
	if !reflect.DeepEqual(repo.lastWeights, expectedWeights) {
		t.Errorf("expected weights %v, got %v", expectedWeights, repo.lastWeights)
	}
}

func TestService_ProcessEvent_ViewEvent(t *testing.T) {
	repo := &fakeRepo{}
	logger := zapTestLogger(t)
	service := NewService(repo, logger)

	ctx := context.Background()
	evt := kafka.Event{
		UserID:     "u-2",
		Type:       kafka.EventTypeView,
		Categories: []int{7, 9},
	}

	err := service.ProcessEvent(ctx, evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.called {
		t.Fatalf("expected repo.UpdatePreferences to be called")
	}
	// для VIEW учитывается только первая категория, вес = 2
	expectedWeights := map[int]int{
		7: 2,
	}
	if !reflect.DeepEqual(repo.lastWeights, expectedWeights) {
		t.Errorf("expected weights %v, got %v", expectedWeights, repo.lastWeights)
	}
}

func TestService_ProcessEvent_PurchaseEvent(t *testing.T) {
	repo := &fakeRepo{}
	logger := zapTestLogger(t)
	service := NewService(repo, logger)

	ctx := context.Background()
	evt := kafka.Event{
		UserID:     "u-3",
		Type:       kafka.EventTypePurchase,
		Categories: []int{4},
	}

	err := service.ProcessEvent(ctx, evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.called {
		t.Fatalf("expected repo.UpdatePreferences to be called")
	}
	// для PURCHASE учитывается только первая категория, вес = 3
	expectedWeights := map[int]int{
		4: 3,
	}
	if !reflect.DeepEqual(repo.lastWeights, expectedWeights) {
		t.Errorf("expected weights %v, got %v", expectedWeights, repo.lastWeights)
	}
}

func TestService_ProcessEvent_NoCategories(t *testing.T) {
	repo := &fakeRepo{}
	logger := zapTestLogger(t)
	service := NewService(repo, logger)

	ctx := context.Background()
	evt := kafka.Event{
		UserID:     "u-4",
		Type:       kafka.EventTypeView,
		Categories: []int{}, // отсутствуют категории
	}

	err := service.ProcessEvent(ctx, evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.called {
		t.Errorf("expected repo.UpdatePreferences NOT to be called when no categories")
	}
}

func TestService_ProcessEvent_RepoError(t *testing.T) {
	repo := &fakeRepo{returnErr: errors.New("db error")}
	logger := zapTestLogger(t)
	service := NewService(repo, logger)

	ctx := context.Background()
	evt := kafka.Event{
		UserID:     "u-5",
		Type:       kafka.EventTypeSearch,
		Categories: []int{2},
	}

	err := service.ProcessEvent(ctx, evt)
	if err == nil {
		t.Errorf("expected error from repo, got nil")
	}
}
