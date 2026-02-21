package api

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/giovaniif/e-commerce/payment/infra/gateways"
	"github.com/giovaniif/e-commerce/payment/infra/loki"
	"github.com/giovaniif/e-commerce/payment/infra/metrics"
	"github.com/giovaniif/e-commerce/payment/infra/requestid"
	"github.com/giovaniif/e-commerce/payment/infra/tracing"
	charge "github.com/giovaniif/e-commerce/payment/use_cases"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type ChargeRequest struct {
	Amount float64 `json:"amount"`
}

func StartServer() {
	logOut := io.Writer(os.Stdout)
	var lokiWriter *loki.Writer
	if lokiURL := os.Getenv("LOKI_URL"); lokiURL != "" {
		if lw := loki.NewWriter(lokiURL, "payment"); lw != nil {
			lokiWriter = lw
			logOut = io.MultiWriter(os.Stdout, lw)
		}
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(logOut, &slog.HandlerOptions{Level: slog.LevelInfo})))
	slog.Info("payment service started", "port", 3132)

	shutdownTracing := tracing.Init("payment")
	if shutdownTracing != nil {
		defer shutdownTracing()
	}

	r := gin.Default()
	chargeGateway := gateways.NewChargeGatewayMemory()
	idempotencyGateway := gateways.NewIdempotencyGatewayMemory()

	r.Use(func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = requestid.Generate()
		}
		c.Request = c.Request.WithContext(requestid.NewContext(c.Request.Context(), id))
		c.Header("X-Request-ID", id)
		c.Next()
		if c.Writer.Status() >= 400 {
			slog.Error("request", "request_id", id, "method", c.Request.Method, "path", c.Request.URL.Path, "status", c.Writer.Status())
		}
	})
	r.Use(tracing.Middleware("payment"))
	r.Use(metrics.Middleware)

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
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
	if shutdownTracing != nil {
		shutdownTracing()
	}
	if lokiWriter != nil {
		_ = lokiWriter.Close()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		fmt.Printf("Payment shutdown: %v\n", err)
	} else {
		fmt.Println("Payment stopped")
	}
}
