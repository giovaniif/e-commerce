package release_from_event

import (
	"context"
	"encoding/json"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/giovaniif/e-commerce/stock/domain/item"
)

type ReleaseFromEvent struct {
	itemRepository item.Repository
}

func NewReleaseFromEvent(itemRepository item.Repository) *ReleaseFromEvent {
	return &ReleaseFromEvent{itemRepository: itemRepository}
}

type paymentProcessedPayload struct {
	OrderID       string  `json:"order_id"`
	ReservationID int64   `json:"reservation_id"`
	Success       bool    `json:"success"`
	FailureReason string  `json:"failure_reason"`
	Traceparent   string  `json:"traceparent"`
}

func (r *ReleaseFromEvent) Handle(ctx context.Context, eventType string, payload []byte, traceparent string) error {
	carrier := propagation.MapCarrier{"traceparent": traceparent}
	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)
	ctx, span := otel.Tracer("stock").Start(ctx, "consume."+eventType)
	defer span.End()

	var event paymentProcessedPayload
	if err := json.Unmarshal(payload, &event); err != nil {
		return err
	}

	if event.Success {
		return nil
	}

	tp := traceparent
	if event.Traceparent != "" {
		tp = event.Traceparent
	}

	slog.InfoContext(ctx, "releasing stock due to payment failure",
		"order_id", event.OrderID, "reservation_id", event.ReservationID, "reason", event.FailureReason)

	return r.itemRepository.ReleaseReservation(int32(event.ReservationID), tp)
}
