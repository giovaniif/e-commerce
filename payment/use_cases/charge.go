package charge

import (
	protocols "github.com/giovaniif/e-commerce/payment/protocols"
)

func NewCharge(chargeGateway protocols.ChargeGateway) *Charge {
	return &Charge{
		chargeGateway: chargeGateway,
	}
}

func (c *Charge) Charge(amount float64) error {
	return c.chargeGateway.Charge(amount)
}

type Charge struct {
	chargeGateway protocols.ChargeGateway
}
