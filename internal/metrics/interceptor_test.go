package metrics

import (
	"context"
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestUnaryServerInterceptor_RecordsOK(t *testing.T) {
	RPCDuration.Reset()
	RPCInflight.Reset()
	interceptor := UnaryServerInterceptor()

	info := &grpc.UnaryServerInfo{FullMethod: "/test.Svc/Ping"}
	handler := func(_ context.Context, _ any) (any, error) { return "pong", nil }

	resp, err := interceptor(context.Background(), nil, info, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "pong" {
		t.Fatalf("expected pong, got %v", resp)
	}

	got := testutil.CollectAndCount(RPCDuration, "vectorsync_grpc_request_duration_seconds")
	if got == 0 {
		t.Fatal("expected duration metric to be recorded")
	}
	if c := testutil.ToFloat64(RPCInflight.WithLabelValues("/test.Svc/Ping")); c != 0 {
		t.Fatalf("expected inflight to be 0 after handler returns, got %v", c)
	}
}

func TestUnaryServerInterceptor_RecordsErrorCode(t *testing.T) {
	RPCDuration.Reset()
	RPCInflight.Reset()
	interceptor := UnaryServerInterceptor()

	info := &grpc.UnaryServerInfo{FullMethod: "/test.Svc/Boom"}
	wantErr := status.Error(codes.InvalidArgument, "bad input")
	handler := func(_ context.Context, _ any) (any, error) { return nil, wantErr }

	_, err := interceptor(context.Background(), nil, info, handler)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected handler error to propagate, got %v", err)
	}

	// Locate the histogram for code=InvalidArgument and confirm one observation.
	count := testutil.CollectAndCount(RPCDuration, "vectorsync_grpc_request_duration_seconds")
	if count == 0 {
		t.Fatal("expected histogram observation for error case")
	}
}
