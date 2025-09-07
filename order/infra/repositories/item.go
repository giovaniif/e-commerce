package repositories

import (
	item "github.com/giovaniif/e-commerce/order/domain/item"
)

type ItemRepositoryMemory struct {
  items map[int32]item.Item
}

func NewItemRepositoryMemory() *ItemRepositoryMemory {
	return &ItemRepositoryMemory{
		items: make(map[int32]item.Item),
	}
}

func (r *ItemRepositoryMemory) GetItem(itemId int32) item.Item {
	return r.items[itemId]
}

func (r *ItemRepositoryMemory) Save(item item.Item) {
	r.items[item.Id] = item
}

func (r *ItemRepositoryMemory) Create(item item.Item) {
	r.items[item.Id] = item
}