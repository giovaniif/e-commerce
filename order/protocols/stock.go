package protocols

type Reservation struct {
	Id int32
  TotalFee float64
}

type StockGateway interface {
	Reserve(itemId int32, quantity int32) (*Reservation, error)
  Release(reservationId int32) error
	Complete(reservationId int32) error
}
