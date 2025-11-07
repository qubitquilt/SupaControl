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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/qubitquilt/supacontrol/server/api"
	supacontrolv1alpha1 "github.com/qubitquilt/supacontrol/server/api/v1alpha1"
	"github.com/qubitquilt/supacontrol/server/controllers"
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

	// Initialize CR client for API handlers
	crClient, err := k8s.NewCRClient(k8sClient.GetConfig())
	if err != nil {
		return fmt.Errorf("failed to create CR client: %w", err)
	}
	log.Println("Initialized CR client")

	// Set up controller manager
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	mgr, err := ctrl.NewManager(k8sClient.GetConfig(), ctrl.Options{
		Scheme: crClient.GetScheme(),
		// LeaderElection can be enabled for HA deployments
		LeaderElection:   false,
		LeaderElectionID: "supacontrol-leader-election",
	})
	if err != nil {
		return fmt.Errorf("failed to create controller manager: %w", err)
	}

	// Register the CRD scheme
	if err := supacontrolv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return fmt.Errorf("failed to add scheme: %w", err)
	}

	// Set up the controller
	reconciler := &controllers.SupabaseInstanceReconciler{
		Client:               mgr.GetClient(),
		Scheme:               mgr.GetScheme(),
		ChartRepo:            cfg.SupabaseChartRepo,
		ChartName:            cfg.SupabaseChartName,
		ChartVersion:         cfg.SupabaseChartVersion,
		DefaultIngressClass:  cfg.DefaultIngressClass,
		DefaultIngressDomain: cfg.DefaultIngressDomain,
		CertManagerIssuer:    cfg.CertManagerIssuer,
	}

	if err := reconciler.SetupWithManager(mgr); err != nil {
		return fmt.Errorf("failed to setup controller: %w", err)
	}

	log.Println("Initialized controller manager")

	// Channel for internal errors that should trigger shutdown
	errChan := make(chan error, 1)

	// Start controller manager in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		log.Println("Starting controller manager...")
		if err := mgr.Start(ctx); err != nil {
			errChan <- fmt.Errorf("controller manager error: %w", err)
		}
	}()

	// Wait for controller cache to sync
	log.Println("Waiting for controller cache to sync...")
	if !mgr.GetCache().WaitForCacheSync(ctx) {
		return fmt.Errorf("failed to sync controller cache")
	}
	log.Println("Controller cache synced")

	// Initialize Echo server
	e := echo.New()
	e.HideBanner = true

	// Initialize handler with CR client
	handler := api.NewHandler(authService, dbClient, crClient)

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

	// Channel for shutdown signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Start server
	go func() {
		addr := cfg.GetServerAddr()
		log.Printf("Server listening on %s", addr)
		if err := e.Start(addr); err != nil {
			errChan <- fmt.Errorf("server error: %w", err)
		}
	}()

	// Wait for shutdown signal or error
	select {
	case <-quit:
		log.Println("Received shutdown signal")
	case err := <-errChan:
		log.Printf("Internal error: %v", err)
	}

	log.Println("Shutting down server...")

	// Stop controller manager first
	cancel()
	log.Println("Controller manager stopped")

	// Gracefully shutdown the HTTP server with a timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	log.Println("Server stopped")
	return nil
}
