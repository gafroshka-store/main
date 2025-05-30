package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type Producer struct {
	Writer WriterInterface // Используем интерфейс
	Logger *zap.SugaredLogger
}

func NewProducer(brokers []string, topic string, logger *zap.SugaredLogger) *Producer {
	return &Producer{
		Writer: &kafkaWriterWrapper{ // Обёртка над реальным Writer
			Writer: &kafka.Writer{
				Addr:     kafka.TCP(brokers...),
				Topic:    topic,
				Balancer: &kafka.LeastBytes{},
			},
		},
		Logger: logger,
	}
}

// Обёртка для реализации интерфейса
type kafkaWriterWrapper struct {
	Writer *kafka.Writer
}

func (w *kafkaWriterWrapper) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	return w.Writer.WriteMessages(ctx, msgs...)
}

func (w *kafkaWriterWrapper) Close() error {
	return w.Writer.Close()
}

func (p *Producer) SendEvent(ctx context.Context, event Event) error {
	value, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	err = p.Writer.WriteMessages(ctx, kafka.Message{
		Value: value,
	})

	if err != nil {
		p.Logger.Errorf("Failed to write Kafka message: %v", err)
		return err
	}

	return nil
}

func (p *Producer) Close() error {
	return p.Writer.Close()
}
