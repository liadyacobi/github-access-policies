package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/liadyacobi/github-access-policies/gen/v1"
	"github.com/liadyacobi/github-access-policies/internal/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "50001" // Default port if not set
	}

	policyDir := os.Getenv("POLICY_DIR")
	if policyDir == "" {
		policyDir = "./policies" // Default policy directory if not set
	}

	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		log.Fatal("GITHUB_TOKEN environment variable is required")
	}

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Create and register the GitHub scanner server
	scannerServer := server.NewGithubScannerServer(githubToken, policyDir)
	pb.RegisterGithubScannerServer(grpcServer, scannerServer)

	// Enable reflection (useful for tools like grpcui)
	reflection.Register(grpcServer)

	// Create listener
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("Failed to create listener: %v", err)
	}

	// Handle graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		log.Printf("Starting gRPC server on port %s", port)
		log.Printf("Using policy directory: %s", policyDir)
		log.Println("Server is ready to accept connections...")

		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-c
	log.Println("Received shutdown signal, gracefully stopping server...")
	grpcServer.GracefulStop()
	log.Println("Server stopped")
}
