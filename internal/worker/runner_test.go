package worker_test

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/lllypuk/flowra/internal/config"
	"github.com/lllypuk/flowra/internal/worker"
)

func TestRun_ValidateDependencies(t *testing.T) {
	t.Parallel()

	baseCfg := config.DefaultConfig()

	tests := []struct {
		name    string
		cfg     *config.Config
		db      *mongo.Database
		redis   *redis.Client
		wantErr string
	}{
		{
			name:    "nil config",
			cfg:     nil,
			db:      nil,
			redis:   nil,
			wantErr: "config is nil",
		},
		{
			name:    "nil mongodb",
			cfg:     baseCfg,
			db:      nil,
			redis:   nil,
			wantErr: "mongodb database is nil",
		},
		{
			name:    "nil redis",
			cfg:     baseCfg,
			db:      new(mongo.Database),
			redis:   nil,
			wantErr: "redis client is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := worker.Run(context.Background(), tt.cfg, tt.db, tt.redis)
			require.Error(t, err)
			assert.EqualError(t, err, tt.wantErr)
		})
	}
}
