package checkout

import (
	"errors"
	"testing"
	"time"

	protocols "github.com/giovaniif/e-commerce/order/protocols"
)

type mockStockGateway struct {
	reservedInputs []struct{ itemId, quantity int32 }
	reserveResult  *protocols.Reservation
	reserveErr     error
	releasedIds    []int32
	releaseErr     error
	completedIds   []int32
	completeErr    error
}

func (m *mockStockGateway) Reserve(itemId int32, quantity int32) (*protocols.Reservation, error) {
	m.reservedInputs = append(m.reservedInputs, struct{ itemId, quantity int32 }{itemId, quantity})
	return m.reserveResult, m.reserveErr
}

func (m *mockStockGateway) Release(reservationId int32) error {
	m.releasedIds = append(m.releasedIds, reservationId)
	return m.releaseErr
}

func (m *mockStockGateway) Complete(reservationId int32) error {
	m.completedIds = append(m.completedIds, reservationId)
	return m.completeErr
}

type mockPaymentGateway struct {
	charged   []float64
	chargeErr error
}

func (m *mockPaymentGateway) Charge(amount float64) error {
	m.charged = append(m.charged, amount)
	return m.chargeErr
}

type mockCheckoutGateway struct {
	reserveIdempotencyKeyResult *protocols.CheckoutIdempotencyKeyResult
	reserveIdempotencyKeyErr    error
	markSuccessCalled           bool
	markFailureCalled           bool
	markSuccessKey              string
	markFailureKey              string
}

func (m *mockCheckoutGateway) ReserveIdempotencyKey(idempotencyKey string) (*protocols.CheckoutIdempotencyKeyResult, error) {
	return m.reserveIdempotencyKeyResult, m.reserveIdempotencyKeyErr
}

func (m *mockCheckoutGateway) MarkSuccess(idempotencyKey string) error {
	m.markSuccessCalled = true
	m.markSuccessKey = idempotencyKey
	return nil
}

func (m *mockCheckoutGateway) MarkFailure(idempotencyKey string) error {
	m.markFailureCalled = true
	m.markFailureKey = idempotencyKey
	return nil
}

type MockSleeper struct{}

func (m *MockSleeper) Sleep(duration time.Duration) {
	// no-op
}

