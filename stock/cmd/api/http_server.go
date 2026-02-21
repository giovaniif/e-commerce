package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/giovaniif/e-commerce/stock/domain/item"
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
	items := make(map[int32]*item.Item)
	items[1] = &item.Item{Id: 1, Price: 10, InitialStock: 10}
	reservations := make(map[int32]*item.Reservation)
	itemRepository := repositories.NewItemRepository(items, reservations)
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
	slog.Info("stock service started", "port", 3133)

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
		reservation, err := reserveUseCase.Reserve(reserveRequest.ItemId, reserveRequest.Quantity)
		if err != nil {
			switch {
			case errors.Is(err, repositories.ErrItemNotFound):
				c.String(http.StatusNotFound, err.Error())
			case errors.Is(err, repositories.ErrInsufficientStock):
				c.String(http.StatusConflict, err.Error())
			default:
				c.String(http.StatusInternalServerError, err.Error())
			}
			return
		}
		c.JSON(http.StatusOK, gin.H{"reservationId": reservation.ReservationId, "totalFee": reservation.TotalFee})
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
			fmt.Println("failed to bind json")
			c.String(http.StatusBadRequest, err.Error())
			return
		}
		err := completeUseCase.Complete(complete.Input{ReservationId: completeRequest.ReservationId})
		if err != nil {
			fmt.Println("failed to complete stock")
			c.String(http.StatusInternalServerError, err.Error())
		} else {
			fmt.Println("complete successful")
			c.String(http.StatusOK, "Complete successful")
		}
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