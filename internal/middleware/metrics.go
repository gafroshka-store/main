package middleware

import (
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Histogram of response time for handler",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	httpInFlightRequests = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_in_flight_requests",
			Help: "Current number of HTTP requests being handled",
		},
	)

	httpErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_errors_total",
			Help: "Total number of HTTP error responses (status 4xx and 5xx)",
		},
		[]string{"method", "path", "status"},
	)

	goGoroutines = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "go_goroutines",
			Help: "Number of goroutines",
		},
	)

	goMemStatsHeapAlloc = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "go_memstats_heap_alloc_bytes",
			Help: "Number of heap bytes allocated and still in use",
		},
	)

	goMemStatsStackInuse = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "go_memstats_stack_inuse_bytes",
			Help: "Bytes in stack spans in use",
		},
	)
)

func init() {
	prometheus.MustRegister(
		httpRequestsTotal,
		httpRequestDuration,
		httpInFlightRequests,
		httpErrorsTotal,
		goGoroutines,
		goMemStatsHeapAlloc,
		goMemStatsStackInuse,
	)

	// Запускаем сборку метрик runtime в фоне
	go collectGoRuntimeMetrics()
}

// Функция для периодического обновления метрик runtime
func collectGoRuntimeMetrics() {
	for {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		goGoroutines.Set(float64(runtime.NumGoroutine()))
		goMemStatsHeapAlloc.Set(float64(m.HeapAlloc))
		goMemStatsStackInuse.Set(float64(m.StackInuse))

		time.Sleep(10 * time.Second)
	}
}

func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpInFlightRequests.Inc()
		start := time.Now()

		rr := &responseRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(rr, r)

		duration := time.Since(start).Seconds()

		httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, strconv.Itoa(rr.status)).Inc()
		httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)

		if rr.status >= 400 {
			httpErrorsTotal.WithLabelValues(r.Method, r.URL.Path, strconv.Itoa(rr.status)).Inc()
		}

		httpInFlightRequests.Dec()
	})
}

type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.status = code
	rr.ResponseWriter.WriteHeader(code)
}
