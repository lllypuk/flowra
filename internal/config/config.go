// Package config provides configuration loading and validation for the application.
package config

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Default configuration constants.
const (
	DefaultHost            = "0.0.0.0"
	DefaultPort            = 8080
	DefaultReadTimeout     = 30 * time.Second
	DefaultWriteTimeout    = 30 * time.Second
	DefaultShutdownTimeout = 10 * time.Second

	DefaultMongoDBTimeout     = 10 * time.Second
	DefaultMongoDBMaxPoolSize = 100

	DefaultRedisPoolSize = 10

	DefaultAccessTokenTTL  = 15 * time.Minute
	DefaultRefreshTokenTTL = 7 * 24 * time.Hour // 7 days

	DefaultWSBufferSize   = 1024
	DefaultWSPingInterval = 30 * time.Second
	DefaultWSPongTimeout  = 60 * time.Second

	DefaultJWTLeeway          = 30 * time.Second
	DefaultJWTRefreshInterval = 1 * time.Hour
)

// AppMode defines the application wiring mode.
type AppMode string

// Application wiring modes.
const (
	// AppModeReal uses real implementations (MongoDB, Redis, Keycloak, etc.).
	// This is the default mode and should be used in production.
	AppModeReal AppMode = "real"

	// AppModeMock uses mock implementations for development/testing.
	// This mode is NOT allowed in production environments.
	AppModeMock AppMode = "mock"
)

// Config holds the complete application configuration.
type Config struct {
	App       AppConfig       `yaml:"app"`
	Server    ServerConfig    `yaml:"server"`
	MongoDB   MongoDBConfig   `yaml:"mongodb"`
	Redis     RedisConfig     `yaml:"redis"`
	Keycloak  KeycloakConfig  `yaml:"keycloak"`
	Auth      AuthConfig      `yaml:"auth"`
	EventBus  EventBusConfig  `yaml:"eventbus"`
	Log       LogConfig       `yaml:"log"`
	WebSocket WebSocketConfig `yaml:"websocket"`
}

// AppConfig holds application-level configuration.
type AppConfig struct {
	// Mode controls dependency wiring: "real" (default) or "mock".
	// In production, only "real" mode is allowed.
	Mode AppMode `yaml:"mode" env:"APP_MODE"`

	// Name is the application name used in logs and metrics.
	Name string `yaml:"name" env:"APP_NAME"`
}

// IsRealMode returns true if the application should use real implementations.
func (c AppConfig) IsRealMode() bool {
	return c.Mode == "" || c.Mode == AppModeReal
}

// IsMockMode returns true if the application should use mock implementations.
func (c AppConfig) IsMockMode() bool {
	return c.Mode == AppModeMock
}

// ServerConfig holds HTTP server configuration.
//
//nolint:golines // Struct tags require longer lines for readability
type ServerConfig struct {
	Host            string        `yaml:"host" env:"SERVER_HOST"`
	Port            int           `yaml:"port" env:"SERVER_PORT"`
	ReadTimeout     time.Duration `yaml:"read_timeout" env:"SERVER_READ_TIMEOUT"`
	WriteTimeout    time.Duration `yaml:"write_timeout" env:"SERVER_WRITE_TIMEOUT"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" env:"SERVER_SHUTDOWN_TIMEOUT"`
}

