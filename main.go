package main

import (
	"go-barcode-server/server"
	"go-barcode-server/web"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Create server instance
	srv := server.NewServer()

	// Start TCP server
	go func() {
		if err := srv.StartTCPServer(":8081"); err != nil {
			log.Fatalf("Failed to start TCP server: %v", err)
		}
	}()

	// Start COM port monitoring
	go srv.MonitorCOMPort()

	// Setup web handlers
	webHandler := web.NewWebHandler(srv)

	// Start HTTP server for web interface
	go func() {
		log.Println("Starting web server on http://localhost:8080")
		if err := http.ListenAndServe(":8080", webHandler); err != nil {
			log.Fatalf("Failed to start web server: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down server...")
	srv.Stop()
}
