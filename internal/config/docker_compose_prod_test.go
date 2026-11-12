package config_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestDockerComposeProd_RequiredServicesAndEnv(t *testing.T) {
	t.Parallel()

	composePath := filepath.Join(repoRoot(t), "docker-compose.prod.yml")
	composeData := readYAMLMap(t, composePath)

	servicesRaw, ok := composeData["services"]
	require.True(t, ok, "services section is required")

	services, ok := servicesRaw.(map[string]any)
	require.True(t, ok, "services must be a map")

	requiredServices := []string{"mongodb", "mongo-init", "redis", "app"}
	for _, name := range requiredServices {
		require.Contains(t, services, name)
	}
	for name := range services {
		require.NotContains(t, strings.ToLower(name), "keycloak")
	}

	appRaw, ok := services["app"]
	require.True(t, ok, "app service is required")

	app, ok := appRaw.(map[string]any)
	require.True(t, ok, "app service must be a map")

	environmentRaw, ok := app["environment"]
	require.True(t, ok, "app environment is required")

	environment, ok := environmentRaw.(map[string]any)
	require.True(t, ok, "app environment must be a map")

	requiredEnv := []string{
		"MONGODB_URI",
		"MONGODB_DATABASE",
		"REDIS_ADDR",
		"REDIS_PASSWORD",
		"AUTH_OWNER_TOKEN_SHA256",
	}
	for _, envKey := range requiredEnv {
		require.Contains(t, environment, envKey)
	}

	mongoURI, ok := environment["MONGODB_URI"].(string)
	require.True(t, ok, "MONGODB_URI must be a string")
	require.Contains(t, mongoURI, "@localhost:27017/")
	require.Contains(t, mongoURI, "directConnection=true")
	require.Contains(t, mongoURI, "authSource=admin")
	require.Contains(t, mongoURI, "${MONGODB_ROOT_USERNAME:?MONGODB_ROOT_USERNAME is required}")
	require.Contains(t, mongoURI, "${MONGODB_ROOT_PASSWORD:?MONGODB_ROOT_PASSWORD is required}")

	redisAddr, ok := environment["REDIS_ADDR"].(string)
	require.True(t, ok, "REDIS_ADDR must be a string")
	require.Equal(t, "localhost:6379", redisAddr)

	require.Equal(t, "${REDIS_PASSWORD:?REDIS_PASSWORD is required}", environment["REDIS_PASSWORD"])

	workerMode, ok := environment["FLOWRA_WORKER"].(string)
	require.True(t, ok, "FLOWRA_WORKER must be a string")
	require.Contains(t, workerMode, "true")

	require.Equal(
		t,
		"${AUTH_OWNER_TOKEN_SHA256:?AUTH_OWNER_TOKEN_SHA256 is required}",
		environment["AUTH_OWNER_TOKEN_SHA256"],
	)

	require.Equal(t, "host", app["network_mode"], "token auth needs a loopback listener on the host")
	_, hasPorts := app["ports"]
	require.False(t, hasPorts, "ports do not apply on the host network")
	_, hasNetworks := app["networks"]
	require.False(t, hasNetworks, "networks cannot be combined with the host network")
}

func TestEnvExample_RequiresExplicitSecrets(t *testing.T) {
	t.Parallel()

	envPath := filepath.Join(repoRoot(t), ".env.example")
	data, err := os.ReadFile(envPath)
	require.NoError(t, err)

	envContent := string(data)
	require.Contains(t, envContent, "MONGODB_ROOT_PASSWORD=")
	require.Contains(t, envContent, "REDIS_PASSWORD=")
	require.Contains(t, envContent, "AUTH_OWNER_TOKEN_SHA256=")
	require.NotContains(t, envContent, "change-me-")
	require.NotContains(t, envContent, "KEYCLOAK")
}

func TestDockerComposeProd_MongoReplicaSetReadinessAndDependencies(t *testing.T) {
	t.Parallel()

	composePath := filepath.Join(repoRoot(t), "docker-compose.prod.yml")
	composeData := readYAMLMap(t, composePath)

	services := mustMapValue(t, composeData["services"], "services must be a map")
	mongodb := mustMapValue(t, services["mongodb"], "mongodb service must be a map")
	requireLoopbackPorts(t, mongodb, "mongodb")
	entrypoint := strings.Join(stringifySlice(t, mustSliceValue(t, mongodb["entrypoint"], "mongodb entrypoint")), " ")
	require.Contains(t, entrypoint, "--replSet rs0", "transactions need a replica set")
	require.Contains(t, entrypoint, "--keyFile", "a replica set with auth does not start without a key file")

	healthcheck := mustMapValue(t, mongodb["healthcheck"], "mongodb healthcheck is required")
	healthcheckCommand := mustSliceValue(t, healthcheck["test"], "mongodb healthcheck test must be a list")
	joinedHealthcheck := strings.Join(stringifySlice(t, healthcheckCommand), " ")
	require.Contains(t, joinedHealthcheck, "--authenticationDatabase")
	require.Contains(t, joinedHealthcheck, "db.adminCommand('ping').ok")

	mongoInit := mustMapValue(t, services["mongo-init"], "mongo-init service must be a map")
	command, ok := mongoInit["command"].(string)
	require.True(t, ok, "mongo-init command must be a string")
	require.Contains(t, command, "until mongosh --host mongodb:27017 --username")
	require.Contains(t, command, "--authenticationDatabase admin")
	require.Contains(t, command, "db.adminCommand('ping')")

	app := mustMapValue(t, services["app"], "app service must be a map")
	dependsOn := mustMapValue(t, app["depends_on"], "app depends_on must be a map")
	mongodbDependency := mustMapValue(t, dependsOn["mongodb"], "app must depend on mongodb")

	condition, ok := mongodbDependency["condition"].(string)
	require.True(t, ok, "mongodb dependency condition must be a string")
	require.Equal(t, "service_healthy", condition)

	redis := mustMapValue(t, services["redis"], "redis service must be a map")
	requireLoopbackPorts(t, redis, "redis")

	redisCommand := mustSliceValue(t, redis["command"], "redis command must be a list")
	require.Contains(t, strings.Join(stringifySlice(t, redisCommand), " "), "--requirepass")
}

