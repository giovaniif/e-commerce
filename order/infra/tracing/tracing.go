package tracing

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "order"

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
	// WithEndpoint expects host:port (no scheme). Strip URL scheme if present.
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

// Middleware returns a Gin middleware that creates a span per request.
// If X-Request-ID is present and valid (32 hex chars), it is used as trace_id for log/trace correlation.
func Middleware(serviceName string) gin.HandlerFunc {
	tracer := otel.Tracer(tracerName)
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		requestID := c.GetHeader("X-Request-ID")
		if requestID != "" && len(requestID) == 32 {
			if tid, err := trace.TraceIDFromHex(requestID); err == nil {
				var spanID trace.SpanID
				if _, err := hex.Decode(spanID[:], []byte(requestID[16:32])); err != nil {
					rand.Read(spanID[:])
				}
				sc := trace.NewSpanContext(trace.SpanContextConfig{
					TraceID:    tid,
					SpanID:     spanID,
					TraceFlags: trace.FlagsSampled,
					Remote:     true,
				})
				ctx = trace.ContextWithRemoteSpanContext(ctx, sc)
			}
		}
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

// Inject sets trace context into headers for outgoing HTTP requests (e.g. Order → Stock/Payment).
func Inject(ctx context.Context, header http.Header) {
	propagator.Inject(ctx, propagation.HeaderCarrier(header))
}

// parseOTLPEndpoint returns "host:port" from OTEL_EXPORTER_OTLP_ENDPOINT (e.g. "http://tempo:4318" -> "tempo:4318").
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
