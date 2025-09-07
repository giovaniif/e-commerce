package repositories

import (
	"errors"
	"fmt"

	"github.com/giovaniif/e-commerce/stock/domain/item"
)

type ItemRepository struct {
	items map[int32]*item.Item
	reservations map[int32]*item.Reservation
}

func NewItemRepository(items map[int32]*item.Item, reservations map[int32]*item.Reservation) *ItemRepository {
	return &ItemRepository{items: items, reservations: reservations}
}

func (r *ItemRepository) GetItem(itemId int32) (*item.Item, error) {
	repositoryItem, ok := r.items[itemId]
	if !ok {
		return nil, errors.New("item not found")
	}
	var itemReservations []item.Reservation
	for _, reservation := range r.reservations {
		if reservation.ItemId == itemId {
			itemReservations = append(itemReservations, *reservation)
		}
	}
	repositoryItem.Reservations = itemReservations
	return repositoryItem, nil
}

func (r *ItemRepository) Reserve(reservationItem *item.Item, quantity int32) (*item.Reservation, error) {
	if reservationItem.GetAvailableStock() < quantity {
		return nil, errors.New("insufficient stock")
	}

	newId := int32(len(r.reservations) + 1)
	reservation := &item.Reservation{
		Id: newId,
		TotalFee: float64(quantity) * reservationItem.Price,
		Quantity: quantity,
		ItemId: reservationItem.Id,
		Status: "reserved",
	}
	r.reservations[newId] = reservation
	return reservation, nil
}

func (r *ItemRepository) ReleaseReservation(reservationId int32) (error) {
	reservation, ok := r.reservations[reservationId]
	if !ok {
		return errors.New("reservation not found")
	}
	r.reservations[reservationId] = &item.Reservation{
		Id: reservationId,
		TotalFee: reservation.TotalFee,
		Quantity: reservation.Quantity,
		ItemId: reservation.ItemId,
		Status: "canceled",
	}
  return nil
}

func (r *ItemRepository) CompleteReservation(reservationId int32) error {
	reservation, ok := r.reservations[reservationId]
	if !ok {
		fmt.Printf("reservation %d not found", reservationId)
		return errors.New("reservation not found")
	}
	r.reservations[reservationId] = &item.Reservation{
		Id: reservationId,
		TotalFee: reservation.TotalFee,
		Quantity: reservation.Quantity,
		ItemId: reservation.ItemId,
		Status: "completed",
	}
	return nil
}

func (r *ItemRepository) Save(it *item.Item) {
	r.items[it.Id] = it
}