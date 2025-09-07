package item

type Item struct {
	Id int32
	Price float64
  InitialStock int32
  Reservations []Reservation
}

func (i *Item) GetAvailableStock() int32 {
	availableStock := i.InitialStock
	for _, reservation := range i.Reservations {
		if reservation.Status != "canceled" {
			availableStock -= reservation.Quantity
		}
	}
	return availableStock
}

type Reservation struct {
	Id int32
	TotalFee float64
	Quantity int32
	ItemId int32
	Status string
}