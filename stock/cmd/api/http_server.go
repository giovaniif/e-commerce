package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/giovaniif/e-commerce/stock/domain/item"
	"github.com/giovaniif/e-commerce/stock/infra/repositories"
	"github.com/giovaniif/e-commerce/stock/use_cases/complete"
	"github.com/giovaniif/e-commerce/stock/use_cases/release"
	"github.com/giovaniif/e-commerce/stock/use_cases/reserve"
)


type ReserveRequest struct {
	ItemId int32 `json:"itemId"`
	Quantity int32 `json:"quantity"`
}

type ReleaseRequest struct {
	ReservationId int32 `json:"reservationId"`
}

type CompleteRequest struct {
	ReservationId int32 `json:"reservationId"`
}

func StartServer() {
	items := make(map[int32]*item.Item)
	items[1] = &item.Item{Id: 1, Price: 10, InitialStock: 10}
	reservations := make(map[int32]*item.Reservation)
	itemRepository := repositories.NewItemRepository(items, reservations)
	reserveUseCase := reserve.NewReserve(itemRepository)
	releaseUseCase := release.NewRelease(itemRepository)
	completeUseCase := complete.NewComplete(itemRepository)
	r := gin.Default()

	r.POST("/reserve", func(c *gin.Context) {
		var reserveRequest ReserveRequest
		if err := c.ShouldBindJSON(&reserveRequest); err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}
		_, err := reserveUseCase.Reserve(reserveRequest.ItemId, reserveRequest.Quantity)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
		} else {
			c.String(http.StatusOK, "Reserve successful")
		}
	})

	r.POST("/release", func(c *gin.Context) {
		var releaseRequest ReleaseRequest
		if err := c.ShouldBindJSON(&releaseRequest); err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}
		err := releaseUseCase.Release(release.Input{ReservationId: releaseRequest.ReservationId})
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
		} else {
			c.String(http.StatusOK, "Release successful")
		}
	})

	r.POST("/complete", func(c *gin.Context) {
		var completeRequest CompleteRequest
		if err := c.ShouldBindJSON(&completeRequest); err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}
		err := completeUseCase.Complete(complete.Input{ReservationId: completeRequest.ReservationId})
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
		} else {
			c.String(http.StatusOK, "Complete successful")
		}
	})

	r.Run(":3133")
	fmt.Println("Stock is running on port 3133")
}