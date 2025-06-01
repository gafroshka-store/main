package kafka

import (
	"context"
	"encoding/json"
	"errors"

	kgo "github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// Consumer реализует EventConsumer.
type Consumer struct {
	Reader ReaderInterface
	Logger *zap.SugaredLogger
}

func NewConsumer(brokers, topic, groupID string, logger *zap.SugaredLogger) EventConsumer {
	return &Consumer{
		Reader: &kafkaReaderWrapper{
			Reader: kgo.NewReader(kgo.ReaderConfig{
				Brokers:  []string{brokers},
				Topic:    topic,
				GroupID:  groupID,
				MinBytes: 10e3, // 10KB
				MaxBytes: 10e6, // 10MB
			}),
		},
		Logger: logger,
	}
}

type kafkaReaderWrapper struct {
	Reader *kgo.Reader
}

func (w *kafkaReaderWrapper) ReadMessage(ctx context.Context) (kgo.Message, error) {
	return w.Reader.ReadMessage(ctx)
}

func (w *kafkaReaderWrapper) Close() error {
	return w.Reader.Close()
}

func (c *Consumer) Consume(ctx context.Context, handler func(context.Context, Event) error) {
	for {
		msg, err := c.Reader.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			c.Logger.Errorf("Failed to read message: %v", err)
			continue
		}

		var event Event
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			c.Logger.Errorf("Failed to unmarshal event: %v", err)
			continue
		}

		if err := handler(ctx, event); err != nil {
			c.Logger.Errorf("Failed to process event: %v", err)
		}
	}
}

func (c *Consumer) Close() error {
	return c.Reader.Close()
}
