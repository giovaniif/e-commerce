package reserve

import (
	"github.com/giovaniif/e-commerce/stock/domain/item"
)

type Reserve struct {
	itemRepository item.Repository
}

func NewReserve(itemRepository item.Repository) *Reserve {
	return &Reserve{
		itemRepository: itemRepository,
	}
}

func (r *Reserve) Reserve(itemId int32, quantity int32) (Output, error) {
	item, err := r.itemRepository.GetItem(itemId)
	if err != nil {
		return Output{}, err
	}

	reservation, err := r.itemRepository.Reserve(item, quantity)
	if err != nil {
		return Output{}, err
	}

	return Output{
		ReservationId: reservation.Id,
		TotalFee: reservation.TotalFee,
	}, nil
}

type Input struct {
	ItemId int32
	Quantity int32
}

type Output struct {
  ReservationId int32
	TotalFee float64
}
