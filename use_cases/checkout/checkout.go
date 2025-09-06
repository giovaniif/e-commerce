package checkout

import (
	"errors"

	item "github.com/giovaniif/e-commerce/domain/item"
	protocols "github.com/giovaniif/e-commerce/protocols"
)

func NewCheckout(itemRepository item.ItemRepository, paymentGateway protocols.PaymentGateway) *Checkout {
	return &Checkout{
		itemRepository: itemRepository,
		paymentGateway: paymentGateway,
	}
}

func (c *Checkout) Checkout(input Input) (error) {
	item := c.itemRepository.GetItem(input.ItemId)
	if item.Stock < input.Quantity {
		return errors.New("not enough stock")
	}
  item.RemoveStock(input.Quantity)  
	c.itemRepository.Save(item)
	c.paymentGateway.Charge(item.Price * float64(input.Quantity))
	return nil
}

type Input struct {
	ItemId int32
	Quantity int32
}

type Checkout struct {
  itemRepository item.ItemRepository
	paymentGateway protocols.PaymentGateway
}