// Package appcore provides core application interfaces and shared utilities.
package appcore

import (
	"context"
	"time"
)

// HealthChecker checks the health of a specific component or subsystem.
type HealthChecker interface {
	// Check performs health check and returns status.
	Check(ctx context.Context) HealthStatus

	// Name returns the name of this health checker.
	Name() string
}

// HealthStatus represents the health status of a component.
type HealthStatus struct {
	Healthy   bool           `json:"healthy"`
	Message   string         `json:"message,omitempty"`
	Details   map[string]any `json:"details,omitempty"`
	CheckedAt time.Time      `json:"checked_at"`
}
