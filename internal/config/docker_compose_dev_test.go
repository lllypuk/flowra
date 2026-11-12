package config_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDockerComposeDev_ProjectNameIsFixed(t *testing.T) {
	t.Parallel()

	composeData := readYAMLMap(t, filepath.Join(repoRoot(t), "docker-compose.yml"))
	require.Equal(t, "flowra", composeData["name"])
}

func TestDockerCompose_PublishesOnLoopbackOnly(t *testing.T) {
	t.Parallel()

	for _, file := range []string{"docker-compose.yml", "docker-compose.prod.yml"} {
		services := mustMapValue(t, readYAMLMap(t, filepath.Join(repoRoot(t), file))["services"], "services")
		published := 0
		for name, raw := range services {
			service := mustMapValue(t, raw, name)
			ports, ok := service["ports"].([]any)
			if !ok {
				continue
			}
			for _, port := range ports {
				spec := fmt.Sprint(port)
				require.True(t, strings.HasPrefix(spec, "127.0.0.1:"), "%s: %s publishes %q beyond loopback", file, name, spec)
				published++
			}
		}
		require.Positive(t, published, "%s publishes no ports", file)
	}
}
