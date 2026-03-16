package charge_from_event

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/giovaniif/e-commerce/payment/infra/repositories"
	"github.com/giovaniif/e-commerce/payment/protocols"
)

type ChargeFromEvent struct {
	chargeGateway protocols.ChargeGateway
	outboxRepo    *repositories.OutboxRepository
	processedRepo *repositories.ProcessedEventsRepository
	db            *sql.DB
}

func NewChargeFromEvent(
	chargeGateway protocols.ChargeGateway,
	outboxRepo *repositories.OutboxRepository,
	processedRepo *repositories.ProcessedEventsRepository,
	db *sql.DB,
) *ChargeFromEvent {
	return &ChargeFromEvent{
		chargeGateway: chargeGateway,
		outboxRepo:    outboxRepo,
		processedRepo: processedRepo,
		db:            db,
	}
}

type stockReservedPayload struct {
	OrderID        string  `json:"order_id"`
	IdempotencyKey string  `json:"idempotency_key"`
	ItemID         int32   `json:"item_id"`
	Quantity       int32   `json:"quantity"`
	ReservationID  int64   `json:"reservation_id"`
	TotalFee       float64 `json:"total_fee"`
	Traceparent    string  `json:"traceparent"`
}

func (c *ChargeFromEvent) Handle(ctx context.Context, eventType string, payload []byte, traceparent string) error {
	tp := traceparent
	carrier := propagation.MapCarrier{"traceparent": tp}
	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)
	ctx, span := otel.Tracer("payment").Start(ctx, "consume."+eventType)
	defer span.End()

	var event stockReservedPayload
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("unmarshal StockReserved: %w", err)
	}
	if event.Traceparent != "" {
		tp = event.Traceparent
	}

	// Deduplication using order_id as the event key (at most one StockReserved per order).
	eventID, err := repositories.ParseEventID(event.OrderID)
	if err != nil {
		return fmt.Errorf("parse order_id as UUID: %w", err)
	}

	already, err := c.processedRepo.IsProcessed(ctx, eventID)
	if err != nil {
		return fmt.Errorf("check processed: %w", err)
	}
	if already {
		slog.InfoContext(ctx, "payment already processed for order", "order_id", event.OrderID)
		return nil
	}

	// Attempt charge
	chargeErr := c.chargeGateway.Charge(event.TotalFee)

	success := chargeErr == nil
	failureReason := ""
	if !success {
		failureReason = chargeErr.Error()
		slog.WarnContext(ctx, "charge failed", "order_id", event.OrderID, "error", chargeErr)
	}

	outboxPayload, _ := json.Marshal(map[string]interface{}{
		"order_id":        event.OrderID,
		"idempotency_key": event.IdempotencyKey,
		"reservation_id":  event.ReservationID,
		"amount":          event.TotalFee,
		"success":         success,
		"failure_reason":  failureReason,
		"traceparent":     tp,
	})

	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := c.processedRepo.MarkProcessedTx(ctx, tx, eventID); err != nil {
		return fmt.Errorf("mark processed: %w", err)
	}

	if err := c.outboxRepo.InsertTx(ctx, tx, repositories.OutboxEvent{
		AggregateType: "PAYMENT",
		AggregateID:   event.OrderID,
		Type:          "PaymentProcessed",
		Payload:       outboxPayload,
		Traceparent:   tp,
	}); err != nil {
		return fmt.Errorf("insert outbox PaymentProcessed: %w", err)
	}

	return tx.Commit()
}
