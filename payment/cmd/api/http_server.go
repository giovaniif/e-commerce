package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	idempotencyGateway := gateways.NewIdempotencyGatewayMemory()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	r.POST("/charge", func(c *gin.Context) {
		chargeUseCase := charge.NewCharge(chargeGateway, idempotencyGateway)
		idempotencyKey := c.GetHeader("Idempotency-Key")
		if idempotencyKey == "" {
			c.String(http.StatusBadRequest, "Idempotency-Key header is required")
			return
		}
		var chargeRequest ChargeRequest
		if err := c.ShouldBindJSON(&chargeRequest); err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}
		err := chargeUseCase.Charge(charge.ChargeInput{
			IdempotencyKey: idempotencyKey,
			Amount:         chargeRequest.Amount,
		})
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
		} else {
			c.String(http.StatusOK, "Charge successful")
		}
	})

	srv := &http.Server{Addr: ":3132", Handler: r}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Payment server: %v\n", err)
		}
	}()
	fmt.Println("Payment is running on port 3132")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	fmt.Println("Payment shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		fmt.Printf("Payment shutdown: %v\n", err)
	} else {
		fmt.Println("Payment stopped")
	}
}
