package scripts_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/config"
)

type installEnv struct {
	root string
	home string
	bin  string
	log  string
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	return filepath.Dir(filepath.Dir(wd))
}

func writeFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), mode))
}

func copyFile(t *testing.T, from, to string) {
	t.Helper()
	data, err := os.ReadFile(from)
	require.NoError(t, err)
	writeFile(t, to, string(data), 0o755)
}

func newInstallEnv(t *testing.T) installEnv {
	t.Helper()
	src := repoRoot(t)
	dir := t.TempDir()
	env := installEnv{
		root: filepath.Join(dir, "repo"),
		home: filepath.Join(dir, "home"),
		bin:  filepath.Join(dir, "stubs"),
		log:  filepath.Join(dir, "calls.log"),
	}
	require.NoError(t, os.MkdirAll(env.home, 0o755))

	files := []string{
		"scripts/install-user-units.sh",
		"scripts/leadfeed-sync.sh",
		"scripts/units/flowra-api.service.in",
		"scripts/units/leadfeed-sync.service.in",
		"scripts/units/leadfeed-sync.timer.in",
		"configs/config.yaml",
		"docker-compose.yml",
	}
	for _, f := range files {
		copyFile(t, filepath.Join(src, f), filepath.Join(env.root, f))
	}

	stub := "#!/usr/bin/env bash\necho \"$(basename \"$0\") $*\" >> " + env.log + "\n"
	writeFile(t, filepath.Join(env.bin, "systemctl"), stub, 0o755)
	writeFile(t, filepath.Join(env.bin, "docker-compose"), stub, 0o755)
	writeFile(t, filepath.Join(env.bin, "go"), stub+
		"while [[ $# -gt 0 ]]; do if [[ $1 == -o ]]; then mkdir -p \"$(dirname \"$2\")\"; "+
		"printf '#!/bin/sh\\n' > \"$2\"; chmod +x \"$2\"; fi; shift; done\n", 0o755)
	return env
}

func (e installEnv) run(t *testing.T) string {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "bash", filepath.Join(e.root, "scripts/install-user-units.sh"))
	cmd.Env = []string{
		"HOME=" + e.home,
		"PATH=" + e.bin + ":/usr/bin:/bin",
		"LEAD_TASK=/opt/fake/lead-task",
		"LEAD_SESSION=lead-test",
		"AGTERM_SOCKET=" + filepath.Join(e.home, "agterm.sock"),
	}
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	return string(out)
}

var tokenLine = regexp.MustCompile(`(?m)^(owner[^:]*|phone): ([0-9a-f]{64})$`)

func printedTokens(out string) map[string]string {
	tokens := map[string]string{}
	for _, m := range tokenLine.FindAllStringSubmatch(out, -1) {
		name := "owner"
		if m[1] == "phone" {
			name = "phone"
		}
		tokens[name] = m[2]
	}
	return tokens
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}

func TestInstallUserUnits_FirstRunWritesFilesAndHidesTokens(t *testing.T) {
	env := newInstallEnv(t)
	out := env.run(t)

	tokens := printedTokens(out)
	require.Len(t, tokens, 2, out)

	cfgDir := filepath.Join(env.home, ".config/flowra")
	for _, name := range []string{"flowra.env", "config.yaml"} {
		info, err := os.Stat(filepath.Join(cfgDir, name))
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0o600), info.Mode().Perm(), name)
	}

	require.NoError(t, filepath.WalkDir(env.home, func(path string, d os.DirEntry, err error) error {
		require.NoError(t, err)
		if d.IsDir() {
			return nil
		}
		content := readFile(t, path)
		for name, token := range tokens {
			require.NotContains(t, content, token, "%s token leaked into %s", name, path)
		}
		return nil
	}))

	envFile := readFile(t, filepath.Join(cfgDir, "flowra.env"))
	for _, want := range []string{
		"CONFIG_PATH=" + filepath.Join(cfgDir, "config.yaml"),
		"AUTH_OWNER_TOKEN_SHA256=",
		"LEAD_TASK=/opt/fake/lead-task",
		"LEAD_SESSION=lead-test",
		"AGTERM_SOCKET=" + filepath.Join(env.home, "agterm.sock"),
		"PATH=",
	} {
		require.Contains(t, envFile, want)
	}

	unitDir := filepath.Join(env.home, ".config/systemd/user")
	for _, unit := range []string{"flowra-api.service", "leadfeed-sync.service", "leadfeed-sync.timer"} {
		content := readFile(t, filepath.Join(unitDir, unit))
		require.NotContains(t, content, "@ROOT@")
		require.NotContains(t, content, "@ENV_FILE@")
	}
	api := readFile(t, filepath.Join(unitDir, "flowra-api.service"))
	require.Contains(t, api, "WorkingDirectory="+env.root)
	require.Contains(t, api, "EnvironmentFile="+filepath.Join(cfgDir, "flowra.env"))
	require.Contains(t, api, "Restart=on-failure")

	calls := readFile(t, env.log)
	compose := filepath.Join(env.root, "docker-compose.yml")
	require.Contains(t, calls, "docker-compose -f "+compose+" up -d mongodb redis mongo-init")
	require.Contains(t, calls, "systemctl --user enable --now flowra-api.service leadfeed-sync.timer")
	require.FileExists(t, filepath.Join(env.root, "bin/api"))
	require.FileExists(t, filepath.Join(env.root, "bin/leadfeed"))
}

