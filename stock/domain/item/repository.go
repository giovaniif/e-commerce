package item

type Repository interface {
	GetItem(itemId int32) (*Item, error)
	Reserve(reservationItem *Item, quantity int32, orderId string, idempotencyKey string, traceparent string) (*Reservation, error)
	ReleaseReservation(reservationId int32, traceparent string) error
	CompleteReservation(reservationId int32) error
}
