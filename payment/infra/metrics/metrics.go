package metrics

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	RequestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)
	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

func NormalizePath(p string) string {
	p = strings.TrimPrefix(p, "/")
	if idx := strings.Index(p, "/"); idx >= 0 {
		p = p[:idx]
	}
	if p == "" {
		return "root"
	}
	return p
}

func Middleware(c *gin.Context) {
	if c.Request.URL.Path == "/metrics" {
		c.Next()
		return
	}
	start := time.Now()
	c.Next()
	duration := time.Since(start).Seconds()
	path := NormalizePath(c.Request.URL.Path)
	status := strconv.Itoa(c.Writer.Status())
	RequestTotal.WithLabelValues(c.Request.Method, path, status).Inc()
	RequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
}
