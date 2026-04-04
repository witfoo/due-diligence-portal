package handler

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/witfoo/due-diligence-portal/internal/repository"
)

// HealthHandler provides health check endpoints.
type HealthHandler struct {
	db        *repository.DB
	version   string
	commit    string
	buildTime string
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(db *repository.DB, version, commit, buildTime string) *HealthHandler {
	return &HealthHandler{
		db:        db,
		version:   version,
		commit:    commit,
		buildTime: buildTime,
	}
}

// RegisterRoutes registers health check routes on the Echo instance.
func (h *HealthHandler) RegisterRoutes(e *echo.Echo) {
	e.GET("/health", h.Health)
	e.GET("/ready", h.Ready)
	e.GET("/version", h.Version)
}

// Health returns a basic health check response.
func (h *HealthHandler) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// Ready checks if the service is ready to handle requests.
func (h *HealthHandler) Ready(c echo.Context) error {
	checks := map[string]string{}

	// Check SQLite connectivity.
	if err := h.db.Ping(); err != nil {
		checks["sqlite"] = "error: " + err.Error()
		return c.JSON(http.StatusServiceUnavailable, map[string]any{
			"status":    "not ready",
			"checks":    checks,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	}
	checks["sqlite"] = "ok"

	return c.JSON(http.StatusOK, map[string]any{
		"status":    "ready",
		"checks":    checks,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// Version returns build version information.
func (h *HealthHandler) Version(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"version":    h.version,
		"commit":     h.commit,
		"build_time": h.buildTime,
	})
}
