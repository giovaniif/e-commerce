package checkout

import (
	"fmt"

	protocols "github.com/giovaniif/e-commerce/order/protocols"
)

func NewCheckout(stockGateway protocols.StockGateway, paymentGateway protocols.PaymentGateway, checkoutGateway protocols.CheckoutGateway) *Checkout {
	return &Checkout{
		stockGateway:    stockGateway,
		paymentGateway:  paymentGateway,
		checkoutGateway: checkoutGateway,
	}
}

func (c *Checkout) Checkout(input Input) error {
	result, err := c.checkoutGateway.ReserveIdempotencyKey(input.IdempotencyKey)
	if err != nil {
		fmt.Println("failed to check idempotency key")
		return err
	}
	if result != nil {
		return nil
	}

	success := false
	defer func() {
		if success {
			c.checkoutGateway.MarkSuccess(input.IdempotencyKey)
		} else {
			c.checkoutGateway.MarkFailure(input.IdempotencyKey)
		}
	}()

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

	success = true
	return nil
}

type Input struct {
	ItemId         int32
	Quantity       int32
	IdempotencyKey string
}

type Checkout struct {
	stockGateway    protocols.StockGateway
	paymentGateway  protocols.PaymentGateway
	checkoutGateway protocols.CheckoutGateway
}
