package checkout

import (
	"context"
	"errors"
	"testing"
	"time"

	protocols "github.com/giovaniif/e-commerce/order/protocols"
)

type mockCheckoutGateway struct {
	reserveIdempotencyKeyResult *protocols.CheckoutIdempotencyKeyResult
	reserveIdempotencyKeyErr    error
	markSuccessCalled           bool
	markFailureCalled           bool
	markSuccessKey              string
	markFailureKey              string
}

func (m *mockCheckoutGateway) ReserveIdempotencyKey(ctx context.Context, idempotencyKey string) (*protocols.CheckoutIdempotencyKeyResult, error) {
	return m.reserveIdempotencyKeyResult, m.reserveIdempotencyKeyErr
}

func (m *mockCheckoutGateway) MarkSuccess(ctx context.Context, idempotencyKey string) error {
	m.markSuccessCalled = true
	m.markSuccessKey = idempotencyKey
	return nil
}

func (m *mockCheckoutGateway) MarkFailure(ctx context.Context, idempotencyKey string) error {
	m.markFailureCalled = true
	m.markFailureKey = idempotencyKey
	return nil
}

// TestCheckoutWithExistingIdempotencyKey verifies that a previously completed checkout
// short-circuits without touching the DB and without calling MarkSuccess/MarkFailure.
func TestCheckoutWithExistingIdempotencyKey(t *testing.T) {
	checkoutGateway := &mockCheckoutGateway{
		reserveIdempotencyKeyResult: &protocols.CheckoutIdempotencyKeyResult{Success: true},
	}
	// nil repos/db: the early-return path never reaches DB code.
	uc := NewCheckout(checkoutGateway, nil, nil, nil)

	err := uc.Checkout(context.Background(), Input{ItemId: 1, Quantity: 2, IdempotencyKey: "abc-123"})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if checkoutGateway.markSuccessCalled || checkoutGateway.markFailureCalled {
		t.Fatalf("expected neither MarkSuccess nor MarkFailure on already-processed key")
	}
}

// TestCheckoutWithProcessingIdempotencyKey verifies that an error from ReserveIdempotencyKey
// is propagated and no DB is touched.
func TestCheckoutWithProcessingIdempotencyKey(t *testing.T) {
	checkoutGateway := &mockCheckoutGateway{
		reserveIdempotencyKeyErr: errors.New("idempotency key is already being processed"),
	}
	uc := NewCheckout(checkoutGateway, nil, nil, nil)

	err := uc.Checkout(context.Background(), Input{ItemId: 1, Quantity: 2, IdempotencyKey: "xyz-456"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err.Error() != "idempotency key is already being processed" {
		t.Fatalf("unexpected error: %v", err)
	}
	if checkoutGateway.markSuccessCalled || checkoutGateway.markFailureCalled {
		t.Fatalf("expected neither MarkSuccess nor MarkFailure when idempotency returns error")
	}
}

// TestCheckoutContextError verifies that an expired context returns DeadlineExceeded immediately.
func TestCheckoutContextError(t *testing.T) {
	checkoutGateway := &mockCheckoutGateway{}
	uc := NewCheckout(checkoutGateway, nil, nil, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	err := uc.Checkout(ctx, Input{ItemId: 1, Quantity: 2, IdempotencyKey: "context-error"})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}
	if checkoutGateway.markSuccessCalled || checkoutGateway.markFailureCalled {
		t.Fatalf("expected neither MarkSuccess nor MarkFailure when context expired")
	}
}
