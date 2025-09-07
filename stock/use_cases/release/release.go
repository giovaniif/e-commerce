package release

import (
	"github.com/giovaniif/e-commerce/stock/domain/item"
)

type Release struct {
	itemRepository item.Repository
}

func NewRelease(itemRepository item.Repository) *Release {
	return &Release{
		itemRepository: itemRepository,
	}
}

func (r *Release) Release(input Input) (error) {
	err := r.itemRepository.ReleaseReservation(input.ReservationId)
	if err != nil {
		return err
	}

	return nil
}

type Input struct {
	ReservationId int32
}
