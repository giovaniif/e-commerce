package kafka

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/giovaniif/e-commerce/payment/protocols"
)

type Consumer struct {
	reader  *kafka.Reader
	handler protocols.EventHandler
}

func NewConsumer(brokers []string, topic, groupID string, handler protocols.EventHandler) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       topic,
		GroupID:     groupID,
		MinBytes:    1,
		MaxBytes:    10e6,
		MaxWait:     500 * time.Millisecond,
	})
	return &Consumer{reader: reader, handler: handler}
}

func (c *Consumer) Run(ctx context.Context) {
	dlqTopic := c.reader.Config().Topic + "_DLQ"
	dlqWriter := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  c.reader.Config().Brokers,
		Topic:    dlqTopic,
		Balancer: &kafka.LeastBytes{},
	})
	defer dlqWriter.Close()

	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("kafka fetch message", "topic", c.reader.Config().Topic, "error", err)
			continue
		}

		traceparent := extractHeader(msg.Headers, "traceparent")

		var lastErr error
		for attempt := 0; attempt < 3; attempt++ {
			if err := c.handler.Handle(ctx, msg.Topic, msg.Value, traceparent); err != nil {
				lastErr = err
				if attempt < 2 {
					delay := time.Duration(math.Pow(2, float64(attempt))) * 100 * time.Millisecond
					time.Sleep(delay)
				}
				continue
			}
			lastErr = nil
			break
		}

		if lastErr != nil {
			slog.Error("kafka handler failed after retries, sending to DLQ",
				"topic", msg.Topic, "error", lastErr)
			_ = dlqWriter.WriteMessages(ctx, kafka.Message{
				Key:   msg.Key,
				Value: msg.Value,
			})
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			slog.Error("kafka commit message", "error", err)
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}

func extractHeader(headers []kafka.Header, key string) string {
	for _, h := range headers {
		if h.Key == key {
			return fmt.Sprintf("%s", h.Value)
		}
	}
	return ""
}
