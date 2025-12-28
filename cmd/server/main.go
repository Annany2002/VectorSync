package main

import (
	"net"

	pb "github.com/Annany2002/vector-sync/api/proto/v1"
	"github.com/Annany2002/vector-sync/internal/db"
	grpcHandler "github.com/Annany2002/vector-sync/internal/grpc"
	"github.com/Annany2002/vector-sync/internal/logger"
	"github.com/Annany2002/vector-sync/internal/services"
	"google.golang.org/grpc"
)

var (
	log = logger.NewLogger()
)

func main() {
	// Step 1: Connect to database
	log.Infof("Connecting to database...")
	dbConn, err := db.Connect()
	if err != nil {
		log.Errorf("Failed to connect to database: %v", err)
		return
	}
	defer dbConn.Close()
	log.Infof("Database connected successfully")

	// Step 2: Create repository layer (talks to database)
	collectionRepo := db.NewCollectionRepo(dbConn)

	// Step 3: Create service layer (business logic)
	collectionService := services.NewCollectionService(*collectionRepo)

	// Step 4: Create handler layer (handles gRPC requests)
	collectionHandler := grpcHandler.NewCollectionHandler(collectionService)

	// Step 5: Create gRPC server
	grpcServer := grpc.NewServer()

	// Step 6: Register our collection service with the gRPC server
	pb.RegisterCollectionServiceServer(grpcServer, collectionHandler)

	// Step 7: Start listening on port 6309
	listener, err := net.Listen("tcp", ":6309")
	if err != nil {
		log.Errorf("Failed to listen on port 6309: %v", err)
		return
	}

	log.Infof("VectorSync gRPC server started on :6309")

	// Step 8: Start serving gRPC requests (this blocks until shutdown)
	if err := grpcServer.Serve(listener); err != nil {
		log.Errorf("Failed to serve gRPC: %v", err)
	}
}
