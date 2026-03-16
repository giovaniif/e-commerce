package checkout

import (
	"context"
	"encoding/json"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/giovaniif/e-commerce/order/infra/repositories"
)

type FinalizeOrder struct {
	orderRepo *repositories.OrderRepository
}

func NewFinalizeOrder(orderRepo *repositories.OrderRepository) *FinalizeOrder {
	return &FinalizeOrder{orderRepo: orderRepo}
}

type paymentProcessedPayload struct {
	OrderID        string `json:"order_id"`
	IdempotencyKey string `json:"idempotency_key"`
	Success        bool   `json:"success"`
	FailureReason  string `json:"failure_reason"`
	Traceparent    string `json:"traceparent"`
}

func (f *FinalizeOrder) Handle(ctx context.Context, eventType string, payload []byte, traceparent string) error {
	tp := traceparent
	carrier := propagation.MapCarrier{"traceparent": tp}
	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)
	ctx, span := otel.Tracer("order").Start(ctx, "consume."+eventType)
	defer span.End()

	var event paymentProcessedPayload
	if err := json.Unmarshal(payload, &event); err != nil {
		return err
	}

	status := "completed"
	if !event.Success {
		status = "failed"
		slog.WarnContext(ctx, "payment failed for order",
			"order_id", event.OrderID,
			"idempotency_key", event.IdempotencyKey,
			"reason", event.FailureReason)
	} else {
		slog.InfoContext(ctx, "payment completed for order",
			"order_id", event.OrderID,
			"idempotency_key", event.IdempotencyKey)
	}

	return f.orderRepo.UpdateStatus(ctx, event.IdempotencyKey, status)
}
