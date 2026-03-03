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

	requiredServices := []string{"mongodb", "mongo-init", "redis", "keycloak-db", "keycloak", "app"}
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
		"KEYCLOAK_PUBLIC_URL",
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

	mongoURI, ok := environment["MONGODB_URI"].(string)
	require.True(t, ok, "MONGODB_URI must be a string")
	require.Contains(t, mongoURI, "replicaSet=rs0")

	redisAddr, ok := environment["REDIS_ADDR"].(string)
	require.True(t, ok, "REDIS_ADDR must be a string")
	require.Equal(t, "redis:6379", redisAddr)

	workerMode, ok := environment["FLOWRA_WORKER"].(string)
	require.True(t, ok, "FLOWRA_WORKER must be a string")
	require.Contains(t, workerMode, "true")

	require.Equal(
		t,
		"${KEYCLOAK_CLIENT_SECRET:?KEYCLOAK_CLIENT_SECRET is required}",
		environment["KEYCLOAK_CLIENT_SECRET"],
	)
	require.Equal(
		t,
		"${KEYCLOAK_ADMIN_PASSWORD:?KEYCLOAK_ADMIN_PASSWORD is required}",
		environment["KEYCLOAK_ADMIN_PASSWORD"],
	)
	require.Equal(t, "${AUTH_JWT_SECRET:?AUTH_JWT_SECRET is required}", environment["AUTH_JWT_SECRET"])
}

func TestEnvExample_RequiresExplicitSecrets(t *testing.T) {
	t.Parallel()

	envPath := filepath.Join(repoRoot(t), ".env.example")
	data, err := os.ReadFile(envPath)
	require.NoError(t, err)

	envContent := string(data)
	require.Contains(t, envContent, "KEYCLOAK_CLIENT_SECRET=")
	require.Contains(t, envContent, "KEYCLOAK_ADMIN_PASSWORD=")
	require.Contains(t, envContent, "KEYCLOAK_DB_PASSWORD=")
	require.Contains(t, envContent, "KEYCLOAK_PUBLIC_URL=")
	require.Contains(t, envContent, "FLOWRA_PUBLIC_URL=")
	require.Contains(t, envContent, "AUTH_JWT_SECRET=")
	require.NotContains(t, envContent, "change-me-")
}

func TestDockerComposeProd_KeycloakRealmImportAndAppDependency(t *testing.T) {
	t.Parallel()

	composePath := filepath.Join(repoRoot(t), "docker-compose.prod.yml")
	composeData := readYAMLMap(t, composePath)

	services := mustMapValue(t, composeData["services"], "services must be a map")
	keycloak := mustMapValue(t, services["keycloak"], "keycloak service must be a map")
	keycloakDB := mustMapValue(t, services["keycloak-db"], "keycloak-db service must be a map")

	keycloakDBEnv := mustMapValue(t, keycloakDB["environment"], "keycloak-db environment is required")
	require.Equal(
		t,
		"${KEYCLOAK_DB_PASSWORD:?KEYCLOAK_DB_PASSWORD is required}",
		keycloakDBEnv["POSTGRES_PASSWORD"],
	)

	keycloakDependsOn := mustMapValue(t, keycloak["depends_on"], "keycloak depends_on is required")
	keycloakDBDependency := mustMapValue(t, keycloakDependsOn["keycloak-db"], "keycloak must depend on keycloak-db")
	require.Equal(t, "service_healthy", keycloakDBDependency["condition"])

	entrypoint := mustSliceValue(t, keycloak["entrypoint"], "keycloak entrypoint must be a list")
	entrypointValues := stringifySlice(t, entrypoint)
	require.GreaterOrEqual(t, len(entrypointValues), 3)
	require.Equal(t, "/bin/bash", entrypointValues[0])
	require.Equal(t, "-ec", entrypointValues[1])

	commandScript := entrypointValues[2]
	require.Contains(t, commandScript, "kc.sh start")
	require.Contains(t, commandScript, "--import-realm")
	require.Contains(t, commandScript, "json_escape")
	require.Contains(t, commandScript, "__FLOWRA_PUBLIC_URL__")
	require.Contains(t, commandScript, "--hostname-strict=true")
	require.NotContains(t, commandScript, "--hostname-strict=false")
	require.NotContains(t, commandScript, "start-dev")

	_, hasCommand := keycloak["command"]
	require.False(t, hasCommand, "keycloak command should not be split from the bash script entrypoint")

	volumes := mustSliceValue(t, keycloak["volumes"], "keycloak volumes must be a list")
	require.Contains(
		t,
		stringifySlice(t, volumes),
		"./configs/keycloak-prod/realm-export.template.json:/opt/keycloak/realm-template/realm-export.template.json:ro",
	)

	healthcheck := mustMapValue(t, keycloak["healthcheck"], "keycloak healthcheck is required")
	healthcheckCommand := mustSliceValue(t, healthcheck["test"], "keycloak healthcheck test must be a list")
	joinedHealthcheck := strings.Join(stringifySlice(t, healthcheckCommand), " ")
	require.Contains(t, joinedHealthcheck, "/realms/flowra/.well-known/openid-configuration")
	require.Contains(t, joinedHealthcheck, "200 OK")

	keycloakEnv := mustMapValue(t, keycloak["environment"], "keycloak environment is required")
	require.Equal(t, "postgres", keycloakEnv["KC_DB"])
	require.Equal(
		t,
		"${KEYCLOAK_CLIENT_SECRET:?KEYCLOAK_CLIENT_SECRET is required}",
		keycloakEnv["KEYCLOAK_CLIENT_SECRET"],
	)
	require.Equal(t, "${FLOWRA_PUBLIC_URL:?FLOWRA_PUBLIC_URL is required}", keycloakEnv["FLOWRA_PUBLIC_URL"])
	require.Equal(t, "${KEYCLOAK_PUBLIC_URL:?KEYCLOAK_PUBLIC_URL is required}", keycloakEnv["KEYCLOAK_PUBLIC_URL"])
	keycloakAdminPassword, ok := keycloakEnv["KEYCLOAK_ADMIN_PASSWORD"].(string)
	require.True(t, ok, "keycloak admin password value must be a string")

	app := mustMapValue(t, services["app"], "app service must be a map")
	dependsOn := mustMapValue(t, app["depends_on"], "app depends_on must be a map")
	keycloakDependency := mustMapValue(t, dependsOn["keycloak"], "app must depend on keycloak")

	condition, ok := keycloakDependency["condition"].(string)
	require.True(t, ok, "keycloak dependency condition must be a string")
	require.Equal(t, "service_healthy", condition)

	appEnv := mustMapValue(t, app["environment"], "app environment is required")
	appAdminPassword, ok := appEnv["KEYCLOAK_ADMIN_PASSWORD"].(string)
	require.True(t, ok, "app keycloak admin password value must be a string")
	require.Equal(t, keycloakAdminPassword, appAdminPassword)
	require.Equal(t, "${KEYCLOAK_PUBLIC_URL:?KEYCLOAK_PUBLIC_URL is required}", appEnv["KEYCLOAK_PUBLIC_URL"])
}

