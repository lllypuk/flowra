package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/connstring"

	"github.com/lllypuk/flowra/internal/config"
	"github.com/lllypuk/flowra/internal/infrastructure/mongodb"
)

const (
	connectTimeout = 20 * time.Second
	resetTimeout   = 60 * time.Second
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if err := run(logger); err != nil {
		logger.Error("reset failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	resetCollections := []string{
		mongodb.CollectionEvents,
		mongodb.CollectionChatReadModel,
		mongodb.CollectionTaskReadModel,
		mongodb.CollectionOutbox,
		mongodb.CollectionRepairQueue,
	}

	configPath := flag.String("config", "", "path to config file (optional)")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err = guardLocalOnly(*configPath, cfg); err != nil {
		return fmt.Errorf("reset is blocked by safety guard: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), resetTimeout)
	defer cancel()

	connectCtx, connectCancel := context.WithTimeout(context.Background(), connectTimeout)

	client, err := mongo.Connect(options.Client().ApplyURI(cfg.MongoDB.URI))
	if err != nil {
		connectCancel()
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	defer func() {
		if disconnectErr := client.Disconnect(context.Background()); disconnectErr != nil {
			logger.Warn("failed to disconnect MongoDB client", slog.String("error", disconnectErr.Error()))
		}
	}()

	err = client.Ping(connectCtx, nil)
	connectCancel()
	if err != nil {
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db := client.Database(cfg.MongoDB.Database)

	for _, collectionName := range resetCollections {
		err = db.Collection(collectionName).Drop(ctx)
		if err != nil && !isNamespaceNotFound(err) {
			return fmt.Errorf("failed to drop %s: %w", collectionName, err)
		}

		logger.Info("collection dropped", slog.String("collection", collectionName))
	}

	if err = mongodb.CreateAllIndexes(ctx, db); err != nil {
		return fmt.Errorf("failed to recreate MongoDB indexes: %w", err)
	}

	logger.Info("data reset completed",
		slog.String("database", cfg.MongoDB.Database),
		slog.Int("collections_reset", len(resetCollections)),
	)

	return nil
}

func loadConfig(configPath string) (*config.Config, error) {
	if strings.TrimSpace(configPath) == "" {
		return config.Load()
	}
	return config.LoadFromPath(configPath)
}

func guardLocalOnly(configPath string, cfg *config.Config) error {
	if cfg == nil {
		return errors.New("config is nil")
	}

	combined := strings.ToLower(strings.Join([]string{
		configPath,
		os.Getenv("CONFIG_PATH"),
		os.Getenv("APP_ENV"),
		os.Getenv("ENV"),
		os.Getenv("GO_ENV"),
	}, " "))

	if strings.Contains(combined, "prod") || strings.Contains(combined, "production") {
		return errors.New("production-like environment detected in config/env markers")
	}

	hosts, err := extractMongoHosts(cfg.MongoDB.URI)
	if err != nil {
		return fmt.Errorf("failed to parse mongodb URI: %w", err)
	}

	if len(hosts) == 0 {
		return errors.New("mongodb URI does not contain hosts")
	}

	for _, host := range hosts {
		if !isAllowedLocalMongoHost(host) {
			return fmt.Errorf("host %q is not allowed; only local/dev/test MongoDB is supported", host)
		}
	}

	return nil
}

func extractMongoHosts(uri string) ([]string, error) {
	conn, err := connstring.ParseAndValidate(uri)
	if err != nil {
		return nil, err
	}

	hosts := make([]string, 0, len(conn.Hosts))
	for _, host := range conn.Hosts {
		if trimmed := strings.TrimSpace(host); trimmed != "" {
			hosts = append(hosts, trimmed)
		}
	}

	return hosts, nil
}

func isAllowedLocalMongoHost(hostPort string) bool {
	host := strings.TrimSpace(hostPort)
	if host == "" {
		return false
	}

	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}

	host = strings.Trim(host, "[]")
	if host == "" {
		return false
	}

	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}

	switch strings.ToLower(host) {
	case "localhost", "mongodb", "host.docker.internal":
		return true
	default:
		return false
	}
}

func isNamespaceNotFound(err error) bool {
	if err == nil {
		return false
	}

	var cmdErr mongo.CommandError
	if errors.As(err, &cmdErr) && cmdErr.Code == 26 {
		return true
	}

	return strings.Contains(strings.ToLower(err.Error()), "ns not found")
}
