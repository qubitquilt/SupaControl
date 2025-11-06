package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/qubitquilt/supacontrol/server/api"
	"github.com/qubitquilt/supacontrol/server/internal/auth"
	"github.com/qubitquilt/supacontrol/server/internal/config"
	"github.com/qubitquilt/supacontrol/server/internal/db"
	"github.com/qubitquilt/supacontrol/server/internal/k8s"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	log.Println("Starting SupaControl server...")

	// Initialize database
	dbClient, err := db.NewClient(cfg.GetDSN())
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer dbClient.Close()

	log.Println("Connected to database")

	// Run migrations
	migrationsPath := filepath.Join("internal", "db", "migrations")
	if err := dbClient.RunMigrations(migrationsPath); err != nil {
		log.Printf("Warning: failed to run migrations: %v", err)
		log.Println("If this is the first run, ensure migrations are available")
	}

	// Initialize authentication service
	authService := auth.NewService(cfg.JWTSecret)
	log.Println("Initialized authentication service")

	// Initialize Kubernetes client
	k8sClient, err := k8s.NewClient(cfg.KubeConfig)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}
	log.Println("Connected to Kubernetes cluster")

	// Initialize orchestrator
	orchestrator := k8s.NewOrchestrator(
		k8sClient,
		cfg.SupabaseChartRepo,
		cfg.SupabaseChartName,
		cfg.SupabaseChartVersion,
		cfg.DefaultIngressClass,
		cfg.DefaultIngressDomain,
	)
	log.Println("Initialized orchestrator")

	// Initialize Echo server
	e := echo.New()
	e.HideBanner = true

	// Initialize handler
	handler := api.NewHandler(authService, dbClient, orchestrator)

	// Setup routes
	api.SetupRouter(e, handler, authService, dbClient)

	// Serve static files for the UI (if available)
	// This assumes the UI is built and available at ../ui/dist
	uiPath := filepath.Join("..", "ui", "dist")
	if _, err := os.Stat(uiPath); err == nil {
		e.Static("/", uiPath)
		log.Printf("Serving UI from %s", uiPath)
	} else {
		log.Println("UI not found - API only mode")
	}

	// Start server
	go func() {
		addr := cfg.GetServerAddr()
		log.Printf("Server listening on %s", addr)
		if err := e.Start(addr); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Gracefully shutdown the server with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	log.Println("Server stopped")
	return nil
}
