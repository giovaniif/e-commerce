package protocols

import "context"

type CheckoutIdempotencyKeyResult struct {
	Success bool
	Error   error
}

type CheckoutGateway interface {
	ReserveIdempotencyKey(ctx context.Context, idempotencyKey string) (*CheckoutIdempotencyKeyResult, error)
	MarkFailure(ctx context.Context, idempotencyKey string) error
	MarkSuccess(ctx context.Context, idempotencyKey string) error
}
