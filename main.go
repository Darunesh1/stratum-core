package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"stratum/api"
	"stratum/mcp"
)

func main() {
	serveMode := flag.Bool("serve", false, "Start the web HTTP API server and dashboard")
	port := flag.Int("port", 8080, "Port to run the HTTP server on")
	dbPath := flag.String("db", "data/db/papers.db", "Path to the DuckDB database file")
	flag.Parse()

	ctx := context.Background()

	if *serveMode {
		fmt.Printf("Starting Stratum Web Server on http://localhost:%d...\n", *port)
		addr := fmt.Sprintf(":%d", *port)
		srv := api.NewAPIServer(addr, *dbPath)
		if err := srv.RegisterRoutes(); err != nil {
			log.Fatalf("Failed to register routes: %v", err)
		}
		if err := srv.Start(); err != nil {
			log.Fatalf("HTTP server failed: %v", err)
		}

		// Wait for interrupt signal to gracefully shutdown
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
		<-stop

		log.Println("Shutting down Stratum Web Server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Stop(shutdownCtx); err != nil {
			log.Printf("Graceful shutdown failed: %v", err)
		}
		log.Println("Server exited cleanly")
	} else {
		// Default: Run as an MCP Server
		fmt.Fprintln(os.Stderr, "Starting Stratum MCP Server (stdio mode)...")
		server := mcp.NewMCPServer("stratum-mcp", "1.0.0")
		if err := server.RegisterTools(); err != nil {
			log.Fatalf("Failed to register MCP tools: %v", err)
		}
		if err := server.Start(ctx); err != nil {
			log.Fatalf("MCP server failed: %v", err)
		}
	}
}
