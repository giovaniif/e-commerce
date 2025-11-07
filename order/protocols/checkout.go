package protocols

type CheckoutIdempotencyKeyResult struct {
	Success bool
	Error   error
}

type CheckoutGateway interface {
	ReserveIdempotencyKey(idempotencyKey string) (*CheckoutIdempotencyKeyResult, error)
	MarkFailure(idempotencyKey string) error
	MarkSuccess(idempotencyKey string) error
}
