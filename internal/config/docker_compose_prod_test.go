package config_test

import (
	"os"
	"path/filepath"
	"runtime"
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

	requiredServices := []string{"mongodb", "mongo-init", "redis", "keycloak", "app"}
	for _, name := range requiredServices {
		require.Contains(t, services, name)
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
		"KEYCLOAK_URL",
		"KEYCLOAK_REALM",
		"KEYCLOAK_CLIENT_ID",
		"KEYCLOAK_CLIENT_SECRET",
		"KEYCLOAK_ADMIN_USERNAME",
		"KEYCLOAK_ADMIN_PASSWORD",
		"AUTH_JWT_SECRET",
	}
	for _, envKey := range requiredEnv {
		require.Contains(t, environment, envKey)
	}
}

func TestEnvExample_HasSecretPlaceholders(t *testing.T) {
	t.Parallel()

	envPath := filepath.Join(repoRoot(t), ".env.example")
	data, err := os.ReadFile(envPath)
	require.NoError(t, err)

	envContent := string(data)
	require.Contains(t, envContent, "KEYCLOAK_CLIENT_SECRET=change-me-keycloak-client-secret")
	require.Contains(t, envContent, "KEYCLOAK_ADMIN_PASSWORD=change-me-keycloak-admin-password")
	require.Contains(t, envContent, "AUTH_JWT_SECRET=change-me-auth-jwt-secret")
}

func TestDockerComposeProd_KeycloakRealmImportAndAppDependency(t *testing.T) {
	t.Parallel()

	composePath := filepath.Join(repoRoot(t), "docker-compose.prod.yml")
	composeData := readYAMLMap(t, composePath)

	services := mustMapValue(t, composeData["services"], "services must be a map")
	keycloak := mustMapValue(t, services["keycloak"], "keycloak service must be a map")

	command := mustSliceValue(t, keycloak["command"], "keycloak command must be a list")
	require.Contains(t, stringifySlice(t, command), "--import-realm")

	volumes := mustSliceValue(t, keycloak["volumes"], "keycloak volumes must be a list")
	require.Contains(
		t,
		stringifySlice(t, volumes),
		"./configs/keycloak/realm-export.json:/opt/keycloak/data/import/realm-export.json:ro",
	)

	healthcheck := mustMapValue(t, keycloak["healthcheck"], "keycloak healthcheck is required")
	healthcheckCommand := mustSliceValue(t, healthcheck["test"], "keycloak healthcheck test must be a list")
	require.Contains(
		t,
		stringifySlice(t, healthcheckCommand),
		"exec 3<>/dev/tcp/127.0.0.1/8080 && printf 'GET /realms/flowra/.well-known/openid-configuration HTTP/1.1\\r\\nHost: localhost\\r\\nConnection: close\\r\\n\\r\\n' >&3 && grep -q '200 OK' <&3",
	)

	app := mustMapValue(t, services["app"], "app service must be a map")
	dependsOn := mustMapValue(t, app["depends_on"], "app depends_on must be a map")
	keycloakDependency := mustMapValue(t, dependsOn["keycloak"], "app must depend on keycloak")

	condition, ok := keycloakDependency["condition"].(string)
	require.True(t, ok, "keycloak dependency condition must be a string")
	require.Equal(t, "service_healthy", condition)
}

func TestDockerComposeProd_MongoReplicaSetReadinessAndDependencies(t *testing.T) {
	t.Parallel()

	composePath := filepath.Join(repoRoot(t), "docker-compose.prod.yml")
	composeData := readYAMLMap(t, composePath)

	services := mustMapValue(t, composeData["services"], "services must be a map")
	mongodb := mustMapValue(t, services["mongodb"], "mongodb service must be a map")

	healthcheck := mustMapValue(t, mongodb["healthcheck"], "mongodb healthcheck is required")
	healthcheckCommand := mustSliceValue(t, healthcheck["test"], "mongodb healthcheck test must be a list")
	require.Contains(t, stringifySlice(t, healthcheckCommand), "rs.status().ok")

	mongoInit := mustMapValue(t, services["mongo-init"], "mongo-init service must be a map")
	command, ok := mongoInit["command"].(string)
	require.True(t, ok, "mongo-init command must be a string")
	require.Contains(t, command, "until mongosh --host mongodb:27017 --eval")
	require.Contains(t, command, "db.adminCommand('ping')")

	app := mustMapValue(t, services["app"], "app service must be a map")
	dependsOn := mustMapValue(t, app["depends_on"], "app depends_on must be a map")
	mongodbDependency := mustMapValue(t, dependsOn["mongodb"], "app must depend on mongodb")

	condition, ok := mongodbDependency["condition"].(string)
	require.True(t, ok, "mongodb dependency condition must be a string")
	require.Equal(t, "service_healthy", condition)
}

func TestMakefile_HasDockerProductionTargets(t *testing.T) {
	t.Parallel()

	makefilePath := filepath.Join(repoRoot(t), "Makefile")
	data, err := os.ReadFile(makefilePath)
	require.NoError(t, err)

	content := string(data)
	require.Contains(t, content, "docker-build: ## Build production Docker image\n\tdocker build -t flowra:latest .")
	require.Contains(
		t,
		content,
		"docker-prod-up: ## Start production Docker stack\n\tdocker compose -f docker-compose.prod.yml up -d --build",
	)
	require.Contains(
		t,
		content,
		"docker-prod-down: ## Stop production Docker stack\n\tdocker compose -f docker-compose.prod.yml down",
	)
	require.Contains(
		t,
		content,
		"docker-prod-logs: ## Show production Docker logs\n\tdocker compose -f docker-compose.prod.yml logs -f",
	)
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
