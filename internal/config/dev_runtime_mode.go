package config

import (
	"log/slog"
	"os"
	"strings"
)

// LogDevRuntimeMode emits runtime mode diagnostics for local development workflows.
func LogDevRuntimeMode(logger *slog.Logger, cfg *Config, component string) {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("FLOWRA_DEV_MODE")))
	if mode == "" {
		mode = "unspecified"
	}

	logger.Info("runtime mode",
		slog.String("component", component),
		slog.String("dev_mode", mode),
		slog.Bool("outbox_enabled", cfg.Outbox.Enabled),
	)

	switch mode {
	case "fullstack":
		if !cfg.Outbox.Enabled {
			logger.Warn("full-stack mode expects outbox to be enabled")
		}
	case "lite":
		if component == "api" && cfg.Outbox.Enabled {
			logger.Warn("dev-lite mode with outbox enabled requires worker for fresh projections")
		}
		if component == "worker" {
			logger.Warn("worker started in dev-lite mode; dev-lite is intended for API-only runs")
		}
	case "unspecified":
		if !cfg.IsDevelopment() {
			return
		}
		if component == "api" && cfg.Outbox.Enabled {
			logger.Warn("dev mode is unspecified; use make dev (full-stack) or make dev-lite explicitly")
		}
		if component == "worker" {
			logger.Warn("dev mode is unspecified; use make dev for full-stack runtime")
		}
	default:
		logger.Warn("unknown FLOWRA_DEV_MODE value",
			slog.String("value", mode),
			slog.String("expected", "fullstack|lite"),
		)
	}
}
