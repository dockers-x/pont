package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"pont/internal/config"
	"pont/internal/db"
	"pont/internal/logger"
	"pont/internal/server"
	"pont/internal/service"
	"pont/version"
)

func main() {
	// Get environment variables
	dataDir := getEnv("DATA_DIR", "./data")
	logDir := getEnv("LOG_DIR", filepath.Join(dataDir, "logs"))
	logLevel := getEnv("LOG_LEVEL", "info")
	port := getEnv("PORT", "13333")

	// Ensure directories exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create data directory: %v\n", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create log directory: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logFile := filepath.Join(logDir, "pont.log")
	if err := logger.Init(logLevel, logFile); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Sugar.Infof("Starting Pont %s", version.GetFullVersion())
	logger.Sugar.Infof("Data directory: %s", dataDir)
	logger.Sugar.Infof("Log directory: %s", logDir)

	// Start log cleanup routine
	logger.StartCleanupRoutine()

	// Initialize database
	client, err := db.Init(dataDir)
	if err != nil {
		logger.Sugar.Fatalf("Failed to initialize database: %v", err)
	}
	defer client.Close()

	logger.Sugar.Info("Database initialized successfully")

	// Initialize configuration manager
	cfgMgr := config.NewManager(client)
	logger.Sugar.Info("Configuration manager initialized")

	// Initialize service manager
	svcMgr := service.NewManager(cfgMgr)
	logger.Sugar.Info("Service manager initialized")

	// Initialize HTTP server
	addr := "0.0.0.0:" + port
	srv := server.NewServer(addr, cfgMgr, svcMgr)

	// Start server in goroutine
	go func() {
		logger.Sugar.Infof("HTTP server listening on %s", addr)
		if err := srv.Start(); err != nil {
			logger.Sugar.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Sugar.Info("Shutdown signal received, gracefully shutting down...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop all tunnels
	logger.Sugar.Info("Stopping all tunnels...")
	if err := svcMgr.StopAll(); err != nil {
		logger.Sugar.Warnf("Error stopping tunnels: %v", err)
	}

	// Shutdown HTTP server
	logger.Sugar.Info("Shutting down HTTP server...")
	if err := srv.Shutdown(ctx); err != nil {
		logger.Sugar.Warnf("Error shutting down server: %v", err)
	}

	logger.Sugar.Info("Shutdown complete")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
