package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
)

// fakeReader реализует ReaderInterface и отдаёт заранее подготовленные сообщения и ошибки.
type fakeReader struct {
	// messages — список сообщений, которые нужно отдать в порядке индексов.
	messages []kafka.Message
	// errors — ошибки, которые нужно возвращать после того, как закончатся messages.
	// Количество ошибок может быть меньше, чем количество циклов чтения; тогда после исчерпания всех
	// сообщений и всех ошибок вернётся context.Canceled.
	errors []error
	// idx указывает, сколько раз уже вызывался ReadMessage.
	idx int
}

func (f *fakeReader) ReadMessage(ctx context.Context) (kafka.Message, error) {
	// Если ещё есть необработанные сообщения — возвращаем текущее
	if f.idx < len(f.messages) {
		msg := f.messages[f.idx]
		f.idx++
		return msg, nil
	}
	// Если уже отдали все сообщения, но остались ошибки — возвращаем следующую ошибку
	errIdx := f.idx - len(f.messages)
	if errIdx < len(f.errors) {
		err := f.errors[errIdx]
		f.idx++
		return kafka.Message{}, err
	}
	// Иначе — возвращаем context.Canceled, чтобы Consumer.Consume вышел
	return kafka.Message{}, context.Canceled
}

func (f *fakeReader) Close() error {
	return nil
}

func TestConsumer_Consume_ValidEvent(t *testing.T) {
	// Подготовим валидный Event и запишем его в fakeReader.messages
	evt := Event{
		UserID:     "test-user",
		Type:       EventTypeSearch,
		Categories: []int{2, 4},
		Timestamp:  time.Now().UTC(),
	}
	payload, _ := json.Marshal(evt)
	msg := kafka.Message{Value: payload}

	// fakeReader вернёт сначала валидное сообщение, потом вернёт ошибку context.Canceled, чтобы прервать цикл.
	fr := &fakeReader{
		messages: []kafka.Message{msg},
		errors:   []error{context.Canceled},
	}

	logger := zapTestLogger(t)
	consumer := &Consumer{
		Reader: fr,
		Logger: logger,
	}

	var called bool
	var received Event

	// handler запишет, что был вызван, и запомнит сам Event
	handler := func(ctx context.Context, e Event) error {
		called = true
		received = e
		return nil
	}

	consumer.Consume(context.Background(), handler)

	// Проверяем, что handler действительно вызвался один раз
	if !called {
		t.Fatal("ожидали, что handler будет вызван для валидного события")
	}
	// Проверим, что пришёл именно тот Event, который мы сериализовали
	if received.UserID != evt.UserID {
		t.Errorf("ожидали UserID=%q, получили=%q", evt.UserID, received.UserID)
	}
	if received.Type != evt.Type {
		t.Errorf("ожидали Type=%q, получили=%q", evt.Type, received.Type)
	}
	if len(received.Categories) != len(evt.Categories) {
		t.Errorf("ожидали len(Categories)=%d, получили=%d",
			len(evt.Categories), len(received.Categories))
	}
}

func TestConsumer_Consume_InvalidJSON(t *testing.T) {
	// Подготовим сообщение с некорректным JSON
	badMsg := kafka.Message{Value: []byte(`{"user_id": 123, bad json`)}
	fr := &fakeReader{
		messages: []kafka.Message{badMsg},
		errors:   []error{context.Canceled},
	}

	logger := zapTestLogger(t)
	consumer := &Consumer{
		Reader: fr,
		Logger: logger,
	}

	called := false
	handler := func(ctx context.Context, e Event) error {
		called = true
		return nil
	}

	consumer.Consume(context.Background(), handler)

	// При некорректном JSON handler НЕ должен вызываться
	if called {
		t.Error("ожидали, что handler НЕ будет вызван при некорректном JSON")
	}
}

func TestConsumer_Consume_HandlerError(t *testing.T) {
	// Подготовим валидный Event
	evt := Event{
		UserID:     "user-err",
		Type:       EventTypeView,
		Categories: []int{7},
		Timestamp:  time.Now().UTC(),
	}
	payload, _ := json.Marshal(evt)
	msg := kafka.Message{Value: payload}

	// fakeReader вернёт сообщение, а затем context.Canceled, чтобы выйти из цикла
	fr := &fakeReader{
		messages: []kafka.Message{msg},
		errors:   []error{context.Canceled},
	}

	logger := zapTestLogger(t)
	consumer := &Consumer{
		Reader: fr,
		Logger: logger,
	}

	var called bool
	handler := func(ctx context.Context, e Event) error {
		called = true
		// Возвращаем ошибку, чтобы Consumer залогировал её, но сам не паникующий
		return errors.New("simulated handler failure")
	}

	consumer.Consume(context.Background(), handler)

	// Даже если handler вернул ошибку, Consume не должен была «пропустить» вызов
	if !called {
		t.Error("ожидали, что handler всё же будет вызван, даже если он вернул ошибку")
	}
}
