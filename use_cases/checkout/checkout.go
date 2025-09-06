package checkout

import (
	"errors"

	item "github.com/giovaniif/e-commerce/domain/item"
	protocols "github.com/giovaniif/e-commerce/protocols"
)

func (c *Checkout) Checkout(input Input) (error) {
	item := c.itemRepository.GetItem(input.itemId)
	if item.Stock < input.quantity {
		return errors.New("not enough stock")
	}
  item.RemoveStock(input.quantity)  
	c.itemRepository.Save(item)
	c.paymentGateway.Charge(item.Price * float64(input.quantity))
	return nil
}

type Input struct{
	itemId int32
	quantity int32
}

type Checkout struct {
  itemRepository item.ItemRepository
	paymentGateway protocols.PaymentGateway
}