func TestDockerComposeProd_MongoReplicaSetReadinessAndDependencies(t *testing.T) {
	t.Parallel()

	composePath := filepath.Join(repoRoot(t), "docker-compose.prod.yml")
	composeData := readYAMLMap(t, composePath)

	services := mustMapValue(t, composeData["services"], "services must be a map")
	mongodb := mustMapValue(t, services["mongodb"], "mongodb service must be a map")
	_, hasMongoPorts := mongodb["ports"]
	require.False(t, hasMongoPorts, "mongodb should not expose host ports in production compose")

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

	redis := mustMapValue(t, services["redis"], "redis service must be a map")
	_, hasRedisPorts := redis["ports"]
	require.False(t, hasRedisPorts, "redis should not expose host ports in production compose")
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
	require.Empty(t, auth["jwt_secret"])

	keycloak := mustMapValue(t, configData["keycloak"], "keycloak section is required")
	require.Empty(t, keycloak["client_secret"])
	require.Empty(t, keycloak["admin_password"])

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
	require.Contains(t, content, "keycloak_db_data")
	require.Contains(t, content, "KEYCLOAK_DB_PASSWORD")
	require.Contains(t, content, "KEYCLOAK_PUBLIC_URL")
	require.Contains(t, content, "FLOWRA_PUBLIC_URL")
	require.Contains(t, content, "configs/keycloak-prod/realm-export.template.json")
	require.Contains(t, content, "FLOWRA_WORKER=true")
	require.Contains(t, content, "--with-worker")

	require.NotContains(t, content, "FLOWRA_MONGODB_URI")
	require.NotContains(t, content, "FLOWRA_REDIS_ADDR")
	require.NotContains(t, content, "FLOWRA_KEYCLOAK_URL")
	require.NotContains(t, content, "FLOWRA_AUTH_JWT_SECRET")
	require.NotContains(t, content, "FLOWRA_LOG_LEVEL")
	require.NotContains(t, content, "FLOWRA_ENV")
}

func TestComposeAndRealmTemplate_UsesRuntimeSecretAndNoSeededUsers(t *testing.T) {
	t.Parallel()

	composePath := filepath.Join(repoRoot(t), "docker-compose.prod.yml")
	composeData := readYAMLMap(t, composePath)
	services := mustMapValue(t, composeData["services"], "services must be a map")

	app := mustMapValue(t, services["app"], "app service must be a map")
	appEnv := mustMapValue(t, app["environment"], "app environment is required")
	clientSecret, ok := appEnv["KEYCLOAK_CLIENT_SECRET"].(string)
	require.True(t, ok, "KEYCLOAK_CLIENT_SECRET must be a string")
	require.Equal(t, "${KEYCLOAK_CLIENT_SECRET:?KEYCLOAK_CLIENT_SECRET is required}", clientSecret)

	keycloak := mustMapValue(t, services["keycloak"], "keycloak service must be a map")
	keycloakEnv := mustMapValue(t, keycloak["environment"], "keycloak environment is required")
	require.Equal(t, clientSecret, keycloakEnv["KEYCLOAK_CLIENT_SECRET"])

	realmPath := filepath.Join(repoRoot(t), "configs", "keycloak-prod", "realm-export.template.json")
	realmData, err := os.ReadFile(realmPath)
	require.NoError(t, err)
	require.Contains(t, string(realmData), "\"secret\": \"__KEYCLOAK_CLIENT_SECRET__\"")
	require.Contains(t, string(realmData), "\"__FLOWRA_PUBLIC_URL__/auth/callback\"")
	require.Contains(t, string(realmData), "\"registrationAllowed\": false")
	require.Contains(t, string(realmData), "\"verifyEmail\": true")
	require.NotContains(t, string(realmData), "\"users\": [")
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
