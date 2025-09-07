package protocols

type PaymentGateway interface {
	Charge(amount float64) error
}