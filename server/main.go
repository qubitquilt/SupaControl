// Package main provides the entry point and main application logic for the SupaControl server.
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

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/labstack/echo/v4"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	batchv1 "k8s.io/api/batch/v1"
	coordinationv1 "k8s.io/api/coordination/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	storagev1 "k8s.io/api/storage/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

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
	defer func() {
		if closeErr := dbClient.Close(); closeErr != nil {
			log.Printf("Error closing database client: %v", closeErr)
		}
	}()

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

	// Create a comprehensive scheme for the controller manager
	// Use the client-go scheme as the base since it includes all standard Kubernetes API groups
	ctrlScheme := scheme.Scheme

	// Register all additional API groups that might be needed
	utilruntime.Must(appsv1.AddToScheme(ctrlScheme))
	utilruntime.Must(autoscalingv1.AddToScheme(ctrlScheme))
	utilruntime.Must(batchv1.AddToScheme(ctrlScheme))
	utilruntime.Must(coordinationv1.AddToScheme(ctrlScheme))
	utilruntime.Must(networkingv1.AddToScheme(ctrlScheme))
	utilruntime.Must(policyv1.AddToScheme(ctrlScheme))
	utilruntime.Must(rbacv1.AddToScheme(ctrlScheme))
	utilruntime.Must(schedulingv1.AddToScheme(ctrlScheme))
	utilruntime.Must(storagev1.AddToScheme(ctrlScheme))

	// Custom Resource Definitions
	utilruntime.Must(supacontrolv1alpha1.AddToScheme(ctrlScheme))

	mgr, err := ctrl.NewManager(k8sClient.GetConfig(), ctrl.Options{
		Scheme: ctrlScheme,
		// LeaderElection for HA deployments (configured via LEADER_ELECTION_ENABLED env var)
		LeaderElection:   cfg.LeaderElectionEnabled,
		LeaderElectionID: "supacontrol-leader-election",
		Metrics:          server.Options{BindAddress: "0"},
	})
	if err != nil {
		return fmt.Errorf("failed to create controller manager: %w", err)
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

	// Initialize handler with CR client and k8s client
	handler := api.NewHandler(authService, dbClient, crClient, k8sClient)

	// Setup routes
	api.SetupRouter(e, handler, authService, dbClient)

	// Serve static files for the UI (if available)
	// This assumes the UI is built and available at ../ui/dist
	uiPath := filepath.Join("..", "ui", "dist")
	if _, err := os.Stat(uiPath); err == nil {
		e.Static("/", uiPath)
		// Catch-all route for SPA routing: serve index.html for non-API routes
		e.GET("/*", func(c echo.Context) error {
			return c.File(filepath.Join(uiPath, "index.html"))
		})
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
