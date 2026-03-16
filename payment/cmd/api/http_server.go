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
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/giovaniif/e-commerce/payment/infra/gateways"
	kafkaconsumer "github.com/giovaniif/e-commerce/payment/infra/kafka"
	"github.com/giovaniif/e-commerce/payment/infra/loki"
	"github.com/giovaniif/e-commerce/payment/infra/metrics"
	"github.com/giovaniif/e-commerce/payment/infra/repositories"
	"github.com/giovaniif/e-commerce/payment/infra/requestid"
	"github.com/giovaniif/e-commerce/payment/infra/tracing"
	"github.com/giovaniif/e-commerce/payment/protocols"
	charge_from_event "github.com/giovaniif/e-commerce/payment/use_cases"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

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

	// Postgres for outbox and processed_events
	var db *sql.DB
	if postgresURL := os.Getenv("POSTGRES_URL"); postgresURL != "" {
		var err error
		db, err = sql.Open("postgres", postgresURL)
		if err != nil {
			slog.Warn("failed to open postgres", "error", err)
		} else {
			db.SetMaxOpenConns(40)
			db.SetMaxIdleConns(20)
			db.SetConnMaxLifetime(5 * time.Minute)
			if err := db.Ping(); err != nil {
				slog.Warn("failed to ping postgres", "error", err)
				db = nil
			} else {
				slog.Info("postgres connected for payment outbox")
				defer db.Close()
			}
		}
	}

	// Start Kafka consumer
	kafkaBrokers := strings.Split(os.Getenv("KAFKA_BROKERS"), ",")
	if len(kafkaBrokers) > 0 && kafkaBrokers[0] != "" && db != nil {
		outboxRepo := repositories.NewOutboxRepository(db)
		processedRepo := repositories.NewProcessedEventsRepository(db)
		chargeHandler := charge_from_event.NewChargeFromEvent(chargeGateway, outboxRepo, processedRepo, db)
		consumer := kafkaconsumer.NewConsumer(kafkaBrokers, "stock.StockReserved", "payment-stock-consumer", chargeHandler)

		ctx, cancelConsumer := context.WithCancel(context.Background())
		go consumer.Run(ctx)
		defer func() {
			cancelConsumer()
			consumer.Close()
		}()
		slog.Info("Kafka consumer started", "brokers", kafkaBrokers, "topic", "stock.StockReserved")
	} else {
		slog.Warn("KAFKA_BROKERS not set or postgres unavailable, Kafka consumer not started")
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
