package api

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/qubitquilt/supacontrol/server/internal/auth"
	"github.com/qubitquilt/supacontrol/server/internal/db"
)

// SetupRouter configures all routes for the API
func SetupRouter(e *echo.Echo, handler *Handler, authService *auth.Service, dbClient *db.Client) {
	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Public routes
	e.GET("/healthz", handler.HealthCheck)
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
