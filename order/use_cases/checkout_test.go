package checkout

import (
	"errors"
	"testing"

	protocols "github.com/giovaniif/e-commerce/order/protocols"
)

type mockStockGateway struct {
	reservedInputs    []struct{ itemId, quantity int32 }
	reserveResult     *protocols.Reservation
	reserveErr        error
	releasedIds       []int32
	releaseErr        error
	completedIds      []int32
	completeErr       error
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
	charged []float64
	chargeErr error
}

func (m *mockPaymentGateway) Charge(amount float64) error {
	m.charged = append(m.charged, amount)
	return m.chargeErr
}

func TestCheckoutReserveError(t *testing.T) {
	stock := &mockStockGateway{reserveErr: errors.New("reserve error")}
	payment := &mockPaymentGateway{}
	uc := NewCheckout(stock, payment)

	err := uc.Checkout(Input{ItemId: 1, Quantity: 2})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if len(stock.reservedInputs) != 1 {
		t.Fatalf("expected Reserve to be called once, got %d", len(stock.reservedInputs))
	}
}

func TestCheckoutChargeWithTotalFee(t *testing.T) {
	stock := &mockStockGateway{reserveResult: &protocols.Reservation{Id: 1, TotalFee: 123.45}}
	payment := &mockPaymentGateway{}
	uc := NewCheckout(stock, payment)

	_ = uc.Checkout(Input{ItemId: 1, Quantity: 2})
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
	uc := NewCheckout(stock, payment)

	err := uc.Checkout(Input{ItemId: 1, Quantity: 2})
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
	uc := NewCheckout(stock, payment)

	_ = uc.Checkout(Input{ItemId: 1, Quantity: 1})
	if len(stock.completedIds) != 1 || stock.completedIds[0] != 3 {
		t.Fatalf("expected Complete called with res-3, got %v", stock.completedIds)
	}
}

func TestCheckoutReleaseOnCompleteFail(t *testing.T) {
	stock := &mockStockGateway{reserveResult: &protocols.Reservation{Id: 4, TotalFee: 10}, completeErr: errors.New("complete error")}
	payment := &mockPaymentGateway{}
	uc := NewCheckout(stock, payment)

	err := uc.Checkout(Input{ItemId: 1, Quantity: 1})
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
	uc := NewCheckout(stock, payment)

	err := uc.Checkout(Input{ItemId: 1, Quantity: 2})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if len(payment.charged) != 1 || payment.charged[0] != 20 {
		t.Fatalf("expected Charge called with 20, got %v", payment.charged)
	}
	if len(stock.completedIds) != 1 || stock.completedIds[0] != 5 {
		t.Fatalf("expected Complete called with res-5, got %v", stock.completedIds)
	}
}