package gateways

type PaymentGatewayMemory struct {}

func NewPaymentGatewayMemory() *PaymentGatewayMemory {
	return &PaymentGatewayMemory{}
}

func (p *PaymentGatewayMemory) Charge(amount float64) error {
	return nil
}