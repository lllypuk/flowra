package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lllypuk/flowra/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()

	assert.NotNil(t, cfg)

	// Server defaults
	assert.Equal(t, config.DefaultHost, cfg.Server.Host)
	assert.Equal(t, config.DefaultPort, cfg.Server.Port)
	assert.Equal(t, config.DefaultReadTimeout, cfg.Server.ReadTimeout)
	assert.Equal(t, config.DefaultWriteTimeout, cfg.Server.WriteTimeout)
	assert.Equal(t, config.DefaultShutdownTimeout, cfg.Server.ShutdownTimeout)

	// MongoDB defaults
	assert.Equal(t, "mongodb://localhost:27017", cfg.MongoDB.URI)
	assert.Equal(t, "flowra", cfg.MongoDB.Database)
	assert.Equal(t, config.DefaultMongoDBTimeout, cfg.MongoDB.Timeout)
	assert.Equal(t, uint64(config.DefaultMongoDBMaxPoolSize), cfg.MongoDB.MaxPoolSize)

	// Redis defaults
	assert.Equal(t, "localhost:6379", cfg.Redis.Addr)
	assert.Empty(t, cfg.Redis.Password)
	assert.Equal(t, 0, cfg.Redis.DB)
	assert.Equal(t, config.DefaultRedisPoolSize, cfg.Redis.PoolSize)

	// Auth defaults
	assert.Equal(t, "dev-secret-change-in-production", cfg.Auth.JWTSecret)
	assert.Equal(t, config.DefaultAccessTokenTTL, cfg.Auth.AccessTokenTTL)
	assert.Equal(t, config.DefaultRefreshTokenTTL, cfg.Auth.RefreshTokenTTL)

	// EventBus defaults
	assert.Equal(t, "redis", cfg.EventBus.Type)
	assert.Equal(t, "events:", cfg.EventBus.RedisChannelPrefix)

	// Log defaults
	assert.Equal(t, "info", cfg.Log.Level)
	assert.Equal(t, "json", cfg.Log.Format)

	// WebSocket defaults
	assert.Equal(t, config.DefaultWSBufferSize, cfg.WebSocket.ReadBufferSize)
	assert.Equal(t, config.DefaultWSBufferSize, cfg.WebSocket.WriteBufferSize)
	assert.Equal(t, config.DefaultWSPingInterval, cfg.WebSocket.PingInterval)
	assert.Equal(t, config.DefaultWSPongTimeout, cfg.WebSocket.PongTimeout)
}

func TestServerConfig_Address(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		port     int
		expected string
	}{
		{
			name:     "default address",
			host:     "0.0.0.0",
			port:     8080,
			expected: "0.0.0.0:8080",
		},
		{
			name:     "localhost",
			host:     "localhost",
			port:     3000,
			expected: "localhost:3000",
		},
		{
			name:     "custom host and port",
			host:     "192.168.1.100",
			port:     9090,
			expected: "192.168.1.100:9090",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.ServerConfig{
				Host: tt.host,
				Port: tt.port,
			}
			assert.Equal(t, tt.expected, cfg.Address())
		})
	}
}

func TestConfig_Validate_Valid(t *testing.T) {
	cfg := config.DefaultConfig()
	err := cfg.Validate()
	assert.NoError(t, err)
}

func TestConfig_Validate_InvalidPort(t *testing.T) {
	tests := []struct {
		name string
		port int
	}{
		{"negative port", -1},
		{"zero port", 0},
		{"port too high", 65536},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Server.Port = tt.port
			err := cfg.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "server.port")
		})
	}
}

