package metrics

import (
	"net/http"
	"strconv"
	"time"
)

// statusRecorder captures the HTTP status code written upstream so the
// middleware can label its latency histogram.
type statusRecorder struct {
	http.ResponseWriter
	code int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.code = code
	r.ResponseWriter.WriteHeader(code)
}

// HTTPMiddleware records HTTP gateway request duration, labelled by status code.
// Path is intentionally not labelled to keep cardinality bounded; per-RPC
// latency is captured by UnaryServerInterceptor below.
func HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, code: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(rec, r)
		HTTPDuration.WithLabelValues(strconv.Itoa(rec.code)).Observe(time.Since(start).Seconds())
	})
}
