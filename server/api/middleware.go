package api

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/qubitquilt/supacontrol/server/internal/auth"
	"github.com/qubitquilt/supacontrol/server/internal/db"
	"github.com/qubitquilt/supacontrol/server/internal/metrics"
)

// loggerKey is a private type for context keys to prevent collisions
type loggerKey struct{}

// AuthContext holds authentication information
type AuthContext struct {
	UserID   int64
	Username string
	Role     string
	IsAPIKey bool
}

// AuthMiddleware creates middleware for authentication
func AuthMiddleware(authService *auth.Service, dbClient *db.Client) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing authorization header")
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid authorization header format")
			}

			token := parts[1]

			// Try API key first (starts with "sk_")
			if strings.HasPrefix(token, "sk_") {
				return authenticateAPIKey(c, next, authService, dbClient, token)
			}

			// Otherwise, try JWT
			return authenticateJWT(c, next, authService, dbClient, token)
		}
	}
}

// authenticateAPIKey authenticates using an API key
func authenticateAPIKey(c echo.Context, next echo.HandlerFunc, authService *auth.Service, dbClient *db.Client, apiKey string) error {
	// Hash the API key
	keyHash, err := authService.HashAPIKey(apiKey)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to hash API key")
	}

	// Get API key from database
	apiKeyRecord, err := dbClient.GetAPIKeyByHash(keyHash)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to verify API key")
	}

	if apiKeyRecord == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid API key")
	}

	// Get user
	user, err := dbClient.GetUserByID(apiKeyRecord.UserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get user")
	}

	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "user not found")
	}

	// Update last used timestamp (async, don't wait)
	go func() {
		if err := dbClient.UpdateAPIKeyLastUsed(apiKeyRecord.ID); err != nil {
			slog.Error("Failed to update API key last used timestamp", "api_key_id", apiKeyRecord.ID, "error", err)
		}
	}()

	// Set auth context
	c.Set("auth", &AuthContext{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		IsAPIKey: true,
	})

	return next(c)
}

// authenticateJWT authenticates using a JWT token
func authenticateJWT(c echo.Context, next echo.HandlerFunc, authService *auth.Service, dbClient *db.Client, token string) error {
	claims, err := authService.ValidateJWT(token)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid JWT token")
	}

	// Verify user still exists
	user, err := dbClient.GetUserByID(claims.UserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to verify user")
	}

	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "user not found")
	}

	// Set auth context
	c.Set("auth", &AuthContext{
		UserID:   claims.UserID,
		Username: claims.Username,
		Role:     claims.Role,
		IsAPIKey: false,
	})

	return next(c)
}

// GetAuthContext retrieves the auth context from the request
func GetAuthContext(c echo.Context) *AuthContext {
	auth, ok := c.Get("auth").(*AuthContext)
	if !ok {
		return nil
	}
	return auth
}

// RequireAdmin middleware ensures the user is an admin
func RequireAdmin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authCtx := GetAuthContext(c)
		if authCtx == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
		}

		if authCtx.Role != "admin" {
			return echo.NewHTTPError(http.StatusForbidden, "admin access required")
		}

		return next(c)
	}
}

// CorrelationIDMiddleware generates a unique request ID for each request
// and adds it to the response header and logger context for tracing
func CorrelationIDMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Generate a unique request ID
			requestID := uuid.New().String()

			// Add to response header for client tracking
			c.Response().Header().Set("X-Request-ID", requestID)

			// Create a structured logger with the request ID
			logger := slog.With(
				"request_id", requestID,
				"method", c.Request().Method,
				"path", c.Request().URL.Path,
			)

			// Store logger in context for use in handlers using typed key
			ctx := context.WithValue(c.Request().Context(), loggerKey{}, logger)
			c.SetRequest(c.Request().WithContext(ctx))

			// Log the incoming request
			logger.Info("incoming request",
				"remote_addr", c.RealIP(),
			)

			return next(c)
		}
	}
}

// GetLogger retrieves the structured logger from the request context
func GetLogger(c echo.Context) *slog.Logger {
	if logger, ok := c.Request().Context().Value(loggerKey{}).(*slog.Logger); ok {
		return logger
	}
	// Fallback to default logger
	return slog.Default()
}

// MetricsMiddleware records API metrics for all requests
func MetricsMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			// Extract endpoint pattern (e.g., /api/v1/instances/:name)
			endpoint := c.Path()
			method := c.Request().Method

			// Call the next handler
			err := next(c)

			// Calculate duration
			duration := time.Since(start).Seconds()

			// Get status code
			statusCode := c.Response().Status
			if statusCode == 0 {
				statusCode = http.StatusOK
			}

			// If there was an error, extract status code
			if err != nil {
				if he, ok := err.(*echo.HTTPError); ok {
					statusCode = he.Code
				} else {
					statusCode = http.StatusInternalServerError
				}
			}

			// Record metrics
			metrics.APIRequestsTotal.WithLabelValues(
				endpoint,
				method,
				http.StatusText(statusCode),
			).Inc()

			metrics.APIRequestDuration.WithLabelValues(
				endpoint,
				method,
			).Observe(duration)

			// Log the response
			logger := GetLogger(c)
			logger.Info("request completed",
				"status", statusCode,
				"duration_ms", duration*1000,
			)

			return err
		}
	}
}

// ObserveHistogram is a helper to time operations and record to a histogram
func ObserveHistogram(histogram prometheus.Observer) func() {
	start := time.Now()
	return func() {
		histogram.Observe(time.Since(start).Seconds())
	}
}