func TestConfig_Validate_InvalidTimeouts(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*config.Config)
		errMsg string
	}{
		{
			name: "negative read timeout",
			modify: func(c *config.Config) {
				c.Server.ReadTimeout = -1 * time.Second
			},
			errMsg: "server.read_timeout must be positive",
		},
		{
			name: "zero write timeout",
			modify: func(c *config.Config) {
				c.Server.WriteTimeout = 0
			},
			errMsg: "server.write_timeout must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			tt.modify(cfg)
			err := cfg.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestConfig_Validate_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*config.Config)
		errMsg string
	}{
		{
			name: "missing mongodb uri",
			modify: func(c *config.Config) {
				c.MongoDB.URI = ""
			},
			errMsg: "mongodb.uri is required",
		},
		{
			name: "missing mongodb database",
			modify: func(c *config.Config) {
				c.MongoDB.Database = ""
			},
			errMsg: "mongodb.database is required",
		},
		{
			name: "missing redis addr",
			modify: func(c *config.Config) {
				c.Redis.Addr = ""
			},
			errMsg: "redis.addr is required",
		},
		{
			name: "missing jwt secret",
			modify: func(c *config.Config) {
				c.Auth.JWTSecret = ""
			},
			errMsg: "auth.jwt_secret is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			tt.modify(cfg)
			err := cfg.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestConfig_Validate_InvalidLogLevel(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Log.Level = "invalid"
	err := cfg.Validate()
	require.Error(t, err)
	assert.ErrorIs(t, err, config.ErrConfigInvalid)
}

func TestConfig_Validate_InvalidLogFormat(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Log.Format = "xml"
	err := cfg.Validate()
	require.Error(t, err)
	assert.ErrorIs(t, err, config.ErrConfigInvalid)
}

func TestConfig_Validate_InvalidEventBusType(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.EventBus.Type = "kafka"
	err := cfg.Validate()
	require.Error(t, err)
	assert.ErrorIs(t, err, config.ErrConfigInvalid)
}

func TestConfig_Validate_ValidLogLevels(t *testing.T) {
	validLevels := []string{"debug", "info", "warn", "error", "DEBUG", "INFO", "WARN", "ERROR"}

	for _, level := range validLevels {
		t.Run(level, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Log.Level = level
			err := cfg.Validate()
			assert.NoError(t, err)
		})
	}
}

func TestConfig_Validate_ValidEventBusTypes(t *testing.T) {
	validTypes := []string{"redis", "inmemory", "REDIS", "INMEMORY"}

	for _, busType := range validTypes {
		t.Run(busType, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.EventBus.Type = busType
			err := cfg.Validate()
			assert.NoError(t, err)
		})
	}
}

func TestConfig_IsDevelopment(t *testing.T) {
	tests := []struct {
		name     string
		logLevel string
		expected bool
	}{
		{"debug level", "debug", true},
		{"info level", "info", false},
		{"warn level", "warn", false},
		{"error level", "error", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Log.Level = tt.logLevel
			assert.Equal(t, tt.expected, cfg.IsDevelopment())
		})
	}
}

func TestConfig_IsProduction(t *testing.T) {
	tests := []struct {
		name      string
		jwtSecret string
		expected  bool
	}{
		{"dev secret", "dev-secret-change-in-production", false},
		{"empty secret", "", false},
		{"production secret", "my-secure-production-secret", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Auth.JWTSecret = tt.jwtSecret
			assert.Equal(t, tt.expected, cfg.IsProduction())
		})
	}
}

