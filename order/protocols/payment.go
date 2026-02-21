package protocols

import "context"

type PaymentGateway interface {
	Charge(context context.Context, amount float64) error
}
