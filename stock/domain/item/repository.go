package item

type Repository interface {
	GetItem(itemId int32) (*Item, error)
  Reserve(reservationItem *Item, quantity int32) (*Reservation, error)
  ReleaseReservation(reservationId int32) error
	CompleteReservation(reservationId int32) error
}