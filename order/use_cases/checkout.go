package checkout

import (
	"context"
	"database/sql"
	"encoding/json"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/giovaniif/e-commerce/order/infra/repositories"
	protocols "github.com/giovaniif/e-commerce/order/protocols"
)

type Checkout struct {
	checkoutGateway protocols.CheckoutGateway
	orderRepo       *repositories.OrderRepository
	outboxRepo      *repositories.OutboxRepository
	db              *sql.DB
}

func NewCheckout(
	checkoutGateway protocols.CheckoutGateway,
	orderRepo *repositories.OrderRepository,
	outboxRepo *repositories.OutboxRepository,
	db *sql.DB,
) *Checkout {
	return &Checkout{
		checkoutGateway: checkoutGateway,
		orderRepo:       orderRepo,
		outboxRepo:      outboxRepo,
		db:              db,
	}
}

func (c *Checkout) Checkout(ctx context.Context, input Input) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	result, err := c.checkoutGateway.ReserveIdempotencyKey(ctx, input.IdempotencyKey)
	if err != nil {
		return err
	}
	if result != nil {
		// Already processed — short-circuit before setting up defer.
		return nil
	}

	success := false
	defer func() {
		if success {
			c.checkoutGateway.MarkSuccess(ctx, input.IdempotencyKey)
		} else {
			c.checkoutGateway.MarkFailure(ctx, input.IdempotencyKey)
		}
	}()

	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	orderId, err := c.orderRepo.CreatePendingTx(ctx, tx, input.IdempotencyKey, input.ItemId, input.Quantity)
	if err != nil {
		return err
	}
	if orderId == "" {
		// Idempotent: order already exists for this key.
		success = true
		return nil
	}

	// Inject current OTel span context into traceparent header.
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	traceparent := carrier.Get("traceparent")

	payload, err := json.Marshal(map[string]interface{}{
		"order_id":        orderId,
		"idempotency_key": input.IdempotencyKey,
		"item_id":         input.ItemId,
		"quantity":        input.Quantity,
		"traceparent":     traceparent,
	})
	if err != nil {
		return err
	}

	if err := c.outboxRepo.InsertTx(ctx, tx, repositories.OutboxEvent{
		AggregateType: "ORDER",
		AggregateID:   orderId,
		Type:          "OrderCreated",
		Payload:       payload,
		Traceparent:   traceparent,
	}); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	success = true
	return nil
}

type Input struct {
	ItemId         int32
	Quantity       int32
	IdempotencyKey string
}
