package analytics

import (
	"context"
	"go.uber.org/zap"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// Тест UpdatePreferences: проверяем, что для каждой категории выполняется INSERT ... ON CONFLICT ...,
// и транзакция корректно коммитится.
func TestRepository_UpdatePreferences(t *testing.T) {
	// Создаём sqlmock
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error when opening a stub database connection: %s", err)
	}
	defer db.Close()

	logger := zapTestLogger(t) // вспомогательная функция ниже
	repo := NewRepository(db, logger)

	ctx := context.Background()
	userID := "user-123"
	weights := map[int]int{
		10: 1,
		20: 3,
	}

	// Ожидаем BEGIN
	mock.ExpectBegin()

	// Для каждой пары category->weight ожидаем ExecContext с нужным SQL и аргументами
	for category, weight := range weights {
		// Паттерн регулярки, чтобы не зависеть от пробелов:
		mock.ExpectExec(regexp.QuoteMeta(`
			INSERT INTO user_preferences (user_id, category, weight)
			VALUES ($1, $2, $3)
			ON CONFLICT (user_id, category)
			DO UPDATE SET weight = user_preferences.weight + EXCLUDED.weight
		`)).
			WithArgs(userID, category, weight).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}

	// Ожидаем коммит
	mock.ExpectCommit()

	// Вызываем
	if err := repo.UpdatePreferences(ctx, userID, weights); err != nil {
		t.Errorf("UpdatePreferences returned unexpected error: %v", err)
	}

	// Проверяем, что все ожидания выполнены
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

// Тест GetTopCategories: проверяем, что возвращаются именно те категории, которые «лежат» в rows.
func TestRepository_GetTopCategories(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error when opening a stub database connection: %s", err)
	}
	defer db.Close()

	logger := zapTestLogger(t)
	repo := NewRepository(db, logger)

	ctx := context.Background()
	userID := "user-123"
	limit := 2

	// Подготавливаем фиктивные строки
	rows := sqlmock.NewRows([]string{"category"}).
		AddRow(5).
		AddRow(7)

	// Ожидаем QueryContext с правильным SQL и аргументами
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT category
		FROM user_preferences
		WHERE user_id = $1
		ORDER BY weight DESC
		LIMIT $2
	`)).
		WithArgs(userID, limit).
		WillReturnRows(rows)

	// Вызываем
	result, err := repo.GetTopCategories(ctx, userID, limit)
	if err != nil {
		t.Fatalf("GetTopCategories returned error: %v", err)
	}

	// Проверяем результат
	expected := []int{5, 7}
	if len(result) != len(expected) {
		t.Fatalf("expected %d items, got %d", len(expected), len(result))
	}
	for i := range expected {
		if result[i] != expected[i] {
			t.Errorf("expected category %d at position %d, got %d", expected[i], i, result[i])
		}
	}

	// Проверяем, что все ожидания выполнены
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

// Вспомогательная функция для создания «тихого» логгера.
// Используем zap.NewNop() или похожую реализацию.
func zapTestLogger(t *testing.T) *zap.SugaredLogger {
	t.Helper()
	logger, err := zap.NewDevelopmentConfig().Build(zap.AddCallerSkip(1))
	if err != nil {
		t.Fatalf("failed to create zap logger: %v", err)
	}
	return logger.Sugar()
}
