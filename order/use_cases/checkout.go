package checkout

import (
	"fmt"
	"math"
	"time"

	protocols "github.com/giovaniif/e-commerce/order/protocols"
)

var (
	MAX_RETRIES = 5
	BASE_DELAY  = 1 * time.Second
)

func NewCheckout(stockGateway protocols.StockGateway, paymentGateway protocols.PaymentGateway, checkoutGateway protocols.CheckoutGateway, sleeper protocols.Sleeper) *Checkout {
	return &Checkout{
		stockGateway:    stockGateway,
		paymentGateway:  paymentGateway,
		checkoutGateway: checkoutGateway,
		sleeper: sleeper,
	}
}

func (c *Checkout) Checkout(input Input) error {
	result, err := c.checkoutGateway.ReserveIdempotencyKey(input.IdempotencyKey)
	if err != nil {
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

	reservationOperation := func() (*protocols.Reservation, error) {
		reservation, reservationError := c.stockGateway.Reserve(input.ItemId, input.Quantity)
		return reservation, reservationError
	}
	wrappedOperation := RetryWithBackoff(reservationOperation, c.sleeper)
	reservation, err := wrappedOperation()
	if err != nil {
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

type RetryFunc func() (*protocols.Reservation, error)

func RetryWithBackoff(operation RetryFunc, sleeper protocols.Sleeper) RetryFunc {
    return func() (*protocols.Reservation, error) {
		var lastError error

		for i := 0; i < MAX_RETRIES; i++ {
			val, err := operation()

			if err == nil {
				return val, err
			}

			secRetry := math.Pow(2, float64(i))
			fmt.Printf("Retrying operation in %f seconds\n", secRetry)
			delay := time.Duration(secRetry) * BASE_DELAY
			sleeper.Sleep(delay)
			lastError = err
		}

		return nil, lastError
	}
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
	sleeper         protocols.Sleeper
}
