package main

import (
	"log/slog"
	"testing"

	"github.com/lllypuk/flowra/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected slog.Level
	}{
		{"debug level", "debug", slog.LevelDebug},
		{"info level", "info", slog.LevelInfo},
		{"warn level", "warn", slog.LevelWarn},
		{"error level", "error", slog.LevelError},
		{"unknown defaults to info", "unknown", slog.LevelInfo},
		{"empty defaults to info", "", slog.LevelInfo},
		{"uppercase not handled", "DEBUG", slog.LevelInfo}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseLogLevel(tt.level)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnvironment(t *testing.T) {
	tests := []struct {
		name      string
		logLevel  string
		jwtSecret string
		expected  string
	}{
		{
			name:      "development when debug",
			logLevel:  "debug",
			jwtSecret: "any-secret",
			expected:  "development",
		},
		{
			name:      "production when secure secret",
			logLevel:  "info",
			jwtSecret: "my-secure-production-secret",
			expected:  "production",
		},
		{
			name:      "unknown when info with dev secret",
			logLevel:  "info",
			jwtSecret: "dev-secret-change-in-production",
			expected:  "unknown",
		},
		{
			name:      "development takes priority over production",
			logLevel:  "debug",
			jwtSecret: "my-secure-production-secret",
			expected:  "development",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Log.Level = tt.logLevel
			cfg.Auth.JWTSecret = tt.jwtSecret
			result := getEnvironment(cfg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSetupLogger_JSONFormat(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Log.Level = "info"
	cfg.Log.Format = "json"

	logger := setupLogger(cfg)

	assert.NotNil(t, logger)
}

func TestSetupLogger_TextFormat(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Log.Level = "debug"
	cfg.Log.Format = "text"

	logger := setupLogger(cfg)

	assert.NotNil(t, logger)
}

func TestSetupLogger_DefaultFormat(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Log.Level = "warn"
	cfg.Log.Format = "" // Empty should default to json

	logger := setupLogger(cfg)

	assert.NotNil(t, logger)
}

func TestSetupLogger_AllLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error"}

	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Log.Level = level
			cfg.Log.Format = "json"

			logger := setupLogger(cfg)
			assert.NotNil(t, logger)
		})
	}
}

func TestGracefulShutdownSleepConstant(t *testing.T) {
	// Verify the constant is set to a reasonable value
	assert.Equal(t, int64(100), gracefulShutdownSleep.Milliseconds())
}