// Address returns the full server address (host:port).
func (c ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// MongoDBConfig holds MongoDB connection configuration.
//
//nolint:golines // Struct tags require longer lines for readability
type MongoDBConfig struct {
	URI         string        `yaml:"uri" env:"MONGODB_URI"`
	Database    string        `yaml:"database" env:"MONGODB_DATABASE"`
	Timeout     time.Duration `yaml:"timeout" env:"MONGODB_TIMEOUT"`
	MaxPoolSize uint64        `yaml:"max_pool_size" env:"MONGODB_MAX_POOL_SIZE"`
}

// RedisConfig holds Redis connection configuration.
//
//nolint:golines // Struct tags require longer lines for readability
type RedisConfig struct {
	Addr     string `yaml:"addr" env:"REDIS_ADDR"`
	Password string `yaml:"password" env:"REDIS_PASSWORD"`
	DB       int    `yaml:"db" env:"REDIS_DB"`
	PoolSize int    `yaml:"pool_size" env:"REDIS_POOL_SIZE"`
}

// KeycloakConfig holds Keycloak connection configuration.
//
//nolint:golines // Struct tags require longer lines for readability
type KeycloakConfig struct {
	URL           string    `yaml:"url" env:"KEYCLOAK_URL"`
	Realm         string    `yaml:"realm" env:"KEYCLOAK_REALM"`
	ClientID      string    `yaml:"client_id" env:"KEYCLOAK_CLIENT_ID"`
	ClientSecret  string    `yaml:"client_secret" env:"KEYCLOAK_CLIENT_SECRET"`
	AdminUsername string    `yaml:"admin_username" env:"KEYCLOAK_ADMIN_USERNAME"`
	AdminPassword string    `yaml:"admin_password" env:"KEYCLOAK_ADMIN_PASSWORD"`
	JWT           JWTConfig `yaml:"jwt"`
}

// JWTConfig holds JWT validation configuration.
//
//nolint:golines // Struct tags require longer lines for readability
type JWTConfig struct {
	Leeway          time.Duration `yaml:"leeway" env:"KEYCLOAK_JWT_LEEWAY"`
	RefreshInterval time.Duration `yaml:"refresh_interval" env:"KEYCLOAK_JWT_REFRESH_INTERVAL"`
}

// AuthConfig holds authentication configuration.
//
//nolint:golines // Struct tags require longer lines for readability
type AuthConfig struct {
	JWTSecret       string        `yaml:"jwt_secret" env:"AUTH_JWT_SECRET"`
	AccessTokenTTL  time.Duration `yaml:"access_token_ttl" env:"AUTH_ACCESS_TOKEN_TTL"`
	RefreshTokenTTL time.Duration `yaml:"refresh_token_ttl" env:"AUTH_REFRESH_TOKEN_TTL"`
}

// EventBusConfig holds event bus configuration.
//
//nolint:golines // Struct tags require longer lines for readability
type EventBusConfig struct {
	Type               string `yaml:"type" env:"EVENTBUS_TYPE"` // redis | inmemory
	RedisChannelPrefix string `yaml:"redis_channel_prefix" env:"EVENTBUS_REDIS_CHANNEL_PREFIX"`
}

// LogConfig holds logging configuration.
//
//nolint:golines // Struct tags require longer lines for readability
type LogConfig struct {
	Level  string `yaml:"level" env:"LOG_LEVEL"`   // debug | info | warn | error
	Format string `yaml:"format" env:"LOG_FORMAT"` // json | text
}

// WebSocketConfig holds WebSocket server configuration.
//
//nolint:golines // Struct tags require longer lines for readability
type WebSocketConfig struct {
	ReadBufferSize  int           `yaml:"read_buffer_size" env:"WS_READ_BUFFER_SIZE"`
	WriteBufferSize int           `yaml:"write_buffer_size" env:"WS_WRITE_BUFFER_SIZE"`
	PingInterval    time.Duration `yaml:"ping_interval" env:"WS_PING_INTERVAL"`
	PongTimeout     time.Duration `yaml:"pong_timeout" env:"WS_PONG_TIMEOUT"`
}

// Configuration errors.
var (
	ErrConfigNotFound      = errors.New("configuration file not found")
	ErrConfigInvalid       = errors.New("invalid configuration")
	ErrMissingRequired     = errors.New("missing required configuration")
	ErrInvalidDuration     = errors.New("invalid duration format")
	ErrInvalidLogLevel     = errors.New("invalid log level: must be debug, info, warn, or error")
	ErrInvalidLogFormat    = errors.New("invalid log format: must be json or text")
	ErrInvalidEventBusType = errors.New("invalid event bus type: must be redis or inmemory")
	ErrInvalidAppMode      = errors.New("invalid app mode: must be real or mock")
	ErrMockModeInProd      = errors.New("mock mode is not allowed in production")
)

// DefaultConfig returns a Config with sensible default values.
func DefaultConfig() *Config {
	return &Config{
		App: AppConfig{
			Mode: AppModeReal,
			Name: "flowra",
		},
		Server: ServerConfig{
			Host:            DefaultHost,
			Port:            DefaultPort,
			ReadTimeout:     DefaultReadTimeout,
			WriteTimeout:    DefaultWriteTimeout,
			ShutdownTimeout: DefaultShutdownTimeout,
		},
		MongoDB: MongoDBConfig{
			URI:         "mongodb://localhost:27017",
			Database:    "flowra",
			Timeout:     DefaultMongoDBTimeout,
			MaxPoolSize: DefaultMongoDBMaxPoolSize,
		},
		Redis: RedisConfig{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
			PoolSize: DefaultRedisPoolSize,
		},
		Keycloak: KeycloakConfig{
			URL:      "http://localhost:8090",
			Realm:    "flowra",
			ClientID: "flowra-backend",
			JWT: JWTConfig{
				Leeway:          DefaultJWTLeeway,
				RefreshInterval: DefaultJWTRefreshInterval,
			},
		},
		Auth: AuthConfig{
			JWTSecret:       "dev-secret-change-in-production",
			AccessTokenTTL:  DefaultAccessTokenTTL,
			RefreshTokenTTL: DefaultRefreshTokenTTL,
		},
		EventBus: EventBusConfig{
			Type:               "redis",
			RedisChannelPrefix: "events:",
		},
		Log: LogConfig{
			Level:  "info",
			Format: "json",
		},
		WebSocket: WebSocketConfig{
			ReadBufferSize:  DefaultWSBufferSize,
			WriteBufferSize: DefaultWSBufferSize,
			PingInterval:    DefaultWSPingInterval,
			PongTimeout:     DefaultWSPongTimeout,
		},
	}
}

// Validate validates the configuration and returns an error if invalid.
func (c *Config) Validate() error {
	var errs []error

	errs = c.validateApp(errs)
	errs = c.validateServer(errs)
	errs = c.validateMongoDB(errs)
	errs = c.validateRedis(errs)
	errs = c.validateAuth(errs)
	errs = c.validateLog(errs)
	errs = c.validateEventBus(errs)
	errs = c.validateWebSocket(errs)

	if len(errs) > 0 {
		return fmt.Errorf("%w: %w", ErrConfigInvalid, errors.Join(errs...))
	}

	return nil
}

// validateApp validates application configuration.
func (c *Config) validateApp(errs []error) []error {
	if c.App.Mode != "" && c.App.Mode != AppModeReal && c.App.Mode != AppModeMock {
		errs = append(errs, fmt.Errorf("%w: got %q", ErrInvalidAppMode, c.App.Mode))
	}
	if c.App.IsMockMode() && c.IsProduction() {
		errs = append(errs, ErrMockModeInProd)
	}
	return errs
}

// validateServer validates server configuration.
func (c *Config) validateServer(errs []error) []error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		errs = append(errs, fmt.Errorf("server.port must be between 1 and 65535, got %d", c.Server.Port))
	}
	if c.Server.ReadTimeout <= 0 {
		errs = append(errs, errors.New("server.read_timeout must be positive"))
	}
	if c.Server.WriteTimeout <= 0 {
		errs = append(errs, errors.New("server.write_timeout must be positive"))
	}
	return errs
}

