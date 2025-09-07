package release

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

	releaseCalledWithId int32
}

func (m *mockRepository) GetItem(itemId int32) (*stockitem.Item, error) { return m.getItemResult, m.getItemErr }
func (m *mockRepository) Reserve(reservationItem *stockitem.Item, quantity int32) (*stockitem.Reservation, error) {
	return m.reserveResult, m.reserveErr
}
func (m *mockRepository) ReleaseReservation(reservationId int32) error {
	m.releaseCalledWithId = reservationId
	return m.releaseErr
}
func (m *mockRepository) CompleteReservation(reservationId int32) error { return m.completeErr }

func TestRelease_Success(t *testing.T) {
	repo := &mockRepository{}
	uc := NewRelease(repo)

	err := uc.Release(Input{ReservationId: 10})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if repo.releaseCalledWithId != 10 {
		t.Fatalf("expected release called with 10, got %d", repo.releaseCalledWithId)
	}
}

func TestRelease_Error(t *testing.T) {
	repo := &mockRepository{releaseErr: errors.New("not found")}
	uc := NewRelease(repo)

	err := uc.Release(Input{ReservationId: 10})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}


