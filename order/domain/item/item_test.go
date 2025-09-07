package item

import (
	"testing"
)

func TestItemRemoveStock(t *testing.T) {
	item := Item{Stock: 10}
	item.RemoveStock(5)
	if item.Stock != 5 {
		t.Errorf("Expected stock to be 5, got %d", item.Stock)
	}
}