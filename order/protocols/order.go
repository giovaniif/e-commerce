package protocols

import "context"

type OrderGateway interface {
	SaveOrder(ctx context.Context, idempotencyKey string, itemId int32, quantity int32) error
}
