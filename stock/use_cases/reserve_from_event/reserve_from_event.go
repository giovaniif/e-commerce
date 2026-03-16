package reserve_from_event

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/giovaniif/e-commerce/stock/domain/item"
	"github.com/giovaniif/e-commerce/stock/infra/repositories"
)

type ReserveFromEvent struct {
	itemRepository item.Repository
}

func NewReserveFromEvent(itemRepository item.Repository) *ReserveFromEvent {
	return &ReserveFromEvent{itemRepository: itemRepository}
}

type orderCreatedPayload struct {
	OrderID        string `json:"order_id"`
	IdempotencyKey string `json:"idempotency_key"`
	ItemID         int32  `json:"item_id"`
	Quantity       int32  `json:"quantity"`
	Traceparent    string `json:"traceparent"`
}

func (r *ReserveFromEvent) Handle(ctx context.Context, eventType string, payload []byte, traceparent string) error {
	carrier := propagation.MapCarrier{"traceparent": traceparent}
	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)
	ctx, span := otel.Tracer("stock").Start(ctx, "consume."+eventType)
	defer span.End()

	var event orderCreatedPayload
	if err := json.Unmarshal(payload, &event); err != nil {
		return err
	}

	it, err := r.itemRepository.GetItem(event.ItemID)
	if err != nil {
		return err
	}

	tp := traceparent
	if event.Traceparent != "" {
		tp = event.Traceparent
	}

	_, err = r.itemRepository.Reserve(it, event.Quantity, event.OrderID, event.IdempotencyKey, tp)
	if err != nil {
		if errors.Is(err, repositories.ErrInsufficientStock) {
			slog.WarnContext(ctx, "insufficient stock for order",
				"order_id", event.OrderID, "item_id", event.ItemID, "quantity", event.Quantity)
			return nil
		}
		if errors.Is(err, repositories.ErrAlreadyProcessed) {
			slog.InfoContext(ctx, "reserve already processed (idempotent)",
				"order_id", event.OrderID, "idempotency_key", event.IdempotencyKey)
			return nil
		}
		return err
	}
	return nil
}
