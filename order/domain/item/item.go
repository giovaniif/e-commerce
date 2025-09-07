package item

type Item struct {
	Id int32
	Price float64
	Stock int32
}

func (i *Item) RemoveStock(quantity int32) {
	i.Stock -= quantity
}