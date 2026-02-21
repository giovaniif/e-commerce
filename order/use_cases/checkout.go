package checkout

import (
	"context"
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
		sleeper:         sleeper,
	}
}

func (c *Checkout) Checkout(ctx context.Context, input Input) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	result, err := c.checkoutGateway.ReserveIdempotencyKey(ctx, input.IdempotencyKey)
	if err != nil {
		return err
	}
	keyBeingProcessed := result != nil
	if keyBeingProcessed {
		return nil
	}

	success := false
	defer func() {
		if success {
			c.checkoutGateway.MarkSuccess(ctx, input.IdempotencyKey)
		} else {
			c.checkoutGateway.MarkFailure(ctx, input.IdempotencyKey)
		}
	}()

	reservationOperation := func() (*protocols.Reservation, error) {
		reservation, reservationError := c.stockGateway.Reserve(ctx, input.ItemId, input.Quantity)
		return reservation, reservationError
	}
	wrappedOperation := RetryWithBackoff(ctx, reservationOperation, c.sleeper)
	reservation, err := wrappedOperation()
	if err != nil {
		return err
	}

	err = c.paymentGateway.Charge(ctx, reservation.TotalFee)
	if err != nil {
		c.stockGateway.Release(ctx, reservation.Id)
		return err
	}

	completeStockOperation := RetryWithBackoff(ctx, func() (*protocols.Reservation, error) {
		completeStockError := c.stockGateway.Complete(ctx, reservation.Id)

		return nil, completeStockError
	}, c.sleeper)
	_, err = completeStockOperation()
	if err != nil {
		releaseStockOperation := RetryWithBackoff(ctx, func() (*protocols.Reservation, error) {
			releaseStockError := c.stockGateway.Release(ctx, reservation.Id)
			return nil, releaseStockError
		}, c.sleeper)
		_, releaseStockError := releaseStockOperation()
		if releaseStockError != nil {
			fmt.Printf("Failed to release stock for reservation after complete error %d: %v\n", reservation.Id, releaseStockError)
			return releaseStockError
		}
		return err
	}

	success = true
	return nil
}

type RetryFunc func() (*protocols.Reservation, error)

func RetryWithBackoff(ctx context.Context, operation RetryFunc, sleeper protocols.Sleeper) RetryFunc {
	return func() (*protocols.Reservation, error) {
		var lastError error

		for i := 0; i < MAX_RETRIES; i++ {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			val, err := operation()

			if err == nil {
				return val, err
			}
			lastError = err

			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			secRetry := math.Pow(2, float64(i))
			fmt.Printf("Retrying operation in %f seconds\n", secRetry)
			delay := time.Duration(secRetry) * BASE_DELAY
			sleeper.Sleep(delay)
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
