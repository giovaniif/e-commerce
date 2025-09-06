package checkout

import (
	"testing"

	item "github.com/giovaniif/e-commerce/domain/item"
)

type ItemRepositoryMock struct {
	GetItemFunc func(itemId int32) item.Item
	SaveFunc func(item item.Item)
}

func (m *ItemRepositoryMock) GetItem(itemId int32) item.Item {
	return m.GetItemFunc(itemId)
}

func (m *ItemRepositoryMock) Save(item item.Item) {
	m.SaveFunc(item)
}

type PaymentGatewayMock struct {
	ChargeFunc func(amount float64) error
}

func (m *PaymentGatewayMock) Charge(amount float64) error {
	return m.ChargeFunc(amount)
}

func TestCheckoutNotEnoughStock(t *testing.T) {
	checkout := Checkout{
		itemRepository: &ItemRepositoryMock{
			GetItemFunc: func(itemId int32) item.Item {
				return item.Item{Stock: 0}
			},
			SaveFunc: func(item item.Item) {},
		},
		paymentGateway: &PaymentGatewayMock{
			ChargeFunc: func(amount float64) error {
				return nil
			},
		},
	}
	err := checkout.Checkout(Input{itemId: 1, quantity: 1})
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if err.Error() != "not enough stock" {
		t.Errorf("Expected error, got %v", err)
	}
}