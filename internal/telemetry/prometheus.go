package telemetry

import (
	"runtime"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
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

	goGoroutines = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "go_goroutines",
			Help: "Number of goroutines",
		},
	)

	goMemstatsAlloc = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "go_memstats_alloc_bytes",
			Help: "Number of bytes allocated and still in use",
		},
	)

	goMemstatsHeapAlloc = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "go_memstats_heap_alloc_bytes",
			Help: "Number of heap bytes allocated and still in use",
		},
	)

	goMemstatsHeapSys = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "go_memstats_heap_sys_bytes",
			Help: "Number of heap bytes obtained from the OS",
		},
	)

	goMemstatsHeapInuse = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "go_memstats_heap_inuse_bytes",
			Help: "Number of heap bytes that are in use",
		},
	)

	goMemstatsHeapIdle = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "go_memstats_heap_idle_bytes",
			Help: "Number of heap bytes waiting to be used",
		},
	)

	goMemstatsStackInuse = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "go_memstats_stack_inuse_bytes",
			Help: "Number of bytes used by stack memory",
		},
	)

	goMemstatsSys = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "go_memstats_sys_bytes",
			Help: "Number of bytes obtained from the OS",
		},
	)

	goGcDuration = prometheus.NewSummary(
		prometheus.SummaryOpts{
			Name: "go_gc_duration_seconds",
			Help: "A summary of the pause duration of garbage collection cycles",
		},
	)
)

func init() {
	Registry.MustRegister(httpRequestsTotal)
	Registry.MustRegister(httpRequestDuration)
	Registry.MustRegister(httpRequestsActive)
	Registry.MustRegister(goGoroutines)
	Registry.MustRegister(goMemstatsAlloc)
	Registry.MustRegister(goMemstatsHeapAlloc)
	Registry.MustRegister(goMemstatsHeapSys)
	Registry.MustRegister(goMemstatsHeapInuse)
	Registry.MustRegister(goMemstatsHeapIdle)
	Registry.MustRegister(goMemstatsStackInuse)
	Registry.MustRegister(goMemstatsSys)
	Registry.MustRegister(goGcDuration)
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

func UpdateGoMetrics() {
	goGoroutines.Set(float64(runtime.NumGoroutine()))
	
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	goMemstatsAlloc.Set(float64(m.Alloc))
	goMemstatsHeapAlloc.Set(float64(m.HeapAlloc))
	goMemstatsHeapSys.Set(float64(m.HeapSys))
	goMemstatsHeapInuse.Set(float64(m.HeapInuse))
	goMemstatsHeapIdle.Set(float64(m.HeapIdle))
	goMemstatsStackInuse.Set(float64(m.StackInuse))
	goMemstatsSys.Set(float64(m.Sys))
	goGcDuration.Observe(float64(m.PauseNs[(m.NumGC+255)%256]) / 1e9)
}
