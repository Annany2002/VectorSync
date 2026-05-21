package metrics

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// UnaryServerInterceptor records latency, status code, and in-flight count
// for every gRPC unary request.
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		method := info.FullMethod
		RPCInflight.WithLabelValues(method).Inc()
		start := time.Now()

		resp, err := handler(ctx, req)

		RPCInflight.WithLabelValues(method).Dec()
		code := status.Code(err).String()
		RPCDuration.WithLabelValues(method, code).Observe(time.Since(start).Seconds())
		return resp, err
	}
}
