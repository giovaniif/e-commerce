package protocols

type IdempotencyKeyResult struct {
	Success bool
	Error   error
}

type IdempotencyGateway interface {
	ReserveIdempotencyKey(idempotencyKey string) (*IdempotencyKeyResult, error)
	MarkFailure(idempotencyKey string) error
	MarkSuccess(idempotencyKey string) error
}
