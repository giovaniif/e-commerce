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
	"github.com/redis/go-redis/v9"
	"github.com/giovaniif/e-commerce/payment/infra/gateways"
	"github.com/giovaniif/e-commerce/payment/infra/loki"
	"github.com/giovaniif/e-commerce/payment/infra/metrics"
	"github.com/giovaniif/e-commerce/payment/infra/requestid"
	"github.com/giovaniif/e-commerce/payment/infra/tracing"
	"github.com/giovaniif/e-commerce/payment/protocols"
	charge "github.com/giovaniif/e-commerce/payment/use_cases"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
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

	var chargeGateway protocols.ChargeGateway
	if mongoURL := os.Getenv("MONGO_URL"); mongoURL != "" {
		mongoClient, err := mongo.Connect(options.Client().ApplyURI(mongoURL))
		if err != nil {
			slog.Warn("failed to connect to MongoDB, using in-memory charge gateway", "error", err)
			chargeGateway = gateways.NewChargeGatewayMemory()
		} else if err := mongoClient.Ping(context.Background(), nil); err != nil {
			slog.Warn("failed to ping MongoDB, using in-memory charge gateway", "error", err)
			chargeGateway = gateways.NewChargeGatewayMemory()
		} else {
			chargeGateway = gateways.NewChargeGatewayMongo(mongoClient)
			slog.Info("charge gateway: MongoDB")
		}
	} else {
		slog.Warn("MONGO_URL not set, using in-memory charge gateway")
		chargeGateway = gateways.NewChargeGatewayMemory()
	}

	var idempotencyGateway protocols.IdempotencyGateway
	if redisAddr := os.Getenv("REDIS_ADDR"); redisAddr != "" {
		rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
		if err := rdb.Ping(context.Background()).Err(); err != nil {
			slog.Warn("failed to ping Redis, using in-memory idempotency gateway", "error", err)
			idempotencyGateway = gateways.NewIdempotencyGatewayMemory()
		} else {
			idempotencyGateway = gateways.NewIdempotencyGatewayRedis(rdb)
			slog.Info("idempotency gateway: Redis")
		}
	} else {
		slog.Warn("REDIS_ADDR not set, using in-memory idempotency gateway")
		idempotencyGateway = gateways.NewIdempotencyGatewayMemory()
	}

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
		requestID := requestid.FromContext(c.Request.Context())
		err := chargeUseCase.Charge(charge.ChargeInput{
			IdempotencyKey: idempotencyKey,
			Amount:         chargeRequest.Amount,
		})
		if err != nil {
			slog.ErrorContext(c.Request.Context(), "charge failed", "request_id", requestID, "amount", chargeRequest.Amount, "error", err)
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