// validateMongoDB validates MongoDB configuration.
func (c *Config) validateMongoDB(errs []error) []error {
	if c.MongoDB.URI == "" {
		errs = append(errs, errors.New("mongodb.uri is required"))
	}
	if c.MongoDB.Database == "" {
		errs = append(errs, errors.New("mongodb.database is required"))
	}
	return errs
}

// validateRedis validates Redis configuration.
func (c *Config) validateRedis(errs []error) []error {
	if c.Redis.Addr == "" {
		errs = append(errs, errors.New("redis.addr is required"))
	}
	return errs
}

// validateAuth validates authentication configuration.
func (c *Config) validateAuth(errs []error) []error {
	if c.Auth.JWTSecret == "" {
		errs = append(errs, errors.New("auth.jwt_secret is required"))
	}
	// Note: "dev-secret-change-in-production" is allowed for development
	// In production, deployment validation should catch insecure secrets
	if c.Auth.AccessTokenTTL <= 0 {
		errs = append(errs, errors.New("auth.access_token_ttl must be positive"))
	}
	if c.Auth.RefreshTokenTTL <= 0 {
		errs = append(errs, errors.New("auth.refresh_token_ttl must be positive"))
	}
	return errs
}

// validateLog validates logging configuration.
func (c *Config) validateLog(errs []error) []error {
	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLogLevels[strings.ToLower(c.Log.Level)] {
		errs = append(errs, ErrInvalidLogLevel)
	}
	validLogFormats := map[string]bool{"json": true, "text": true}
	if !validLogFormats[strings.ToLower(c.Log.Format)] {
		errs = append(errs, ErrInvalidLogFormat)
	}
	return errs
}

// validateEventBus validates event bus configuration.
func (c *Config) validateEventBus(errs []error) []error {
	validEventBusTypes := map[string]bool{"redis": true, "inmemory": true}
	if !validEventBusTypes[strings.ToLower(c.EventBus.Type)] {
		errs = append(errs, ErrInvalidEventBusType)
	}
	return errs
}

// validateWebSocket validates WebSocket configuration.
func (c *Config) validateWebSocket(errs []error) []error {
	if c.WebSocket.ReadBufferSize <= 0 {
		errs = append(errs, errors.New("websocket.read_buffer_size must be positive"))
	}
	if c.WebSocket.WriteBufferSize <= 0 {
		errs = append(errs, errors.New("websocket.write_buffer_size must be positive"))
	}
	if c.WebSocket.PingInterval <= 0 {
		errs = append(errs, errors.New("websocket.ping_interval must be positive"))
	}
	if c.WebSocket.PongTimeout <= 0 {
		errs = append(errs, errors.New("websocket.pong_timeout must be positive"))
	}
	return errs
}

