package kafka

import (
	"context"
	"github.com/segmentio/kafka-go"
)

// ReaderInterface интерфейс для Kafka Reader
type ReaderInterface interface {
	ReadMessage(ctx context.Context) (kafka.Message, error)
	Close() error
}

// WriterInterface интерфейс для Kafka Writer
type WriterInterface interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

type EventProducer interface {
	SendEvent(ctx context.Context, event Event) error
	Close() error
}
