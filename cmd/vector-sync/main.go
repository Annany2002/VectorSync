package main

import (
	"context"
	"net"      // for grpc server
	"net/http" // for http gateway

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
	defer dbConn.Close()

	// Create repository layer (talks to database)
	collectionRepo := db.NewCollectionRepo(dbConn)
	documentRepo := db.NewDocumentRepo(dbConn)

	// Create service layer (business logic)
	collectionService := services.NewCollectionService(*collectionRepo)
	documentService := services.NewDocumentService(*documentRepo, *collectionRepo)
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

	// Create the grpc gateway within a go-routine
	go func() {
		// Start serving gRPC requests
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

	log.Infof("VectorSync gRPC server started on :6309")
	log.Infof("VectorSync HTTP gateway started on :8080")

	// Start HTTP server (blocks on main thread)
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Failed to serve HTTP gateway: %v", err)
	}
}