func TestCheckoutReserveError(t *testing.T) {
	stock := &mockStockGateway{reserveErr: errors.New("reserve error")}
	payment := &mockPaymentGateway{}
	checkoutGateway := &mockCheckoutGateway{}
	sleeper := &MockSleeper{}
	uc := NewCheckout(stock, payment, checkoutGateway, sleeper)

	err := uc.Checkout(Input{ItemId: 1, Quantity: 2, IdempotencyKey: "123"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if len(stock.reservedInputs) != 5 {
		t.Fatalf("expected Reserve to be called once, got %d", len(stock.reservedInputs))
	}
	if !checkoutGateway.markFailureCalled {
		t.Fatalf("expected MarkFailure to be called")
	}
	if checkoutGateway.markFailureKey != "123" {
		t.Fatalf("expected MarkFailure called with key '123', got %s", checkoutGateway.markFailureKey)
	}
}

func TestCheckoutChargeWithTotalFee(t *testing.T) {
	stock := &mockStockGateway{reserveResult: &protocols.Reservation{Id: 1, TotalFee: 123.45}}
	payment := &mockPaymentGateway{}
	checkoutGateway := &mockCheckoutGateway{}
	sleeper := &MockSleeper{}
	uc := NewCheckout(stock, payment, checkoutGateway, sleeper)

	_ = uc.Checkout(Input{ItemId: 1, Quantity: 2, IdempotencyKey: "123"})
	if len(payment.charged) != 1 {
		t.Fatalf("expected Charge to be called once, got %d", len(payment.charged))
	}
	if payment.charged[0] != 123.45 {
		t.Fatalf("expected Charge amount 123.45, got %v", payment.charged[0])
	}
}

func TestCheckoutReleaseOnChargeFail(t *testing.T) {
	stock := &mockStockGateway{reserveResult: &protocols.Reservation{Id: 2, TotalFee: 50}}
	payment := &mockPaymentGateway{chargeErr: errors.New("charge error")}
	checkoutGateway := &mockCheckoutGateway{}
	sleeper := &MockSleeper{}
	uc := NewCheckout(stock, payment, checkoutGateway, sleeper)

	err := uc.Checkout(Input{ItemId: 1, Quantity: 2, IdempotencyKey: "123"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if len(stock.releasedIds) != 1 || stock.releasedIds[0] != 2 {
		t.Fatalf("expected Release called with res-2, got %v", stock.releasedIds)
	}
}

func TestCheckoutCompleteCalled(t *testing.T) {
	stock := &mockStockGateway{reserveResult: &protocols.Reservation{Id: 3, TotalFee: 10}}
	payment := &mockPaymentGateway{}
	checkoutGateway := &mockCheckoutGateway{}
	sleeper := &MockSleeper{}
	uc := NewCheckout(stock, payment, checkoutGateway, sleeper)

	_ = uc.Checkout(Input{ItemId: 1, Quantity: 1, IdempotencyKey: "123"})
	if len(stock.completedIds) != 1 || stock.completedIds[0] != 3 {
		t.Fatalf("expected Complete called with res-3, got %v", stock.completedIds)
	}
}

func TestCheckoutReleaseOnCompleteFail(t *testing.T) {
	stock := &mockStockGateway{reserveResult: &protocols.Reservation{Id: 4, TotalFee: 10}, completeErr: errors.New("complete error")}
	payment := &mockPaymentGateway{}
	checkoutGateway := &mockCheckoutGateway{}
	sleeper := &MockSleeper{}
	uc := NewCheckout(stock, payment, checkoutGateway, sleeper)

	err := uc.Checkout(Input{ItemId: 1, Quantity: 1, IdempotencyKey: "123"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if len(stock.releasedIds) != 1 || stock.releasedIds[0] != 4 {
		t.Fatalf("expected Release called with res-4, got %v", stock.releasedIds)
	}
}

func TestCheckoutSuccess(t *testing.T) {
	stock := &mockStockGateway{reserveResult: &protocols.Reservation{Id: 5, TotalFee: 20}}
	payment := &mockPaymentGateway{}
	checkoutGateway := &mockCheckoutGateway{}
	sleeper := &MockSleeper{}
	uc := NewCheckout(stock, payment, checkoutGateway, sleeper)

	err := uc.Checkout(Input{ItemId: 1, Quantity: 2, IdempotencyKey: "123"})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if len(payment.charged) != 1 || payment.charged[0] != 20 {
		t.Fatalf("expected Charge called with 20, got %v", payment.charged)
	}
	if len(stock.completedIds) != 1 || stock.completedIds[0] != 5 {
		t.Fatalf("expected Complete called with res-5, got %v", stock.completedIds)
	}
	if !checkoutGateway.markSuccessCalled {
		t.Fatalf("expected MarkSuccess to be called")
	}
	if checkoutGateway.markSuccessKey != "123" {
		t.Fatalf("expected MarkSuccess called with key '123', got %s", checkoutGateway.markSuccessKey)
	}
}

func TestCheckoutWithExistingIdempotencyKey(t *testing.T) {
	stock := &mockStockGateway{reserveResult: &protocols.Reservation{Id: 6, TotalFee: 20}}
	payment := &mockPaymentGateway{}
	checkoutGateway := &mockCheckoutGateway{
		reserveIdempotencyKeyErr: errors.New("idempotency key is already being processed"),
	}
	sleeper := &MockSleeper{}
	uc := NewCheckout(stock, payment, checkoutGateway, sleeper)

	err := uc.Checkout(Input{ItemId: 1, Quantity: 2, IdempotencyKey: "123"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if len(stock.reservedInputs) != 0 {
		t.Fatalf("expected Reserve not to be called when idempotency key error, got %d calls", len(stock.reservedInputs))
	}
}

func TestCheckoutWithSuccessfulIdempotencyKey(t *testing.T) {
	stock := &mockStockGateway{reserveResult: &protocols.Reservation{Id: 7, TotalFee: 30}}
	payment := &mockPaymentGateway{}
	checkoutGateway := &mockCheckoutGateway{
		reserveIdempotencyKeyResult: &protocols.CheckoutIdempotencyKeyResult{
			Success: true,
			Error:   nil,
		},
	}
	sleeper := &MockSleeper{}
	uc := NewCheckout(stock, payment, checkoutGateway, sleeper)

	err := uc.Checkout(Input{ItemId: 1, Quantity: 2, IdempotencyKey: "abc-123"})
	if err != nil {
		t.Fatalf("expected nil error when idempotency key already succeeded, got %v", err)
	}
	if len(stock.reservedInputs) != 0 {
		t.Fatalf("expected Reserve not to be called when idempotency key already succeeded, got %d calls", len(stock.reservedInputs))
	}
	if len(payment.charged) != 0 {
		t.Fatalf("expected Charge not to be called when idempotency key already succeeded, got %d calls", len(payment.charged))
	}
	if len(stock.completedIds) != 0 {
		t.Fatalf("expected Complete not to be called when idempotency key already succeeded, got %d calls", len(stock.completedIds))
	}
	if checkoutGateway.markSuccessCalled {
		t.Fatalf("expected MarkSuccess not to be called when idempotency key already succeeded")
	}
	if checkoutGateway.markFailureCalled {
		t.Fatalf("expected MarkFailure not to be called when idempotency key already succeeded")
	}
}

func TestCheckoutWithProcessingIdempotencyKey(t *testing.T) {
	stock := &mockStockGateway{reserveResult: &protocols.Reservation{Id: 8, TotalFee: 40}}
	payment := &mockPaymentGateway{}
	checkoutGateway := &mockCheckoutGateway{
		reserveIdempotencyKeyErr: errors.New("idempotency key is already being processed"),
	}
	sleeper := &MockSleeper{}
	uc := NewCheckout(stock, payment, checkoutGateway, sleeper)

	err := uc.Checkout(Input{ItemId: 1, Quantity: 2, IdempotencyKey: "xyz-456"})
	if err == nil {
		t.Fatalf("expected error when idempotency key is processing, got nil")
	}
	if err.Error() != "idempotency key is already being processed" {
		t.Fatalf("expected error message 'idempotency key is already being processed', got %v", err)
	}
	if len(stock.reservedInputs) != 0 {
		t.Fatalf("expected Reserve not to be called when idempotency key is processing, got %d calls", len(stock.reservedInputs))
	}
	if len(payment.charged) != 0 {
		t.Fatalf("expected Charge not to be called when idempotency key is processing, got %d calls", len(payment.charged))
	}
}

func TestCheckoutMarkFailureOnReserveError(t *testing.T) {
	stock := &mockStockGateway{reserveErr: errors.New("reserve error")}
	payment := &mockPaymentGateway{}
	checkoutGateway := &mockCheckoutGateway{}
	sleeper := &MockSleeper{}
	uc := NewCheckout(stock, payment, checkoutGateway, sleeper)

	err := uc.Checkout(Input{ItemId: 1, Quantity: 2, IdempotencyKey: "fail-1"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !checkoutGateway.markFailureCalled {
		t.Fatalf("expected MarkFailure to be called on reserve error")
	}
	if checkoutGateway.markFailureKey != "fail-1" {
		t.Fatalf("expected MarkFailure called with key 'fail-1', got %s", checkoutGateway.markFailureKey)
	}
}

func TestCheckoutMarkFailureOnChargeError(t *testing.T) {
	stock := &mockStockGateway{reserveResult: &protocols.Reservation{Id: 9, TotalFee: 50}}
	payment := &mockPaymentGateway{chargeErr: errors.New("charge error")}
	checkoutGateway := &mockCheckoutGateway{}
	sleeper := &MockSleeper{}
	uc := NewCheckout(stock, payment, checkoutGateway, sleeper)

	err := uc.Checkout(Input{ItemId: 1, Quantity: 2, IdempotencyKey: "fail-2"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !checkoutGateway.markFailureCalled {
		t.Fatalf("expected MarkFailure to be called on charge error")
	}
	if checkoutGateway.markFailureKey != "fail-2" {
		t.Fatalf("expected MarkFailure called with key 'fail-2', got %s", checkoutGateway.markFailureKey)
	}
}

func TestCheckoutMarkFailureOnCompleteError(t *testing.T) {
	stock := &mockStockGateway{
		reserveResult: &protocols.Reservation{Id: 10, TotalFee: 60},
		completeErr:   errors.New("complete error"),
	}
	payment := &mockPaymentGateway{}
	checkoutGateway := &mockCheckoutGateway{}
	sleeper := &MockSleeper{}
	uc := NewCheckout(stock, payment, checkoutGateway, sleeper)

	err := uc.Checkout(Input{ItemId: 1, Quantity: 2, IdempotencyKey: "fail-3"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !checkoutGateway.markFailureCalled {
		t.Fatalf("expected MarkFailure to be called on complete error")
	}
	if checkoutGateway.markFailureKey != "fail-3" {
		t.Fatalf("expected MarkFailure called with key 'fail-3', got %s", checkoutGateway.markFailureKey)
	}
}

func TestCheckoutMarkSuccessOnCompleteSuccess(t *testing.T) {
	stock := &mockStockGateway{reserveResult: &protocols.Reservation{Id: 11, TotalFee: 70}}
	payment := &mockPaymentGateway{}
	checkoutGateway := &mockCheckoutGateway{}
	sleeper := &MockSleeper{}
	uc := NewCheckout(stock, payment, checkoutGateway, sleeper)

	err := uc.Checkout(Input{ItemId: 1, Quantity: 2, IdempotencyKey: "success-1"})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !checkoutGateway.markSuccessCalled {
		t.Fatalf("expected MarkSuccess to be called on successful checkout")
	}
	if checkoutGateway.markSuccessKey != "success-1" {
		t.Fatalf("expected MarkSuccess called with key 'success-1', got %s", checkoutGateway.markSuccessKey)
	}
	if checkoutGateway.markFailureCalled {
		t.Fatalf("expected MarkFailure not to be called on successful checkout")
	}
}
