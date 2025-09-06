package item

type Item struct {
	Price float64
	Stock int32
}

func (i *Item) RemoveStock(quantity int32) {
	i.Stock -= quantity
}