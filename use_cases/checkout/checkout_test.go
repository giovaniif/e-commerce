package checkout

import (
	"testing"

	item "github.com/giovaniif/e-commerce/domain/item"
)

type mockItemRepository struct {
	items map[int32]item.Item
	saved []item.Item
}

func (m *mockItemRepository) GetItem(itemId int32) item.Item {
	return m.items[itemId]
}

func (m *mockItemRepository) Save(it item.Item) {
	m.saved = append(m.saved, it)
}

func (m *mockItemRepository) Create(it item.Item) {
	if m.items == nil {
		m.items = make(map[int32]item.Item)
	}
	m.items[it.Id] = it
}

type mockPaymentGateway struct {
	charged []float64
}

func (m *mockPaymentGateway) Charge(amount float64) error {
	m.charged = append(m.charged, amount)
	return nil
}


func TestCheckoutNotEnoughStock(t *testing.T) {
	repo := &mockItemRepository{items: map[int32]item.Item{1: {Id: 1, Price: 10.0, Stock: 0}}}
	payment := &mockPaymentGateway{}
	checkoutUseCase := NewCheckout(repo, payment)

	err := checkoutUseCase.Checkout(Input{ItemId: 1, Quantity: 1})
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if err.Error() != "not enough stock" {
		t.Errorf("Expected error, got %v", err)
	}
}