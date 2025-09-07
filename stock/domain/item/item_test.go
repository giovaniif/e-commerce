package item

import "testing"

func TestGetAvailableStock(t *testing.T) {
	item := Item{
		InitialStock: 10,
		Reservations: []Reservation{{Quantity: 5, Status: "reserved"}},
	}
	if item.GetAvailableStock() != 5 {
		t.Errorf("Expected available stock to be 5, got %d", item.GetAvailableStock())
	}
	item.Reservations = []Reservation{{Quantity: 5, Status: "canceled"}}
	if item.GetAvailableStock() != 10 {
		t.Errorf("Expected available stock to be 10, got %d", item.GetAvailableStock())
	}
	item.Reservations = []Reservation{{Quantity: 5, Status: "completed"}}
	if item.GetAvailableStock() != 5 {
		t.Errorf("Expected available stock to be 5, got %d", item.GetAvailableStock())
	}
}