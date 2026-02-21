package protocols

import "context"

type PaymentGateway interface {
	Charge(ctx context.Context, amount float64, idempotencyKey string) error
}
