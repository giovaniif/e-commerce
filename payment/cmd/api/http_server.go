package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/giovaniif/e-commerce/payment/infra/gateways"
	charge "github.com/giovaniif/e-commerce/payment/use_cases"
)

type ChargeRequest struct {
	Amount float64 `json:"amount"`
}

func StartServer() {
	r := gin.Default()
	chargeGateway := gateways.NewChargeGatewayMemory()

	r.POST("/charge", func(c *gin.Context) {
		chargeUseCase := charge.NewCharge(chargeGateway)
		var chargeRequest ChargeRequest
		if err := c.ShouldBindJSON(&chargeRequest); err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}
		err := chargeUseCase.Charge(chargeRequest.Amount)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
		} else {
			c.String(http.StatusOK, "Charge successful")
		}
	})

	r.Run(":3132")
	fmt.Println("Payment is running on port 3132")
}