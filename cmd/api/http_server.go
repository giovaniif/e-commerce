package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/giovaniif/e-commerce/domain/item"
	"github.com/giovaniif/e-commerce/infra/gateways"
	"github.com/giovaniif/e-commerce/infra/repositories"
	"github.com/giovaniif/e-commerce/use_cases/checkout"
)

func StartServer() {
	itemRepository := repositories.NewItemRepositoryMemory()
	paymentGateway := gateways.NewPaymentGatewayMemory()
  itemRepository.Create(item.Item{
    Id: 1,
    Price: 10,
    Stock: 10,
  })

	r := gin.Default()

	r.POST("/checkout", func(c *gin.Context) {
		checkoutUseCase := checkout.NewCheckout(itemRepository, paymentGateway)
		err := checkoutUseCase.Checkout(checkout.Input{
			ItemId: 1,
			Quantity: 1,
		})
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
		} else {
			c.String(http.StatusOK, "Checkout successful")
		}
	})

	r.GET("/items", func(c *gin.Context) {
		it := itemRepository.GetItem(1)
		c.JSON(http.StatusOK, it)
	})

	r.Run(":3131")
}