package charge

import (
	"fmt"

	protocols "github.com/giovaniif/e-commerce/payment/protocols"
)

func NewCharge(chargeGateway protocols.ChargeGateway, idempotencyGateway protocols.IdempotencyGateway) *Charge {
	return &Charge{
		chargeGateway:      chargeGateway,
		idempotencyGateway: idempotencyGateway,
	}
}

func (c *Charge) Charge(input ChargeInput) error {
	result, err := c.idempotencyGateway.ReserveIdempotencyKey(input.IdempotencyKey)
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
			c.idempotencyGateway.MarkSuccess(input.IdempotencyKey)
		} else {
			c.idempotencyGateway.MarkFailure(input.IdempotencyKey)
		}
	}()

	err = c.chargeGateway.Charge(input.Amount)
	if err != nil {
		return err
	}

	success = true
	return nil
}

type Charge struct {
	chargeGateway      protocols.ChargeGateway
	idempotencyGateway protocols.IdempotencyGateway
}

type ChargeInput struct {
	Amount         float64
	IdempotencyKey string
}
