package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/tests/testutil"
)

const fixturePath = "../../tests/fixtures/leadfeed/full.jsonl"

type cli struct {
	t    *testing.T
	base []string
}

func newCLI(t *testing.T) cli {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	cont, err := testutil.GetSharedMongoContainer(ctx)
	require.NoError(t, err)
	_, db := testutil.SetupTestMongoDBWithClient(t)

	return cli{t: t, base: []string{"-mongo-uri", cont.URI, "-mongo-db", db.Name()}}
}

func (c cli) run(stdin string, args ...string) (int, string, string) {
	c.t.Helper()
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), append(append([]string{}, c.base...), args...),
		strings.NewReader(stdin), &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func fixture(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(fixturePath)
	require.NoError(t, err)
	return string(raw)
}

func TestRun_ImportFromStdinThenStatus(t *testing.T) {
	c := newCLI(t)

	code, stdout, stderr := c.run("", "status")
	require.Equal(t, exitOK, code, stderr)
	assert.Contains(t, stdout, "published: nothing yet")
	assert.Contains(t, stdout, "last attempt: none")

	code, stdout, stderr = c.run(fixture(t), "import")
	require.Equal(t, exitOK, code, stderr)
	assert.Contains(t, stdout, "published revision=389227faca902622")
	assert.Contains(t, stdout, "tasks=4 records=9")

	code, stdout, stderr = c.run(fixture(t), "import", "-")
	require.Equal(t, exitOK, code, stderr)
	assert.Contains(t, stdout, "refreshed revision=389227faca902622", "the same snapshot again changes nothing")

	code, stdout, stderr = c.run("", "status")
	require.Equal(t, exitOK, code, stderr)
	assert.Contains(t, stdout, "published: revision=389227faca902622")
	assert.Contains(t, stdout, "last attempt: refreshed")
}

func TestRun_ImportFromFile(t *testing.T) {
	c := newCLI(t)

	code, stdout, stderr := c.run("", "import", fixturePath)

	require.Equal(t, exitOK, code, stderr)
	assert.Contains(t, stdout, "published revision=389227faca902622")
}

func TestRun_RejectedImportKeepsFeedAndFails(t *testing.T) {
	c := newCLI(t)
	code, _, stderr := c.run(fixture(t), "import")
	require.Equal(t, exitOK, code, stderr)

	lines := strings.Split(strings.TrimRight(fixture(t), "\n"), "\n")
	truncated := strings.Join(lines[:len(lines)-1], "\n")

	tests := []struct {
		name    string
		stdin   string
		args    []string
		wantErr string
	}{
		{name: "truncated snapshot", stdin: truncated, args: []string{"import"}, wantErr: "no footer"},
		{name: "empty input", stdin: "", args: []string{"import"}, wantErr: "no footer"},
		{name: "over the size limit", stdin: fixture(t), args: []string{"-max-bytes", "512", "import"},
			wantErr: "larger than the import limit"},
		{name: "missing file", args: []string{"import", filepath.Join(t.TempDir(), "absent.jsonl")},
			wantErr: "no such file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rejectCode, stdout, rejectErr := c.run(tt.stdin, tt.args...)

			assert.Equal(t, exitFailed, rejectCode)
			assert.Empty(t, stdout)
			assert.Contains(t, rejectErr, tt.wantErr)

			statusCode, status, _ := c.run("", "status")
			require.Equal(t, exitOK, statusCode)
			assert.Contains(t, status, "published: revision=389227faca902622", "the previous snapshot stays")
		})
	}

	_, status, _ := c.run("", "status")
	assert.Contains(t, status, "last attempt: rejected")
	assert.Contains(t, status, "last error:")
}

func TestRun_Usage(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "no command"},
		{name: "unknown command", args: []string{"purge"}},
		{name: "unknown flag", args: []string{"-nope", "status"}},
		{name: "too many import arguments", args: []string{"import", "a", "b"}},
		{name: "status with an argument", args: []string{"status", "x"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			code := run(context.Background(), tt.args, strings.NewReader(""), &stdout, &stderr)

			assert.Equal(t, exitUsage, code)
			assert.Contains(t, stderr.String(), "usage:")
		})
	}
}

func TestRun_UnreachableMongoFails(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run(context.Background(),
		[]string{"-mongo-uri", "mongodb://127.0.0.1:1/?replicaSet=rs0&serverSelectionTimeoutMS=300", "status"},
		strings.NewReader(""), &stdout, &stderr)

	assert.Equal(t, exitFailed, code)
	assert.NotEmpty(t, stderr.String())
}
