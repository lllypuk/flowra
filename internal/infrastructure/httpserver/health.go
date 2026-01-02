// Package httpserver provides HTTP server infrastructure components.
package httpserver

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
)

// Health status constants - single source of truth for all health endpoints.
const (
	// StatusHealthy indicates the component is fully operational.
	StatusHealthy = "healthy"

	// StatusUnhealthy indicates the component is not operational.
	StatusUnhealthy = "unhealthy"

	// StatusDegraded indicates the component is operational but with issues.
	StatusDegraded = "degraded"

	// StatusReady indicates the service is ready to accept traffic.
	StatusReady = "ready"

	// StatusNotReady indicates the service is not ready to accept traffic.
	StatusNotReady = "not_ready"
)

// ComponentStatus represents the health status of a single component.
type ComponentStatus struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// HealthResponse represents the response for health endpoints.
type HealthResponse struct {
	Status     string            `json:"status"`
	Components []ComponentStatus `json:"components,omitempty"`
}

// HealthChecker defines the interface for checking application health.
// This interface should be implemented by the DI container or a dedicated health service.
type HealthChecker interface {
	// IsReady checks if all infrastructure components are healthy and ready to serve traffic.
	// The context should be from the current request to respect cancellation/deadlines.
	IsReady(ctx context.Context) bool

	// GetHealthStatus returns detailed health status of all components.
	// The context should be from the current request to respect cancellation/deadlines.
	GetHealthStatus(ctx context.Context) []ComponentStatus
}

// HealthEndpoints manages health check endpoint registration.
type HealthEndpoints struct {
	checker HealthChecker
}

// NewHealthEndpoints creates a new HealthEndpoints instance.
func NewHealthEndpoints(checker HealthChecker) *HealthEndpoints {
	return &HealthEndpoints{
		checker: checker,
	}
}

// Register registers all health endpoints on the Echo instance.
// Endpoints registered:
//   - GET /health - Liveness probe (always returns 200 if app is running)
//   - GET /ready - Readiness probe (returns 200 if ready, 503 if not)
//   - GET /health/details - Detailed health status of all components
func (h *HealthEndpoints) Register(e *echo.Echo) {
	e.GET("/health", h.handleHealth)
	e.GET("/ready", h.handleReady)
	e.GET("/health/details", h.handleHealthDetails)
}

// handleHealth handles the liveness probe endpoint.
// This endpoint always returns 200 OK if the application is running.
// Used by Kubernetes liveness probes.
func (h *HealthEndpoints) handleHealth(c echo.Context) error {
	return c.JSON(http.StatusOK, HealthResponse{
		Status: StatusHealthy,
	})
}

// handleReady handles the readiness probe endpoint.
// Returns 200 OK if all components are ready, 503 Service Unavailable otherwise.
// Used by Kubernetes readiness probes and load balancer health checks.
func (h *HealthEndpoints) handleReady(c echo.Context) error {
	ctx := c.Request().Context()

	if h.checker == nil || h.checker.IsReady(ctx) {
		return c.JSON(http.StatusOK, HealthResponse{
			Status:     StatusReady,
			Components: h.getComponentsIfAvailable(ctx),
		})
	}

	return c.JSON(http.StatusServiceUnavailable, HealthResponse{
		Status:     StatusNotReady,
		Components: h.getComponentsIfAvailable(ctx),
	})
}

// handleHealthDetails handles the detailed health status endpoint.
// Returns the status of each component with optional error messages.
func (h *HealthEndpoints) handleHealthDetails(c echo.Context) error {
	ctx := c.Request().Context()

	components := h.getComponentsIfAvailable(ctx)

	// Determine overall status based on component statuses
	overallStatus := StatusHealthy
	statusCode := http.StatusOK

	for _, comp := range components {
		if comp.Status == StatusUnhealthy {
			overallStatus = StatusUnhealthy
			statusCode = http.StatusServiceUnavailable
			break
		}
		if comp.Status == StatusDegraded {
			overallStatus = StatusDegraded
			// Don't break - unhealthy takes precedence
		}
	}

	return c.JSON(statusCode, HealthResponse{
		Status:     overallStatus,
		Components: components,
	})
}

// getComponentsIfAvailable returns component statuses if checker is available.
func (h *HealthEndpoints) getComponentsIfAvailable(ctx context.Context) []ComponentStatus {
	if h.checker == nil {
		return nil
	}
	return h.checker.GetHealthStatus(ctx)
}

// RegisterHealthEndpointsWithChecker registers health endpoints with a HealthChecker.
// This is a convenience function for the Router.
func (r *Router) RegisterHealthEndpointsWithChecker(checker HealthChecker) {
	endpoints := NewHealthEndpoints(checker)
	endpoints.Register(r.echo)
}

// SimpleReadinessCheck is a function type for simple readiness checks.
//
// Deprecated: Use HealthChecker interface instead for proper context handling.
type SimpleReadinessCheck func(ctx context.Context) bool

// RegisterHealthEndpointsSimple registers health endpoints with a simple readiness check function.
// This provides backwards compatibility but ensures proper context handling.
func (r *Router) RegisterHealthEndpointsSimple(check SimpleReadinessCheck) {
	r.echo.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, HealthResponse{
			Status: StatusHealthy,
		})
	})

	r.echo.GET("/ready", func(c echo.Context) error {
		ctx := c.Request().Context()

		if check == nil || check(ctx) {
			return c.JSON(http.StatusOK, HealthResponse{
				Status: StatusReady,
			})
		}

		return c.JSON(http.StatusServiceUnavailable, HealthResponse{
			Status: StatusNotReady,
		})
	})
}
