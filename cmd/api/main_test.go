package main

import (
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/lllypuk/flowra/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestShouldRunWorker(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		envValue string
		expected bool
		wantErr  bool
	}{
		{
			name:     "disabled by default",
			args:     nil,
			envValue: "",
			expected: false,
		},
		{
			name:     "enabled from env",
			args:     nil,
			envValue: "true",
			expected: true,
		},
		{
			name:     "flag enables worker",
			args:     []string{"--with-worker"},
			envValue: "",
			expected: true,
		},
		{
			name:     "flag overrides env true to false",
			args:     []string{"--with-worker=false"},
			envValue: "true",
			expected: false,
		},
		{
			name:     "flag overrides env false to true",
			args:     []string{"--with-worker=true"},
			envValue: "false",
			expected: true,
		},
		{
			name:     "invalid env value returns error",
			args:     nil,
			envValue: "invalid",
			wantErr:  true,
		},
		{
			name:     "invalid flag returns error",
			args:     []string{"--with-worker=maybe"},
			envValue: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getenv := func(key string) string {
				if key == "FLOWRA_WORKER" {
					return tt.envValue
				}

				return ""
			}

			enabled, err := shouldRunWorker(tt.args, getenv)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, enabled)
		})
	}
}

func TestWaitForWorkerShutdown(t *testing.T) {
	t.Run("nil channel returns immediately", func(t *testing.T) {
		start := time.Now()
		waitForWorkerShutdown(nil, 50*time.Millisecond, testLogger())
		assert.Less(t, time.Since(start), 25*time.Millisecond)
	})

	t.Run("closed channel returns immediately", func(t *testing.T) {
		done := make(chan struct{})
		close(done)

		start := time.Now()
		waitForWorkerShutdown(done, 50*time.Millisecond, testLogger())
		assert.Less(t, time.Since(start), 25*time.Millisecond)
	})

	t.Run("waits until timeout when worker does not stop", func(t *testing.T) {
		done := make(chan struct{})
		timeout := 25 * time.Millisecond

		start := time.Now()
		waitForWorkerShutdown(done, timeout, testLogger())
		elapsed := time.Since(start)

		assert.GreaterOrEqual(t, elapsed, timeout)
		assert.Less(t, elapsed, 200*time.Millisecond)
	})
}

func TestWorkerRuntimeError(t *testing.T) {
	t.Run("nil channel", func(t *testing.T) {
		assert.NoError(t, workerRuntimeError(nil))
	})

	t.Run("empty channel", func(t *testing.T) {
		errCh := make(chan error, 1)
		assert.NoError(t, workerRuntimeError(errCh))
	})

	t.Run("channel with error", func(t *testing.T) {
		errCh := make(chan error, 1)
		expected := errors.New("worker failed")
		errCh <- expected

		assert.ErrorIs(t, workerRuntimeError(errCh), expected)
	})
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
