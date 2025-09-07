package complete

import (
	"github.com/giovaniif/e-commerce/stock/domain/item"
)

type Complete struct {
	itemRepository item.Repository
}

func NewComplete(itemRepository item.Repository) *Complete {
	return &Complete{
		itemRepository: itemRepository,
	}
}

func (c *Complete) Complete(input Input) (error) {
	err := c.itemRepository.CompleteReservation(input.ReservationId)
	if err != nil {
		return err
	}

	return nil
}

type Input struct {
	ReservationId int32
}