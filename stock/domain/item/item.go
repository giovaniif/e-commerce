package item

type Item struct {
	Id int32
	Price float64
	Stock int32
}

type Reservation struct {
	Id int32
	TotalFee float64
	Quantity int32
	ItemId int32
}