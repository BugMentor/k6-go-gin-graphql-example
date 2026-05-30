package telemetry

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var Registry = prometheus.NewRegistry()

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_server_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_server_requests_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{0.01, 0.05, 0.1, 0.2, 0.5, 1.0, 2.0, 5.0},
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestsActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_server_requests_active",
			Help: "Number of active HTTP requests",
		},
	)
)

func init() {
	Registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	Registry.MustRegister(collectors.NewGoCollector())
	Registry.MustRegister(httpRequestsTotal)
	Registry.MustRegister(httpRequestDuration)
	Registry.MustRegister(httpRequestsActive)
}

func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		httpRequestsActive.Inc()

		c.Next()

		httpRequestsActive.Dec()
		status := strconv.Itoa(c.Writer.Status())
		duration := time.Since(start).Seconds()

		endpoint := c.FullPath()
		if endpoint == "" {
			endpoint = "unknown"
		}

		httpRequestsTotal.WithLabelValues(c.Request.Method, endpoint, status).Inc()
		httpRequestDuration.WithLabelValues(c.Request.Method, endpoint, status).Observe(duration)
	}
}

func PrometheusHandler() gin.HandlerFunc {
	h := promhttp.HandlerFor(Registry, promhttp.HandlerOpts{})
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}
