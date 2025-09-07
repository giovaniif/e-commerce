package protocols

type ChargeGateway interface {
	Charge(amount float64) error
}