package charge

import (
	"errors"
	"testing"

	protocols "github.com/giovaniif/e-commerce/payment/protocols"
)

type mockChargeGateway struct {
	charged   []float64
	chargeErr error
}

func (m *mockChargeGateway) Charge(amount float64) error {
	m.charged = append(m.charged, amount)
	return m.chargeErr
}

type mockIdempotencyGateway struct {
	reserveIdempotencyKeyResult *protocols.IdempotencyKeyResult
	reserveIdempotencyKeyErr    error
	markSuccessCalled           bool
	markFailureCalled           bool
	markSuccessKey              string
	markFailureKey              string
}

func (m *mockIdempotencyGateway) ReserveIdempotencyKey(idempotencyKey string) (*protocols.IdempotencyKeyResult, error) {
	return m.reserveIdempotencyKeyResult, m.reserveIdempotencyKeyErr
}

func (m *mockIdempotencyGateway) MarkSuccess(idempotencyKey string) error {
	m.markSuccessCalled = true
	m.markSuccessKey = idempotencyKey
	return nil
}

func (m *mockIdempotencyGateway) MarkFailure(idempotencyKey string) error {
	m.markFailureCalled = true
	m.markFailureKey = idempotencyKey
	return nil
}

func TestChargeSuccess(t *testing.T) {
	chargeGateway := &mockChargeGateway{}
	idempotencyGateway := &mockIdempotencyGateway{}
	uc := NewCharge(chargeGateway, idempotencyGateway)

	err := uc.Charge(ChargeInput{
		Amount:         100.50,
		IdempotencyKey: "key-1",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(chargeGateway.charged) != 1 {
		t.Fatalf("expected Charge to be called once, got %d", len(chargeGateway.charged))
	}
	if chargeGateway.charged[0] != 100.50 {
		t.Fatalf("expected Charge amount 100.50, got %v", chargeGateway.charged[0])
	}
	if !idempotencyGateway.markSuccessCalled {
		t.Fatalf("expected MarkSuccess to be called")
	}
	if idempotencyGateway.markSuccessKey != "key-1" {
		t.Fatalf("expected MarkSuccess called with key 'key-1', got %s", idempotencyGateway.markSuccessKey)
	}
}

func TestChargeWithGatewayError(t *testing.T) {
	chargeGateway := &mockChargeGateway{chargeErr: errors.New("charge gateway error")}
	idempotencyGateway := &mockIdempotencyGateway{}
	uc := NewCharge(chargeGateway, idempotencyGateway)

	err := uc.Charge(ChargeInput{
		Amount:         200.75,
		IdempotencyKey: "key-2",
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err.Error() != "charge gateway error" {
		t.Fatalf("expected error 'charge gateway error', got %v", err)
	}
	if len(chargeGateway.charged) != 1 {
		t.Fatalf("expected Charge to be called once, got %d", len(chargeGateway.charged))
	}
	if !idempotencyGateway.markFailureCalled {
		t.Fatalf("expected MarkFailure to be called on gateway error")
	}
	if idempotencyGateway.markFailureKey != "key-2" {
		t.Fatalf("expected MarkFailure called with key 'key-2', got %s", idempotencyGateway.markFailureKey)
	}
}

func TestChargeWithSuccessfulIdempotencyKey(t *testing.T) {
	chargeGateway := &mockChargeGateway{}
	idempotencyGateway := &mockIdempotencyGateway{
		reserveIdempotencyKeyResult: &protocols.IdempotencyKeyResult{
			Success: true,
			Error:   nil,
		},
	}
	uc := NewCharge(chargeGateway, idempotencyGateway)

	err := uc.Charge(ChargeInput{
		Amount:         300.00,
		IdempotencyKey: "key-3",
	})
	if err != nil {
		t.Fatalf("expected nil error when idempotency key already succeeded, got %v", err)
	}
	if len(chargeGateway.charged) != 0 {
		t.Fatalf("expected Charge not to be called when idempotency key already succeeded, got %d calls", len(chargeGateway.charged))
	}
	if idempotencyGateway.markSuccessCalled {
		t.Fatalf("expected MarkSuccess not to be called when idempotency key already succeeded")
	}
	if idempotencyGateway.markFailureCalled {
		t.Fatalf("expected MarkFailure not to be called when idempotency key already succeeded")
	}
}

func TestChargeWithProcessingIdempotencyKey(t *testing.T) {
	chargeGateway := &mockChargeGateway{}
	idempotencyGateway := &mockIdempotencyGateway{
		reserveIdempotencyKeyErr: errors.New("idempotency key is already being processed"),
	}
	uc := NewCharge(chargeGateway, idempotencyGateway)

	err := uc.Charge(ChargeInput{
		Amount:         400.25,
		IdempotencyKey: "key-4",
	})
	if err == nil {
		t.Fatalf("expected error when idempotency key is processing, got nil")
	}
	if err.Error() != "idempotency key is already being processed" {
		t.Fatalf("expected error message 'idempotency key is already being processed', got %v", err)
	}
	if len(chargeGateway.charged) != 0 {
		t.Fatalf("expected Charge not to be called when idempotency key is processing, got %d calls", len(chargeGateway.charged))
	}
	if idempotencyGateway.markSuccessCalled {
		t.Fatalf("expected MarkSuccess not to be called when idempotency key is processing")
	}
	if idempotencyGateway.markFailureCalled {
		t.Fatalf("expected MarkFailure not to be called when idempotency key is processing")
	}
}

func TestChargeMarkFailureOnGatewayError(t *testing.T) {
	chargeGateway := &mockChargeGateway{chargeErr: errors.New("payment failed")}
	idempotencyGateway := &mockIdempotencyGateway{}
	uc := NewCharge(chargeGateway, idempotencyGateway)

	err := uc.Charge(ChargeInput{
		Amount:         500.00,
		IdempotencyKey: "key-5",
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !idempotencyGateway.markFailureCalled {
		t.Fatalf("expected MarkFailure to be called on gateway error")
	}
	if idempotencyGateway.markFailureKey != "key-5" {
		t.Fatalf("expected MarkFailure called with key 'key-5', got %s", idempotencyGateway.markFailureKey)
	}
}

func TestChargeMarkSuccessOnCompleteSuccess(t *testing.T) {
	chargeGateway := &mockChargeGateway{}
	idempotencyGateway := &mockIdempotencyGateway{}
	uc := NewCharge(chargeGateway, idempotencyGateway)

	err := uc.Charge(ChargeInput{
		Amount:         600.50,
		IdempotencyKey: "key-6",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !idempotencyGateway.markSuccessCalled {
		t.Fatalf("expected MarkSuccess to be called on successful charge")
	}
	if idempotencyGateway.markSuccessKey != "key-6" {
		t.Fatalf("expected MarkSuccess called with key 'key-6', got %s", idempotencyGateway.markSuccessKey)
	}
	if idempotencyGateway.markFailureCalled {
		t.Fatalf("expected MarkFailure not to be called on successful charge")
	}
}

func TestChargeWithDifferentAmounts(t *testing.T) {
	chargeGateway := &mockChargeGateway{}
	idempotencyGateway := &mockIdempotencyGateway{}
	uc := NewCharge(chargeGateway, idempotencyGateway)

	testCases := []struct {
		name   string
		amount float64
		key    string
	}{
		{"zero amount", 0.0, "key-zero"},
		{"small amount", 0.01, "key-small"},
		{"large amount", 999999.99, "key-large"},
		{"decimal amount", 123.45, "key-decimal"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			chargeGateway.charged = []float64{}
			idempotencyGateway.markSuccessCalled = false
			idempotencyGateway.markFailureCalled = false

			err := uc.Charge(ChargeInput{
				Amount:         tc.amount,
				IdempotencyKey: tc.key,
			})
			if err != nil {
				t.Fatalf("expected nil error for %s, got %v", tc.name, err)
			}
			if len(chargeGateway.charged) != 1 {
				t.Fatalf("expected Charge to be called once for %s, got %d", tc.name, len(chargeGateway.charged))
			}
			if chargeGateway.charged[0] != tc.amount {
				t.Fatalf("expected Charge amount %v for %s, got %v", tc.amount, tc.name, chargeGateway.charged[0])
			}
		})
	}
}
