package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// fakeWriter реализует WriterInterface и просто запоминает, какие сообщения ему передали.
type fakeWriter struct {
	lastMessages []kafka.Message
	returnError  error
}

func (f *fakeWriter) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	// Запоминаем все пришедшие сообщения
	f.lastMessages = append(f.lastMessages, msgs...)
	return f.returnError
}

func (f *fakeWriter) Close() error {
	return nil
}

func zapTestLogger(t *testing.T) *zap.SugaredLogger {
	t.Helper()
	logger, err := zap.NewDevelopmentConfig().Build(zap.AddCallerSkip(1))
	if err != nil {
		t.Fatalf("не удалось создать zap-логгер: %v", err)
	}
	return logger.Sugar()
}

func TestProducer_SendEvent_Success(t *testing.T) {
	// Подготовка «тихого» логгера
	logger := zapTestLogger(t)
	defer func() { _ = logger.Sync() }()

	// Подменяем Writer на fakeWriter
	fw := &fakeWriter{returnError: nil}
	p := &Producer{
		Writer: fw,
		Logger: logger,
	}

	ctx := context.Background()
	evt := Event{
		UserID:     "user1",
		Type:       EventTypePurchase,
		Categories: []int{1, 2},
		Timestamp:  time.Now().UTC(),
	}

	// Выполняем SendEvent
	if err := p.SendEvent(ctx, evt); err != nil {
		t.Fatalf("ожидали, что SendEvent не вернёт ошибку, но получили: %v", err)
	}

	// Проверяем, что записалось ровно одно сообщение
	if len(fw.lastMessages) != 1 {
		t.Fatalf("ожидали 1 записанное сообщение, но получили %d", len(fw.lastMessages))
	}

	// Разбираем Value из сообщения и сравниваем с исходным Event
	var decoded Event
	if err := json.Unmarshal(fw.lastMessages[0].Value, &decoded); err != nil {
		t.Fatalf("не удалось разобрать записанное сообщение как JSON: %v", err)
	}
	if decoded.UserID != evt.UserID {
		t.Errorf("разобранный UserID не совпал: ожидали %q, получили %q", evt.UserID, decoded.UserID)
	}
	if decoded.Type != evt.Type {
		t.Errorf("разобранный EventType не совпал: ожидали %q, получили %q", evt.Type, decoded.Type)
	}
	// Проверим хотя бы одну категорию
	if len(decoded.Categories) != len(evt.Categories) {
		t.Errorf("размер среза Categories не совпал: ожидали %d, получили %d", len(evt.Categories), len(decoded.Categories))
	}
}

func TestProducer_SendEvent_WriteError(t *testing.T) {
	logger := zapTestLogger(t)
	defer func() { _ = logger.Sync() }()

	// fakeWriter сконфигурирован так, чтобы возвращать ошибку при записи
	fw := &fakeWriter{returnError: errors.New("write failed")}
	p := &Producer{
		Writer: fw,
		Logger: logger,
	}

	ctx := context.Background()
	evt := Event{
		UserID:     "user2",
		Type:       EventTypeView,
		Categories: []int{5},
		Timestamp:  time.Now().UTC(),
	}

	// Ожидаем, что SendEvent вернёт ошибку, потому что fakeWriter.returnError != nil
	if err := p.SendEvent(ctx, evt); err == nil {
		t.Fatalf("ожидали ошибку от SendEvent, но получили nil")
	}
}
