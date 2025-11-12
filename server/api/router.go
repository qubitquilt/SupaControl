package api

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/qubitquilt/supacontrol/server/internal/auth"
	"github.com/qubitquilt/supacontrol/server/internal/db"
)

// SetupRouter configures all routes for the API
func SetupRouter(e *echo.Echo, handler *Handler, authService *auth.Service, dbClient *db.Client) {
	// Middleware (order matters!)
	e.Use(CorrelationIDMiddleware()) // Add request ID first
	e.Use(MetricsMiddleware())       // Record metrics for all requests
	e.Use(middleware.Logger())       // Log after correlation ID is set
	e.Use(middleware.Recover())      // Recover from panics
	e.Use(middleware.CORS())         // CORS headers

	// Public routes
	e.GET("/healthz", handler.HealthCheck)
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler())) // Prometheus metrics endpoint
	e.POST("/api/v1/auth/login", handler.Login)

	// Authenticated routes
	api := e.Group("/api/v1")
	api.Use(AuthMiddleware(authService, dbClient))

	// Auth endpoints
	api.GET("/auth/me", handler.GetAuthMe)
	api.POST("/auth/api-keys", handler.CreateAPIKey)
	api.GET("/auth/api-keys", handler.ListAPIKeys)
	api.DELETE("/auth/api-keys/:id", handler.DeleteAPIKey)

	// Instance endpoints
	api.POST("/instances", handler.CreateInstance)
	api.GET("/instances", handler.ListInstances)
	api.GET("/instances/:name", handler.GetInstance)
	api.DELETE("/instances/:name", handler.DeleteInstance)

	// Instance lifecycle endpoints
	api.POST("/instances/:name/start", handler.StartInstance)
	api.POST("/instances/:name/stop", handler.StopInstance)
	api.POST("/instances/:name/restart", handler.RestartInstance)
	api.GET("/instances/:name/logs", handler.GetLogs)
}
