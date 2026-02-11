package main

import (
	"context"
	"net"      // for grpc server
	"net/http" // for http gateway
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/Annany2002/vector-sync/api/proto/v1/generated"
	"github.com/Annany2002/vector-sync/internal/db"
	grpcHandler "github.com/Annany2002/vector-sync/internal/grpc"
	"github.com/Annany2002/vector-sync/internal/logger"
	"github.com/Annany2002/vector-sync/internal/services"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

// Initialze the logger
var (
	log = logger.NewLogger()
)

func main() {
	// Connect to database
	log.Infof("Connecting to database...")
	dbConn, err := db.Connect()
	if err != nil {
		log.Errorf("Failed to connect to database: %v", err)
		return
	}

	// Create repository layer (talks to database)
	collectionRepo := db.NewCollectionRepo(dbConn)
	documentRepo := db.NewDocumentRepo(dbConn)
	collectionCache := db.NewCollectionCache()

	// Create service layer (business logic)
	collectionService := services.NewCollectionService(*collectionRepo)
	documentService := services.NewDocumentService(*documentRepo, *collectionRepo, collectionCache)
	healthService := services.NewHealthService(dbConn)

	// Create handler layer (handles gRPC requests)
	collectionHandler := grpcHandler.NewCollectionHandler(collectionService)
	documentHandler := grpcHandler.NewDocumentHandler(documentService, collectionService)
	healthHandler := grpcHandler.NewHealthHandler(healthService)

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Register our services with the gRPC server
	pb.RegisterCollectionServiceServer(grpcServer, collectionHandler)
	pb.RegisterDocumentServiceServer(grpcServer, documentHandler)
	pb.RegisterHealthServiceServer(grpcServer, healthHandler)

	// Enable gRPC reflection for grpcurl
	reflection.Register(grpcServer)

	// Start listening on port 6309
	listener, err := net.Listen("tcp", ":6309")
	if err != nil {
		log.Errorf("Failed to listen on port 6309: %v", err)
		return
	}

	// Start gRPC server in a goroutine
	go func() {
		log.Infof("VectorSync gRPC server started on :6309")
		if err := grpcServer.Serve(listener); err != nil {
			log.Errorf("Failed to serve gRPC: %v", err)
		}
	}()

	// Create HTTP gateway mux
	mux := runtime.NewServeMux()

	// Register CollectionService gateway handler
	err = pb.RegisterCollectionServiceHandlerFromEndpoint(
		context.Background(),
		mux,
		"localhost:6309",
		[]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())},
	)
	if err != nil {
		log.Fatalf("Failed to register collection gateway: %v", err)
	}

	// Register DocumentService gateway handler
	err = pb.RegisterDocumentServiceHandlerFromEndpoint(
		context.Background(),
		mux,
		"localhost:6309",
		[]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())},
	)
	if err != nil {
		log.Fatalf("Failed to register document gateway: %v", err)
	}

	// Register HealthService gateway handler
	err = pb.RegisterHealthServiceHandlerFromEndpoint(
		context.Background(),
		mux,
		"localhost:6309",
		[]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())},
	)
	if err != nil {
		log.Fatalf("Failed to register health gateway: %v", err)
	}

	// Create HTTP server with explicit configuration
	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// Start HTTP server in a goroutine
	go func() {
		log.Infof("VectorSync HTTP gateway started on :8080")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to serve HTTP gateway: %v", err)
		}
	}()

	// Set up signal handling for graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Block until we receive a shutdown signal
	sig := <-stop
	log.Infof("Received signal %v, initiating graceful shutdown...", sig)

	// Create a context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown HTTP server first (stop accepting new requests)
	log.Infof("Shutting down HTTP gateway...")
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Errorf("HTTP server shutdown error: %v", err)
	} else {
		log.Infof("HTTP gateway shutdown complete")
	}

	// Gracefully stop gRPC server (waits for active RPCs to complete)
	log.Infof("Shutting down gRPC server...")
	grpcServer.GracefulStop()
	log.Infof("gRPC server shutdown complete")

	// Close database connection
	log.Infof("Closing database connection...")
	if err := dbConn.Close(); err != nil {
		log.Errorf("Database close error: %v", err)
	} else {
		log.Infof("Database connection closed")
	}

	log.Infof("VectorSync shutdown complete")
}
