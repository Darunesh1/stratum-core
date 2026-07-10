package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"context"
	"stratum/api"
)

func main() {
	port := flag.Int("port", 8080, "Port to run the HTTP server on")
	dbPath := flag.String("db", "data/db/papers.db", "Path to the DuckDB database file")
	flag.Parse()

	args := flag.Args()
	if len(args) > 0 && args[0] == "mcp" {
		fmt.Fprintln(os.Stderr, "Starting Stratum MCP Server (stdio mode)...")
		srv := api.NewAPIServer("", *dbPath)
		ctx := context.Background()
		if err := srv.RunMCPStdio(ctx); err != nil {
			log.Fatalf("MCP server failed: %v", err)
		}
		return
	}

	fmt.Printf("Starting Stratum Web Server on http://localhost:%d...\n", *port)
	fmt.Printf("Integrated MCP SSE Server listening at http://localhost:%d/api/mcp\n", *port)
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
}
