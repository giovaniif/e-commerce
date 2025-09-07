package checkout

import (
	"fmt"

	protocols "github.com/giovaniif/e-commerce/order/protocols"
)

func NewCheckout(stockGateway protocols.StockGateway, paymentGateway protocols.PaymentGateway) *Checkout {
	return &Checkout{
		stockGateway: stockGateway,
		paymentGateway: paymentGateway,
	}
}

func (c *Checkout) Checkout(input Input) (error) {
  reservation, err := c.stockGateway.Reserve(input.ItemId, input.Quantity)
	if err != nil {
		fmt.Println("failed to reserve stock")
		return err
	}

	err = c.paymentGateway.Charge(reservation.TotalFee)
	if err != nil {
		fmt.Println("failed to charge")
		c.stockGateway.Release(reservation.Id)
		return err
	}

	fmt.Printf("completing reservation %d", reservation.Id)
	err = c.stockGateway.Complete(reservation.Id)
	if err != nil {
		fmt.Println("failed to complete reservation")
		c.stockGateway.Release(reservation.Id)
		return err
	}

	return nil
}

type Input struct {
	ItemId int32
	Quantity int32
}

type Checkout struct {
  stockGateway protocols.StockGateway
	paymentGateway protocols.PaymentGateway
}