package main

import (
	"net"

	pb "github.com/Annany2002/vector-sync/api/proto/v1"
	"github.com/Annany2002/vector-sync/internal/db"
	grpcHandler "github.com/Annany2002/vector-sync/internal/grpc"
	"github.com/Annany2002/vector-sync/internal/logger"
	"github.com/Annany2002/vector-sync/internal/services"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

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
	documentService := services.NewDocumentService(*documentRepo)

	// Create handler layer (handles gRPC requests)
	collectionHandler := grpcHandler.NewCollectionHandler(collectionService)
	documentHandler := grpcHandler.NewDocumentHandler(documentService)

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Register our services with the gRPC server
	pb.RegisterCollectionServiceServer(grpcServer, collectionHandler)
	pb.RegisterDocumentServiceServer(grpcServer, documentHandler)

	// Enable gRPC reflection for grpcurl
	reflection.Register(grpcServer)

	// Start listening on port 6309
	listener, err := net.Listen("tcp", ":6309")
	if err != nil {
		log.Errorf("Failed to listen on port 6309: %v", err)
		return
	}

	log.Infof("VectorSync gRPC server started on :6309")

	// Start serving gRPC requests
	if err := grpcServer.Serve(listener); err != nil {
		log.Errorf("Failed to serve gRPC: %v", err)
	}
}