func TestInstallUserUnits_RepeatKeepsTokensAndRefreshesUnits(t *testing.T) {
	env := newInstallEnv(t)
	first := env.run(t)
	require.Len(t, printedTokens(first), 2)

	cfgDir := filepath.Join(env.home, ".config/flowra")
	envBefore := readFile(t, filepath.Join(cfgDir, "flowra.env"))
	cfgBefore := readFile(t, filepath.Join(cfgDir, "config.yaml"))

	unit := filepath.Join(env.home, ".config/systemd/user/flowra-api.service")
	require.NoError(t, os.WriteFile(unit, []byte("stale"), 0o600))

	second := env.run(t)
	require.Empty(t, printedTokens(second), "tokens are printed once")
	require.NotContains(t, second, "Save these tokens")
	require.Equal(t, envBefore, readFile(t, filepath.Join(cfgDir, "flowra.env")))
	require.Equal(t, cfgBefore, readFile(t, filepath.Join(cfgDir, "config.yaml")))
	require.Contains(t, readFile(t, unit), "ExecStart="+filepath.Join(env.root, "bin/api"))
	require.Contains(t, readFile(t, env.log), "systemctl --user try-restart flowra-api.service")
}

func TestInstallUserUnits_ConfigPassesAPIAuthCheck(t *testing.T) {
	env := newInstallEnv(t)
	env.run(t)

	cfgDir := filepath.Join(env.home, ".config/flowra")
	for _, line := range strings.Split(readFile(t, filepath.Join(cfgDir, "flowra.env")), "\n") {
		if key, value, ok := strings.Cut(line, "="); ok && key == "AUTH_OWNER_TOKEN_SHA256" {
			t.Setenv(key, value)
		}
	}

	cfg, err := config.NewLoader().Load(filepath.Join(cfgDir, "config.yaml"))
	require.NoError(t, err)
	require.NoError(t, cfg.ValidateAuth())

	names := []string{}
	for _, token := range cfg.Auth.OwnerTokens() {
		names = append(names, token.Name)
	}
	require.ElementsMatch(t, []string{"owner", "phone"}, names)
	require.Equal(t, "127.0.0.1", cfg.Server.Host)
}

func TestInstallUserUnits_UnitsPassSystemdVerify(t *testing.T) {
	analyze, err := exec.LookPath("systemd-analyze")
	if err != nil {
		t.Skip("systemd-analyze not installed")
	}
	env := newInstallEnv(t)
	env.run(t)

	unitDir := filepath.Join(env.home, ".config/systemd/user")
	units := []string{"flowra-api.service", "leadfeed-sync.service", "leadfeed-sync.timer"}
	args := []string{"--user", "verify"}
	for _, u := range units {
		args = append(args, filepath.Join(unitDir, u))
	}
	cmd := exec.CommandContext(t.Context(), analyze, args...)
	cmd.Env = append(os.Environ(), "HOME="+env.home)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
}
