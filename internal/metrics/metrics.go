// Package metrics defines Prometheus collectors and helpers for VectorSync.
//
// All collectors are registered with the default prometheus registry on package
// init. Import the package for its side effects, then mount promhttp.Handler()
// on the gateway and wire UnaryServerInterceptor() into the gRPC server.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const namespace = "vectorsync"

var (
	// RPCDuration histograms gRPC unary call latency, labelled by full method
	// (e.g. "/vectorsync.v1.DocumentService/Search") and grpc status code.
	RPCDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: "grpc",
		Name:      "request_duration_seconds",
		Help:      "Duration of gRPC unary requests.",
		Buckets:   prometheus.ExponentialBuckets(0.001, 2, 14), // 1ms .. ~16s
	}, []string{"method", "code"})

	// RPCInflight tracks currently-executing gRPC unary calls per method.
	RPCInflight = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "grpc",
		Name:      "requests_inflight",
		Help:      "In-flight gRPC unary requests.",
	}, []string{"method"})

	// HTTPDuration histograms HTTP gateway latency by status code.
	// Path label intentionally omitted to bound cardinality; the gRPC
	// interceptor already records per-method latency below it.
	HTTPDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: "http",
		Name:      "request_duration_seconds",
		Help:      "Duration of HTTP gateway requests.",
		Buckets:   prometheus.ExponentialBuckets(0.001, 2, 14),
	}, []string{"code"})

	// CacheOps tracks collection-cache hit/miss counts. Label values: "hit", "miss".
	CacheOps = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "cache",
		Name:      "ops_total",
		Help:      "Collection cache operations.",
	}, []string{"result"})

	// HNSWBuildDuration histograms background HNSW index build wall time.
	HNSWBuildDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: "hnsw",
		Name:      "build_duration_seconds",
		Help:      "Async HNSW index build duration.",
		Buckets:   prometheus.ExponentialBuckets(0.01, 2, 12), // 10ms .. ~40s
	})

	// HNSWBuildErrors counts failed background HNSW builds.
	HNSWBuildErrors = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "hnsw",
		Name:      "build_errors_total",
		Help:      "Failed async HNSW index builds.",
	})

	// IngestChunks counts total chunks produced by the ingestion pipeline.
	IngestChunks = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "ingest",
		Name:      "chunks_total",
		Help:      "Chunks produced during ingestion, labelled by chunker strategy.",
	}, []string{"strategy"})

	// IngestEmbedCalls counts embedding-provider invocations.
	IngestEmbedCalls = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "ingest",
		Name:      "embed_calls_total",
		Help:      "Embedding-provider calls during ingestion.",
	}, []string{"provider", "status"})

	// IngestEmbedDuration histograms time spent inside the embedding client.
	IngestEmbedDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: "ingest",
		Name:      "embed_duration_seconds",
		Help:      "Embedding-provider call duration.",
		Buckets:   prometheus.ExponentialBuckets(0.01, 2, 12),
	}, []string{"provider"})
)
