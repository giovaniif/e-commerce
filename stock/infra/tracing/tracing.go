package tracing

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

const tracerName = "stock"

var propagator = propagation.NewCompositeTextMapPropagator(
	propagation.TraceContext{},
	propagation.Baggage{},
)

// Init initializes the global tracer provider and returns a shutdown function.
// If OTEL_EXPORTER_OTLP_ENDPOINT is not set, returns nil (tracing disabled).
func Init(serviceName string) func() {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		return nil
	}
	if u, err := parseOTLPEndpoint(endpoint); err == nil {
		endpoint = u
	}
	ctx := context.Background()
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil
	}
	res, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes("", semconv.ServiceNameKey.String(serviceName)),
	)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagator)
	return func() { _ = tp.Shutdown(ctx) }
}

// Middleware returns a Gin middleware that extracts trace context from headers and creates a child span.
func Middleware(serviceName string) gin.HandlerFunc {
	tracer := otel.Tracer(tracerName)
	return func(c *gin.Context) {
		ctx := propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))
		spanName := c.Request.Method + " " + c.FullPath()
		if spanName == " " {
			spanName = c.Request.Method + " " + c.Request.URL.Path
		}
		ctx, span := tracer.Start(ctx, spanName)
		defer span.End()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
		span.SetAttributes(
			attribute.Int("http.status_code", c.Writer.Status()),
			attribute.String("http.method", c.Request.Method),
			attribute.String("http.route", c.Request.URL.Path),
		)
		if c.Writer.Status() >= 400 {
			span.SetStatus(codes.Error, http.StatusText(c.Writer.Status()))
		}
	}
}

func parseOTLPEndpoint(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	if !strings.Contains(raw, "://") {
		return raw, nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "4318"
	}
	return host + ":" + port, nil
}