func TestLoadFromPath_ValidYAML(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
server:
  host: "127.0.0.1"
  port: 9090
  read_timeout: 45s
  write_timeout: 45s
  shutdown_timeout: 15s

mongodb:
  uri: "mongodb://testhost:27017"
  database: "testdb"
  timeout: 5s
  max_pool_size: 50

redis:
  addr: "redis:6379"
  password: "testpass"
  db: 1
  pool_size: 20

auth:
  jwt_secret: "test-secret-key"
  access_token_ttl: 30m
  refresh_token_ttl: 24h

log:
  level: "debug"
  format: "text"

eventbus:
  type: "inmemory"
  redis_channel_prefix: "test:"

websocket:
  read_buffer_size: 2048
  write_buffer_size: 2048
  ping_interval: 60s
  pong_timeout: 120s
`
	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	cfg, err := config.LoadFromPath(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify loaded values
	assert.Equal(t, "127.0.0.1", cfg.Server.Host)
	assert.Equal(t, 9090, cfg.Server.Port)
	assert.Equal(t, 45*time.Second, cfg.Server.ReadTimeout)

	assert.Equal(t, "mongodb://testhost:27017", cfg.MongoDB.URI)
	assert.Equal(t, "testdb", cfg.MongoDB.Database)
	assert.Equal(t, uint64(50), cfg.MongoDB.MaxPoolSize)

	assert.Equal(t, "redis:6379", cfg.Redis.Addr)
	assert.Equal(t, "testpass", cfg.Redis.Password)
	assert.Equal(t, 1, cfg.Redis.DB)
	assert.Equal(t, 20, cfg.Redis.PoolSize)

	assert.Equal(t, "test-secret-key", cfg.Auth.JWTSecret)
	assert.Equal(t, 30*time.Minute, cfg.Auth.AccessTokenTTL)
	assert.Equal(t, 24*time.Hour, cfg.Auth.RefreshTokenTTL)

	assert.Equal(t, "debug", cfg.Log.Level)
	assert.Equal(t, "text", cfg.Log.Format)

	assert.Equal(t, "inmemory", cfg.EventBus.Type)
	assert.Equal(t, "test:", cfg.EventBus.RedisChannelPrefix)

	assert.Equal(t, 2048, cfg.WebSocket.ReadBufferSize)
	assert.Equal(t, 60*time.Second, cfg.WebSocket.PingInterval)
}

func TestLoadFromPath_NonExistent(t *testing.T) {
	cfg, err := config.LoadFromPath("/non/existent/path/config.yaml")
	require.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "configuration file not found")
}

func TestLoadFromPath_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	invalidContent := `
server:
  host: "localhost"
  port: this-is-not-a-number
`
	err := os.WriteFile(configPath, []byte(invalidContent), 0o644)
	require.NoError(t, err)

	cfg, err := config.LoadFromPath(configPath)
	require.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "failed to parse config file")
}

func TestLoader_LoadFromEnv(t *testing.T) {
	// Set test environment variables using t.Setenv (auto-cleanup)
	t.Setenv("SERVER_HOST", "env-host")
	t.Setenv("SERVER_PORT", "3333")
	t.Setenv("MONGODB_URI", "mongodb://env-mongo:27017")
	t.Setenv("REDIS_ADDR", "env-redis:6379")
	t.Setenv("AUTH_JWT_SECRET", "env-jwt-secret")
	t.Setenv("LOG_LEVEL", "warn")

	// Create a minimal config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	minimalConfig := `
server:
  host: "file-host"
  port: 8080
`
	err := os.WriteFile(configPath, []byte(minimalConfig), 0o644)
	require.NoError(t, err)

	cfg, err := config.LoadFromPath(configPath)
	require.NoError(t, err)

	// Env vars should override file values
	assert.Equal(t, "env-host", cfg.Server.Host)
	assert.Equal(t, 3333, cfg.Server.Port)
	assert.Equal(t, "mongodb://env-mongo:27017", cfg.MongoDB.URI)
	assert.Equal(t, "env-redis:6379", cfg.Redis.Addr)
	assert.Equal(t, "env-jwt-secret", cfg.Auth.JWTSecret)
	assert.Equal(t, "warn", cfg.Log.Level)
}

func TestLoader_LoadFromEnv_Duration(t *testing.T) {
	t.Setenv("SERVER_READ_TIMEOUT", "2m30s")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, 2*time.Minute+30*time.Second, cfg.Server.ReadTimeout)
}

func TestLoader_LoadFromEnv_InvalidDuration(t *testing.T) {
	t.Setenv("SERVER_READ_TIMEOUT", "not-a-duration")

	cfg, err := config.Load()
	require.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "invalid duration")
}

func TestLoader_ConfigPathEnvVar(t *testing.T) {
	// Create a config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "custom-config.yaml")
	configContent := `
server:
  host: "config-path-host"
  port: 7777
mongodb:
  uri: "mongodb://localhost:27017"
  database: "testdb"
redis:
  addr: "localhost:6379"
auth:
  jwt_secret: "test-secret"
log:
  level: "info"
  format: "json"
eventbus:
  type: "redis"
websocket:
  read_buffer_size: 1024
  write_buffer_size: 1024
  ping_interval: 30s
  pong_timeout: 60s
`
	err := os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	t.Setenv("CONFIG_PATH", configPath)

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "config-path-host", cfg.Server.Host)
	assert.Equal(t, 7777, cfg.Server.Port)
}

func TestLoader_WithConfigPaths(t *testing.T) {
	loader := config.NewLoader()
	customPaths := []string{"/custom/path1.yaml", "/custom/path2.yaml"}
	loader.WithConfigPaths(customPaths)

	// We can't directly check the paths since they are private,
	// but we can verify the method doesn't panic
	assert.NotNil(t, loader)
}

func TestNewLoader(t *testing.T) {
	loader := config.NewLoader()
	assert.NotNil(t, loader)
}

func TestConfig_Validate_WebSocketConfig(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*config.Config)
		errMsg string
	}{
		{
			name: "zero read buffer size",
			modify: func(c *config.Config) {
				c.WebSocket.ReadBufferSize = 0
			},
			errMsg: "websocket.read_buffer_size must be positive",
		},
		{
			name: "negative write buffer size",
			modify: func(c *config.Config) {
				c.WebSocket.WriteBufferSize = -1
			},
			errMsg: "websocket.write_buffer_size must be positive",
		},
		{
			name: "zero ping interval",
			modify: func(c *config.Config) {
				c.WebSocket.PingInterval = 0
			},
			errMsg: "websocket.ping_interval must be positive",
		},
		{
			name: "zero pong timeout",
			modify: func(c *config.Config) {
				c.WebSocket.PongTimeout = 0
			},
			errMsg: "websocket.pong_timeout must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			tt.modify(cfg)
			err := cfg.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestConfig_Validate_AuthTokenTTL(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*config.Config)
		errMsg string
	}{
		{
			name: "zero access token TTL",
			modify: func(c *config.Config) {
				c.Auth.AccessTokenTTL = 0
			},
			errMsg: "auth.access_token_ttl must be positive",
		},
		{
			name: "negative refresh token TTL",
			modify: func(c *config.Config) {
				c.Auth.RefreshTokenTTL = -1 * time.Hour
			},
			errMsg: "auth.refresh_token_ttl must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			tt.modify(cfg)
			err := cfg.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}
