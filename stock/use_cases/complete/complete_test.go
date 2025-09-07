package complete

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

	completeCalledWithId int32
}

func (m *mockRepository) GetItem(itemId int32) (*stockitem.Item, error) { return m.getItemResult, m.getItemErr }
func (m *mockRepository) Reserve(reservationItem *stockitem.Item, quantity int32) (*stockitem.Reservation, error) {
	return m.reserveResult, m.reserveErr
}
func (m *mockRepository) ReleaseReservation(reservationId int32) error { return m.releaseErr }
func (m *mockRepository) CompleteReservation(reservationId int32) error {
	m.completeCalledWithId = reservationId
	return m.completeErr
}

func TestComplete_Success(t *testing.T) {
	repo := &mockRepository{}
	uc := NewComplete(repo)

	err := uc.Complete(Input{ReservationId: 22})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if repo.completeCalledWithId != 22 {
		t.Fatalf("expected complete called with 22, got %d", repo.completeCalledWithId)
	}
}

func TestComplete_Error(t *testing.T) {
	repo := &mockRepository{completeErr: errors.New("cannot complete")}
	uc := NewComplete(repo)

	err := uc.Complete(Input{ReservationId: 22})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}


