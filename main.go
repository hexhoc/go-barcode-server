package main

import (
	"flag"
	"fmt"
	"go-barcode-server/server"
	"go-barcode-server/web"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

const (
	defaultTCPPort = "52000"
	defaultWebPort = "8080"
)

type Config struct {
	TCPPort string
	WebPort string
}

func parseFlags() Config {
	tcpPort := flag.String("tcp", defaultTCPPort, "TCP server port (e.g., 52000)")
	webPort := flag.String("web", defaultWebPort, "Web server port (e.g., 8080)")
	flag.Parse()

	return Config{
		TCPPort: normalizePort(*tcpPort),
		WebPort: normalizePort(*webPort),
	}
}

func normalizePort(port string) string {
	return ":" + port
}

func startTCPServer(srv *server.Server, port string) {
	log.Printf("Starting TCP server on %s", port)
	if err := srv.StartTCPServer(port); err != nil {
		log.Fatalf("Failed to start TCP server: %v", err)
	}
}

func startWebServer(webHandler http.Handler, port string) {
	log.Printf("Starting web server on http://localhost%s", port)
	if err := http.ListenAndServe(port, webHandler); err != nil {
		log.Fatalf("Failed to start web server: %v", err)
	}
}

func waitForShutdownSignal() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	log.Println("Received shutdown signal")
}

func main() {
	// Parse configuration
	config := parseFlags()

	fmt.Printf("Server configuration:\n")
	fmt.Printf("  TCP Port: %s\n", config.TCPPort)
	fmt.Printf("  Web Port: %s\n", config.WebPort)

	// Initialize server
	serverInstance := server.NewServer()

	// Start TCP server in background
	go startTCPServer(serverInstance, config.TCPPort)

	// Start COM port monitoring in background
	go serverInstance.MonitorCOMPort()

	// Initialize and start web server
	webHandler := web.NewWebHandler(serverInstance)
	go startWebServer(webHandler, config.WebPort)

	// Wait for graceful shutdown
	waitForShutdownSignal()

	log.Println("Shutting down server...")
	serverInstance.Stop()
	log.Println("Server shutdown complete")
}
