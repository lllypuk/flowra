package main

import (
	"testing"

	"github.com/lllypuk/flowra/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractMongoHosts(t *testing.T) {
	hosts, err := extractMongoHosts("mongodb://localhost:27017,127.0.0.1:27018/?replicaSet=rs0")
	require.NoError(t, err)
	require.Len(t, hosts, 2)
	assert.Equal(t, "localhost:27017", hosts[0])
	assert.Equal(t, "127.0.0.1:27018", hosts[1])
}

func TestIsAllowedLocalMongoHost(t *testing.T) {
	t.Run("allow localhost names", func(t *testing.T) {
		assert.True(t, isAllowedLocalMongoHost("localhost:27017"))
		assert.True(t, isAllowedLocalMongoHost("mongodb:27017"))
		assert.True(t, isAllowedLocalMongoHost("host.docker.internal:27017"))
	})

	t.Run("allow loopback IPs", func(t *testing.T) {
		assert.True(t, isAllowedLocalMongoHost("127.0.0.1:27017"))
		assert.True(t, isAllowedLocalMongoHost("[::1]:27017"))
	})

	t.Run("deny non-local hosts", func(t *testing.T) {
		assert.False(t, isAllowedLocalMongoHost("mongo.prod.example.com:27017"))
		assert.False(t, isAllowedLocalMongoHost("10.0.10.5:27017"))
	})
}

func TestGuardLocalOnly(t *testing.T) {
	base := config.DefaultConfig()
	base.MongoDB.URI = "mongodb://localhost:27017/?replicaSet=rs0"
	base.MongoDB.Database = "flowra"

	t.Run("allows local config", func(t *testing.T) {
		err := guardLocalOnly("configs/config.dev.yaml", base)
		require.NoError(t, err)
	})

	t.Run("blocks production markers", func(t *testing.T) {
		err := guardLocalOnly("configs/config.prod.yaml", base)
		require.Error(t, err)
	})

	t.Run("blocks remote mongo host", func(t *testing.T) {
		cfg := *base
		cfg.MongoDB.URI = "mongodb+srv://cluster0.mongodb.net"
		err := guardLocalOnly("", &cfg)
		require.Error(t, err)
	})
}
