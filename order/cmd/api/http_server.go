package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/giovaniif/e-commerce/order/infra/gateways"
	"github.com/giovaniif/e-commerce/order/use_cases/checkout"
)

type CheckoutRequest struct {
	ItemId int32 `json:"itemId"`
	Quantity int32 `json:"quantity"`
}

func StartServer() {
	httpClient := &http.Client{}
	stockGateway := gateways.NewStockGatewayHttp(httpClient)
	paymentGateway := gateways.NewPaymentGatewayHttp(httpClient)
	checkoutUseCase := checkout.NewCheckout(stockGateway, paymentGateway)

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