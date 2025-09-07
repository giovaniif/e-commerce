package gateways

type ChargeGatewayMemory struct {
	charged []float64
}

func NewChargeGatewayMemory() *ChargeGatewayMemory {
	return &ChargeGatewayMemory{}
}

func (c *ChargeGatewayMemory) Charge(amount float64) error {
	c.charged = append(c.charged, amount)
	return nil
}