// Load loads configuration from the default config file and environment variables.
func Load() (*Config, error) {
	return LoadFromPath("")
}

// LoadFromPath loads configuration from a specific file path.
// If path is empty, it tries to find the config file in standard locations.
func LoadFromPath(path string) (*Config, error) {
	loader := NewLoader()
	return loader.Load(path)
}

// Loader handles configuration loading from files and environment variables.
type Loader struct {
	configPaths []string
}

// NewLoader creates a new configuration loader.
func NewLoader() *Loader {
	return &Loader{
		configPaths: []string{
			"configs/config.yaml",
			"config.yaml",
			"/etc/flowra/config.yaml",
		},
	}
}

// WithConfigPaths sets custom config paths to search.
func (l *Loader) WithConfigPaths(paths []string) *Loader {
	l.configPaths = paths
	return l
}

// Load loads configuration from file and environment variables.
func (l *Loader) Load(path string) (*Config, error) {
	// Start with default config
	cfg := DefaultConfig()

	// Determine config file path
	configPath := path
	if configPath == "" {
		// Check CONFIG_PATH environment variable first
		if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
			configPath = envPath
		} else {
			// Search in standard locations
			for _, p := range l.configPaths {
				if _, err := os.Stat(p); err == nil {
					configPath = p
					break
				}
			}
		}
	}

	// Load from file if found
	if configPath != "" {
		if err := l.loadFromFile(cfg, configPath); err != nil {
			// Only return error if path was explicitly specified
			if path != "" || os.Getenv("CONFIG_PATH") != "" {
				return nil, fmt.Errorf("failed to load config from %s: %w", configPath, err)
			}
			// Otherwise, continue with defaults + env vars
		}
	}

	// Override with environment variables
	if err := l.loadFromEnv(cfg); err != nil {
		return nil, fmt.Errorf("failed to load config from environment: %w", err)
	}

	// Validate the final configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// loadFromFile loads configuration from a YAML file.
func (l *Loader) loadFromFile(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrConfigNotFound, path)
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if unmarshalErr := yaml.Unmarshal(data, cfg); unmarshalErr != nil {
		return fmt.Errorf("failed to parse config file: %w", unmarshalErr)
	}

	return nil
}

// loadFromEnv loads configuration from environment variables.
func (l *Loader) loadFromEnv(cfg *Config) error {
	return l.loadEnvToStruct(reflect.ValueOf(cfg).Elem())
}

// loadEnvToStruct recursively loads environment variables into a struct.
func (l *Loader) loadEnvToStruct(v reflect.Value) error {
	t := v.Type()

	for i := range v.NumField() {
		field := v.Field(i)
		fieldType := t.Field(i)

		// Handle embedded structs
		if field.Kind() == reflect.Struct {
			if err := l.loadEnvToStruct(field); err != nil {
				return err
			}
			continue
		}

		// Get env tag
		envTag := fieldType.Tag.Get("env")
		if envTag == "" {
			continue
		}

		// Get environment variable value
		envValue := os.Getenv(envTag)
		if envValue == "" {
			continue
		}

		// Set field value based on type
		if err := l.setFieldFromEnv(field, envValue); err != nil {
			return fmt.Errorf("failed to set %s from env %s: %w", fieldType.Name, envTag, err)
		}
	}

	return nil
}

// setFieldFromEnv sets a struct field value from an environment variable string.
//
//nolint:exhaustive // We only support a subset of reflect.Kind for config values
func (l *Loader) setFieldFromEnv(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Check if it's a time.Duration
		if field.Type() == reflect.TypeFor[time.Duration]() {
			d, err := time.ParseDuration(value)
			if err != nil {
				return fmt.Errorf("%w: %s", ErrInvalidDuration, value)
			}
			field.SetInt(int64(d))
		} else {
			i, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid integer value: %s", value)
			}
			field.SetInt(i)
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid unsigned integer value: %s", value)
		}
		field.SetUint(u)

	case reflect.Bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean value: %s", value)
		}
		field.SetBool(b)

	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid float value: %s", value)
		}
		field.SetFloat(f)

	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}

	return nil
}

// IsDevelopment returns true if the log level indicates a development environment.
func (c *Config) IsDevelopment() bool {
	return strings.ToLower(c.Log.Level) == "debug"
}

// IsProduction returns true if authentication appears configured for production.
func (c *Config) IsProduction() bool {
	return c.Auth.JWTSecret != "dev-secret-change-in-production" &&
		c.Auth.JWTSecret != ""
}
