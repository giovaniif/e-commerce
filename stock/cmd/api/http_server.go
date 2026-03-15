package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/giovaniif/e-commerce/stock/infra/gateways"
	"github.com/giovaniif/e-commerce/stock/infra/loki"
	"github.com/giovaniif/e-commerce/stock/infra/metrics"
	"github.com/giovaniif/e-commerce/stock/infra/repositories"
	"github.com/giovaniif/e-commerce/stock/infra/requestid"
	"github.com/giovaniif/e-commerce/stock/infra/tracing"
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
	initialStock := int32(10)
	if s := os.Getenv("STOCK_INITIAL_QUANTITY"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 32); err == nil && n > 0 {
			initialStock = int32(n)
		}
	}
	_ = initialStock // initial stock is seeded via init.sql

	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		fmt.Println("POSTGRES_URL is required")
		os.Exit(1)
	}
	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		fmt.Printf("Failed to open postgres: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()
	db.SetMaxOpenConns(80)
	db.SetMaxIdleConns(40)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err := db.Ping(); err != nil {
		fmt.Printf("Failed to ping postgres: %v\n", err)
		os.Exit(1)
	}
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		fmt.Println("REDIS_ADDR is required")
		os.Exit(1)
	}
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		fmt.Printf("Failed to ping Redis (%s): %v\n", redisAddr, err)
		os.Exit(1)
	}

	itemRepository := repositories.NewItemRepositoryPostgres(db, rdb)
	if err := itemRepository.SeedStockCounters(context.Background()); err != nil {
		fmt.Printf("Failed to seed stock counters: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Stock counters seeded in Redis")

	idempotencyGateway := gateways.NewIdempotencyGatewayRedis(rdb)
	reserveUseCase := reserve.NewReserve(itemRepository)
	releaseUseCase := release.NewRelease(itemRepository)
	completeUseCase := complete.NewComplete(itemRepository)

	logOut := io.Writer(os.Stdout)
	var lokiWriter *loki.Writer
	if lokiURL := os.Getenv("LOKI_URL"); lokiURL != "" {
		if lw := loki.NewWriter(lokiURL, "stock"); lw != nil {
			lokiWriter = lw
			logOut = io.MultiWriter(os.Stdout, lw)
		}
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(logOut, &slog.HandlerOptions{Level: slog.LevelInfo})))
	slog.Info("stock service started", "port", 3133, "initial_stock_item_1", initialStock)

	shutdownTracing := tracing.Init("stock")
	if shutdownTracing != nil {
		defer shutdownTracing()
	}

	r := gin.Default()
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
	r.Use(tracing.Middleware("stock"))
	r.Use(metrics.Middleware)

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	r.POST("/reserve", func(c *gin.Context) {
		var reserveRequest ReserveRequest
		if err := c.ShouldBindJSON(&reserveRequest); err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}
		ctx := c.Request.Context()
		requestID := requestid.FromContext(ctx)
		if idempotencyGateway != nil && requestID != "" {
			if resId, fee, found, err := idempotencyGateway.ReserveIdempotency(ctx, requestID); err == nil && found {
				c.JSON(http.StatusOK, gin.H{"reservationId": resId, "totalFee": fee})
				return
			}
		}
		reservation, err := reserveUseCase.Reserve(reserveRequest.ItemId, reserveRequest.Quantity)
		if err != nil {
			switch {
			case errors.Is(err, repositories.ErrItemNotFound):
				slog.ErrorContext(ctx, "reserve failed: item not found", "request_id", requestID, "item_id", reserveRequest.ItemId, "quantity", reserveRequest.Quantity, "error", err)
				c.String(http.StatusNotFound, err.Error())
			case errors.Is(err, repositories.ErrInsufficientStock):
				slog.WarnContext(ctx, "reserve failed: insufficient stock", "request_id", requestID, "item_id", reserveRequest.ItemId, "quantity", reserveRequest.Quantity)
				c.String(http.StatusConflict, err.Error())
			default:
				slog.ErrorContext(ctx, "reserve failed", "request_id", requestID, "item_id", reserveRequest.ItemId, "quantity", reserveRequest.Quantity, "error", err)
				c.String(http.StatusInternalServerError, err.Error())
			}
			return
		}
		if idempotencyGateway != nil && requestID != "" {
			_ = idempotencyGateway.SaveReserveResult(ctx, requestID, reservation.ReservationId, reservation.TotalFee)
		}
		c.JSON(http.StatusOK, gin.H{"reservationId": reservation.ReservationId, "totalFee": reservation.TotalFee})
	})

	r.POST("/release", func(c *gin.Context) {
		var releaseRequest ReleaseRequest
		if err := c.ShouldBindJSON(&releaseRequest); err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}
		ctx := c.Request.Context()
		if idempotencyGateway != nil {
			if found, err := idempotencyGateway.ReleaseIdempotency(ctx, releaseRequest.ReservationId); err == nil && found {
				c.String(http.StatusOK, "Release successful")
				return
			}
		}
		err := releaseUseCase.Release(release.Input{ReservationId: releaseRequest.ReservationId})
		if err != nil {
			slog.ErrorContext(ctx, "release failed", "request_id", requestid.FromContext(ctx), "reservation_id", releaseRequest.ReservationId, "error", err)
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		if idempotencyGateway != nil {
			_ = idempotencyGateway.SaveReleaseResult(ctx, releaseRequest.ReservationId)
		}
		c.String(http.StatusOK, "Release successful")
	})

	r.POST("/complete", func(c *gin.Context) {
		var completeRequest CompleteRequest
		if err := c.ShouldBindJSON(&completeRequest); err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}
		ctx := c.Request.Context()
		if idempotencyGateway != nil {
			if found, err := idempotencyGateway.CompleteIdempotency(ctx, completeRequest.ReservationId); err == nil && found {
				c.String(http.StatusOK, "Complete successful")
				return
			}
		}
		err := completeUseCase.Complete(complete.Input{ReservationId: completeRequest.ReservationId})
		if err != nil {
			slog.ErrorContext(ctx, "complete failed", "request_id", requestid.FromContext(ctx), "reservation_id", completeRequest.ReservationId, "error", err)
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		if idempotencyGateway != nil {
			_ = idempotencyGateway.SaveCompleteResult(ctx, completeRequest.ReservationId)
		}
		c.String(http.StatusOK, "Complete successful")
	})

	srv := &http.Server{Addr: ":3133", Handler: r}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Stock server: %v\n", err)
		}
	}()
	fmt.Println("Stock is running on port 3133")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	fmt.Println("Stock shutting down...")
	if shutdownTracing != nil {
		shutdownTracing()
	}
	if lokiWriter != nil {
		_ = lokiWriter.Close()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		fmt.Printf("Stock shutdown: %v\n", err)
	} else {
		fmt.Println("Stock stopped")
	}
}