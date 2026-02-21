package protocols

import "context"

type Reservation struct {
	Id       int32
	TotalFee float64
}

type StockGateway interface {
	Reserve(ctx context.Context, itemId int32, quantity int32) (*Reservation, error)
	Release(ctx context.Context, reservationId int32) error
	Complete(ctx context.Context, reservationId int32) error
}
