package protocols

type Reservation struct {
	Id string
  TotalFee float64
}

type StockGateway interface {
	Reserve(itemId int32, quantity int32) (*Reservation, error)
  Release(reservationId string) error
	Complete(reservationId string) error
}
