package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/giovaniif/e-commerce/order/domain/item"
	"github.com/giovaniif/e-commerce/order/infra/gateways"
	"github.com/giovaniif/e-commerce/order/infra/repositories"
	"github.com/giovaniif/e-commerce/order/use_cases/checkout"
)

type CheckoutRequest struct {
	ItemId int32 `json:"itemId"`
	Quantity int32 `json:"quantity"`
}

func StartServer() {
	itemRepository := repositories.NewItemRepositoryMemory()
	paymentGateway := gateways.NewPaymentGatewayHttp()
	checkoutUseCase := checkout.NewCheckout(itemRepository, paymentGateway)
  itemRepository.Create(item.Item{
    Id: 1,
    Price: 10,
    Stock: 10,
  })

	r := gin.Default()

	r.POST("/checkout", func(c *gin.Context) {
		var checkoutRequest CheckoutRequest
		if err := c.ShouldBindJSON(&checkoutRequest); err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}
		err := checkoutUseCase.Checkout(checkout.Input{
			ItemId: checkoutRequest.ItemId,
			Quantity: checkoutRequest.Quantity,
		})
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
		} else {
			c.String(http.StatusOK, "Checkout successful")
		}
	})

	r.Run(":3131")
	fmt.Println("Order is running on port 3131")
}