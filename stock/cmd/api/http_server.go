package api

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	kafkaconsumer "github.com/giovaniif/e-commerce/stock/infra/kafka"
	"github.com/giovaniif/e-commerce/stock/infra/loki"
	"github.com/giovaniif/e-commerce/stock/infra/metrics"
	"github.com/giovaniif/e-commerce/stock/infra/repositories"
	"github.com/giovaniif/e-commerce/stock/infra/requestid"
	"github.com/giovaniif/e-commerce/stock/infra/tracing"
	"github.com/giovaniif/e-commerce/stock/use_cases/release_from_event"
	"github.com/giovaniif/e-commerce/stock/use_cases/reserve_from_event"
)

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

	// Start Kafka consumers
	kafkaBrokers := strings.Split(os.Getenv("KAFKA_BROKERS"), ",")
	if len(kafkaBrokers) > 0 && kafkaBrokers[0] != "" {
		reserveHandler := reserve_from_event.NewReserveFromEvent(itemRepository)
		reserveConsumer := kafkaconsumer.NewConsumer(kafkaBrokers, "order.OrderCreated", "stock-order-consumer", reserveHandler)

		releaseHandler := release_from_event.NewReleaseFromEvent(itemRepository)
		releaseConsumer := kafkaconsumer.NewConsumer(kafkaBrokers, "payment.PaymentProcessed", "stock-payment-consumer", releaseHandler)

		ctx, cancelConsumers := context.WithCancel(context.Background())
		go reserveConsumer.Run(ctx)
		go releaseConsumer.Run(ctx)
		defer func() {
			cancelConsumers()
			reserveConsumer.Close()
			releaseConsumer.Close()
		}()
		slog.Info("Kafka consumers started", "brokers", kafkaBrokers)
	} else {
		slog.Warn("KAFKA_BROKERS not set, Kafka consumers not started")
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
