package grpc

import (
	"context"

	pb "github.com/Annany2002/vector-sync/api/proto/v1/generated"
	"github.com/Annany2002/vector-sync/internal/services"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// HealthHandler handles gRPC health check requests
type HealthHandler struct {
	pb.UnimplementedHealthServiceServer
	service *services.HealthService
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(service *services.HealthService) *HealthHandler {
	return &HealthHandler{
		service: service,
	}
}

// Check performs a health check
func (h *HealthHandler) Check(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	// Determine which health check to perform based on service name
	// Empty string or "liveness" = liveness check
	// "readiness" = readiness check

	if req.Service == "readiness" {
		// Perform readiness check
		ready, err := h.service.CheckReadiness()
		if err != nil || !ready {
			return &pb.HealthCheckResponse{
				Status: pb.ServingStatus_NOT_SERVING,
			}, nil
		}
		return &pb.HealthCheckResponse{
			Status: pb.ServingStatus_SERVING,
		}, nil
	}

	// Default: perform liveness check
	alive, err := h.service.CheckLiveness()
	if err != nil || !alive {
		return nil, status.Error(codes.Internal, "liveness check failed")
	}

	return &pb.HealthCheckResponse{
		Status: pb.ServingStatus_SERVING,
	}, nil
}
