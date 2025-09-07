package repositories

import (
	"errors"

	"github.com/giovaniif/e-commerce/stock/domain/item"
)

type ItemRepository struct {
	items map[int32]*item.Item
	reservations map[int32]*item.Reservation
}

func NewItemRepository() *ItemRepository {
	return &ItemRepository{items: make(map[int32]*item.Item), reservations: make(map[int32]*item.Reservation)}
}

func (r *ItemRepository) GetItem(itemId int32) (*item.Item, error) {
	item, ok := r.items[itemId]
	if !ok {
		return nil, errors.New("item not found")
	}
	return item, nil
}

func (r *ItemRepository) Reserve(reservationItem *item.Item, quantity int32) (*item.Reservation, error) {
	if reservationItem.Stock < quantity {
		return nil, errors.New("insufficient stock")
	}

	reservationItem.Stock -= quantity
	r.Save(reservationItem)

	newId := int32(len(r.reservations) + 1)
	reservation := &item.Reservation{
		Id: newId,
		TotalFee: float64(quantity) * reservationItem.Price,
		Quantity: quantity,
		ItemId: reservationItem.Id,
	}
	r.reservations[newId] = reservation
	return reservation, nil
}

func (r *ItemRepository) ReleaseReservation(reservationId int32) (error) {
	reservation, ok := r.reservations[reservationId]
	if !ok {
		return errors.New("reservation not found")
	}
  item, err := r.GetItem(reservation.ItemId)
  if err != nil {
    return err
  }
  item.Stock += reservation.Quantity
  r.Save(item)
	r.reservations[reservationId] = nil
  return nil
}

func (r *ItemRepository) CompleteReservation(reservationId int32) error {
	_, ok := r.reservations[reservationId]
	if !ok {
		return errors.New("reservation not found")
	}
	r.reservations[reservationId] = nil
	return nil
}

func (r *ItemRepository) Save(it *item.Item) {
	r.items[it.Id] = it
}