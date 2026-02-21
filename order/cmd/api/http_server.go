package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/giovaniif/e-commerce/order/infra/gateways"
	"github.com/giovaniif/e-commerce/order/protocols"
	checkout "github.com/giovaniif/e-commerce/order/use_cases"
)

const defaultCheckoutTimeoutSec = 30

type CheckoutRequest struct {
	ItemId   int32 `json:"itemId"`
	Quantity int32 `json:"quantity"`
}

func StartServer() {
	httpClient := &http.Client{}
	stockGateway := gateways.NewStockGatewayHttp(httpClient)
	paymentGateway := gateways.NewPaymentGatewayHttp(httpClient)

	var checkoutGateway protocols.CheckoutGateway
	if redisAddr := os.Getenv("REDIS_ADDR"); redisAddr != "" {
		rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
		if err := rdb.Ping(context.Background()).Err(); err != nil {
			fmt.Printf("Redis ping failed (%s), using in-memory idempotency: %v\n", redisAddr, err)
			checkoutGateway = gateways.NewCheckoutGatewayMemory()
		} else {
			checkoutGateway = gateways.NewCheckoutGatewayRedis(rdb)
			fmt.Println("Checkout idempotency: Redis (TTL 24h)")
		}
	} else {
		checkoutGateway = gateways.NewCheckoutGatewayMemory()
		fmt.Println("Checkout idempotency: in-memory (set REDIS_ADDR for Redis)")
	}

	sleeperGateway := gateways.NewSleeper()
	checkoutUseCase := checkout.NewCheckout(stockGateway, paymentGateway, checkoutGateway, sleeperGateway)

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		status := "healthy"
		redisCheck := "n/a"
		if redisAddr := os.Getenv("REDIS_ADDR"); redisAddr != "" {
			rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
			if err := rdb.Ping(c.Request.Context()).Err(); err != nil {
				status = "degraded"
				redisCheck = "down"
			} else {
				redisCheck = "up"
			}
			_ = rdb.Close()
		}
		c.JSON(http.StatusOK, gin.H{"status": status, "checks": gin.H{"redis": redisCheck}})
	})

	checkoutTimeoutSec := defaultCheckoutTimeoutSec
	if s := os.Getenv("CHECKOUT_TIMEOUT_SECONDS"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			checkoutTimeoutSec = n
		}
	}

	r.POST("/checkout", func(c *gin.Context) {
		contextWithTimeout, cancel := context.WithTimeout(c.Request.Context(), time.Duration(checkoutTimeoutSec)*time.Second)
		defer cancel()

		var checkoutRequest CheckoutRequest
		if err := c.ShouldBindJSON(&checkoutRequest); err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		idempotencyKey := c.GetHeader("Idempotency-Key")
		if idempotencyKey == "" {
			c.String(http.StatusBadRequest, "Idempotency-Key header is required")
			return
		}

		err := checkoutUseCase.Checkout(contextWithTimeout, checkout.Input{
			ItemId:         checkoutRequest.ItemId,
			Quantity:       checkoutRequest.Quantity,
			IdempotencyKey: idempotencyKey,
		})
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				c.String(http.StatusGatewayTimeout, err.Error())
			} else {
				c.String(http.StatusInternalServerError, err.Error())
			}
		} else {
			c.String(http.StatusOK, "Checkout successful")
		}
	})

	srv := &http.Server{Addr: ":3131", Handler: r}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Order server: %v\n", err)
		}
	}()
	fmt.Println("Order is running on port 3131")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	fmt.Println("Order shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		fmt.Printf("Order shutdown: %v\n", err)
	} else {
		fmt.Println("Order stopped")
	}
}