func TestMakefile_HasDockerProductionTargets(t *testing.T) {
	t.Parallel()

	makefilePath := filepath.Join(repoRoot(t), "Makefile")
	data, err := os.ReadFile(makefilePath)
	require.NoError(t, err)

	content := string(data)
	require.Contains(t, content, "docker-build:")
	require.Contains(t, content, "docker build -t flowra:latest .")
	require.Contains(t, content, "docker-prod-up:")
	require.Contains(t, content, "docker compose -f docker-compose.prod.yml up -d --build")
	require.Contains(t, content, "docker-prod-down:")
	require.Contains(t, content, "docker compose -f docker-compose.prod.yml down")
	require.Contains(t, content, "docker-prod-logs:")
	require.Contains(t, content, "docker compose -f docker-compose.prod.yml logs -f")
}

func TestDockerfile_UsesProductionConfig(t *testing.T) {
	t.Parallel()

	dockerfilePath := filepath.Join(repoRoot(t), "Dockerfile")
	data, err := os.ReadFile(dockerfilePath)
	require.NoError(t, err)

	content := string(data)
	require.Contains(t, content, "configs/config.prod.yaml /etc/flowra/config.yaml")
	require.NotContains(t, content, "configs/config.yaml /etc/flowra/config.yaml")
}

func TestConfigProd_HasNoBakedSecrets(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(repoRoot(t), "configs", "config.prod.yaml")
	configData := readYAMLMap(t, configPath)

	auth := mustMapValue(t, configData["auth"], "auth section is required")
	require.Empty(t, auth["tokens"], "token digests come from the environment")
	require.Empty(t, auth["owner_token_sha256"], "token digests come from the environment")

	server := mustMapValue(t, configData["server"], "server section is required")
	require.Equal(t, "127.0.0.1", server["host"], "the API refuses a non-loopback address")

	rawConfig, err := os.ReadFile(configPath)
	require.NoError(t, err)
	raw := string(rawConfig)
	require.NotContains(t, raw, "dev-secret-change-in-production")
	require.NotContains(t, raw, "admin123")
}

func TestDeploymentDocs_DockerSelfHostedSectionAndEnvNames(t *testing.T) {
	t.Parallel()

	deploymentPath := filepath.Join(repoRoot(t), "docs", "DEPLOYMENT.md")
	data, err := os.ReadFile(deploymentPath)
	require.NoError(t, err)

	content := string(data)
	require.Contains(t, content, "## Docker (self-hosted)")
	require.Contains(t, content, "docker compose -f docker-compose.prod.yml up -d")
	require.Contains(t, content, "uploads_data")
	require.Contains(t, content, "mongodb_data")
	require.Contains(t, content, "redis_data")
	require.Contains(t, content, "MONGODB_ROOT_PASSWORD")
	require.Contains(t, content, "REDIS_PASSWORD")
	require.Contains(t, content, "AUTH_OWNER_TOKEN_SHA256")
	require.NotContains(t, content, "KEYCLOAK")
	require.Contains(t, content, "FLOWRA_WORKER=true")
	require.Contains(t, content, "--with-worker")

	require.NotContains(t, content, "FLOWRA_MONGODB_URI")
	require.NotContains(t, content, "FLOWRA_REDIS_ADDR")
	require.NotContains(t, content, "FLOWRA_KEYCLOAK_URL")
	require.NotContains(t, content, "FLOWRA_AUTH_JWT_SECRET")
	require.NotContains(t, content, "FLOWRA_LOG_LEVEL")
	require.NotContains(t, content, "FLOWRA_ENV")
}

// requireLoopbackPorts checks that a service publishes ports only on the host loopback.
func requireLoopbackPorts(t *testing.T, service map[string]any, name string) {
	t.Helper()
	ports := mustSliceValue(t, service["ports"], name+" ports must be a list")
	for _, port := range stringifySlice(t, ports) {
		require.True(t, strings.HasPrefix(port, "127.0.0.1:"), "%s publishes %q beyond loopback", name, port)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok)

	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
}

func readYAMLMap(t *testing.T, path string) map[string]any {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, yaml.Unmarshal(data, &decoded))

	return decoded
}

func mustMapValue(t *testing.T, value any, msg string) map[string]any {
	t.Helper()

	typed, ok := value.(map[string]any)
	require.True(t, ok, msg)

	return typed
}

func mustSliceValue(t *testing.T, value any, msg string) []any {
	t.Helper()

	typed, ok := value.([]any)
	require.True(t, ok, msg)

	return typed
}

func stringifySlice(t *testing.T, values []any) []string {
	t.Helper()

	strValues := make([]string, 0, len(values))
	for _, value := range values {
		strValue, ok := value.(string)
		require.True(t, ok, "expected list values to be strings")
		strValues = append(strValues, strValue)
	}

	return strValues
}
