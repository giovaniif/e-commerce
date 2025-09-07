package reserve

import (
	"errors"
	"testing"

	stockitem "github.com/giovaniif/e-commerce/stock/domain/item"
)

type mockRepository struct {
	getItemResult *stockitem.Item
	getItemErr    error
	reserveResult *stockitem.Reservation
	reserveErr    error
	releaseErr    error
	completeErr   error

	getItemCalledWithId        int32
	reserveCalledWithItemId    int32
	reserveCalledWithQuantity  int32
	releaseCalledWithId        int32
	completeCalledWithId       int32
}

func (m *mockRepository) GetItem(itemId int32) (*stockitem.Item, error) {
	m.getItemCalledWithId = itemId
	return m.getItemResult, m.getItemErr
}

func (m *mockRepository) Reserve(reservationItem *stockitem.Item, quantity int32) (*stockitem.Reservation, error) {
	if reservationItem != nil {
		m.reserveCalledWithItemId = reservationItem.Id
	}
	m.reserveCalledWithQuantity = quantity
	return m.reserveResult, m.reserveErr
}

func (m *mockRepository) ReleaseReservation(reservationId int32) error {
	m.releaseCalledWithId = reservationId
	return m.releaseErr
}

func (m *mockRepository) CompleteReservation(reservationId int32) error {
	m.completeCalledWithId = reservationId
	return m.completeErr
}

func TestReserve_Success(t *testing.T) {
	repo := &mockRepository{
		getItemResult: &stockitem.Item{Id: 1, Price: 10, InitialStock: 5},
		reserveResult: &stockitem.Reservation{Id: 2, TotalFee: 30, Quantity: 3, ItemId: 1},
	}
	uc := NewReserve(repo)

	out, err := uc.Reserve(1, 3)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if out.ReservationId != 2 || out.TotalFee != 30 {
		t.Fatalf("unexpected output: %+v", out)
	}
	if repo.getItemCalledWithId != 1 {
		t.Fatalf("expected GetItem called with 1, got %d", repo.getItemCalledWithId)
	}
	if repo.reserveCalledWithItemId != 1 || repo.reserveCalledWithQuantity != 3 {
		t.Fatalf("expected Reserve called with (itemId=1, qty=3), got (itemId=%d, qty=%d)", repo.reserveCalledWithItemId, repo.reserveCalledWithQuantity)
	}
}

func TestReserve_GetItemError(t *testing.T) {
	repo := &mockRepository{
		getItemErr: errors.New("not found"),
	}
	uc := NewReserve(repo)

	_, err := uc.Reserve(1, 3)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestReserve_ReserveError(t *testing.T) {
	repo := &mockRepository{
		getItemResult: &stockitem.Item{Id: 1, Price: 10, InitialStock: 5},
		reserveErr: errors.New("cannot reserve"),
	}
	uc := NewReserve(repo)

	_, err := uc.Reserve(1, 3)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}


