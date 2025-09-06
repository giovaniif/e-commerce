package item

type ItemRepository interface {
	GetItem(itemId int32) Item
	Save(item Item)
}