// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package tmpdir

import (
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	xfmt "golang.org/x/exp/errors/fmt"
)

type testingRunIDContextKey struct{}

// ContextWithTestingRunID stores a testing run-id in context so callers in the
// same command can share one tmp subtree.
func ContextWithTestingRunID(ctx context.Context, runID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return ctx
	}
	return context.WithValue(ctx, testingRunIDContextKey{}, runID)
}

// TestingRunIDFromContext reads the testing run-id from context.
func TestingRunIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(testingRunIDContextKey{}).(string)
	return strings.TrimSpace(v)
}

// NewTestingRunID generates a filename-safe run-id for testing temp paths.
func NewTestingRunID() string {
	ts := time.Now().UTC().Format("060102-150405")
	buf := make([]byte, 3)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%s-r%x", ts, time.Now().Unix()%0x1000000)
	}
	return fmt.Sprintf("%s-r%s", ts, hex.EncodeToString(buf))
}

func normalizeTestingRunID(runID string) (string, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return "", nil
	}
	runID = filepath.Clean(runID)
	if runID == "." || runID == ".." || runID == string(filepath.Separator) {
		return "", xfmt.Errorf("runID is invalid")
	}
	if strings.Contains(runID, "/") || strings.Contains(runID, "\\") {
		return "", xfmt.Errorf("runID must be a single path segment")
	}
	return runID, nil
}

func normalizeWorkspaceRoot(workspaceRoot string) string {
	workspaceRoot = strings.TrimSpace(workspaceRoot)
	if workspaceRoot == "" || workspaceRoot == "." {
		if cwd, err := os.Getwd(); err == nil {
			workspaceRoot = cwd
		}
	}
	if abs, err := filepath.Abs(workspaceRoot); err == nil {
		workspaceRoot = abs
	}
	if workspaceRoot == "" {
		workspaceRoot = "."
	}
	return workspaceRoot
}

func workspaceKey(workspaceRoot string) string {
	workspaceRoot = normalizeWorkspaceRoot(workspaceRoot)
	hash := sha1.Sum([]byte(workspaceRoot))
	return hex.EncodeToString(hash[:5])
}

func normalizeTmpRoot(tmpRoot string) (string, error) {
	tmpRoot = strings.TrimSpace(tmpRoot)
	if tmpRoot == "" {
		return "", xfmt.Errorf("tmpRoot is required")
	}
	if absTmpRoot, err := filepath.Abs(tmpRoot); err == nil {
		tmpRoot = absTmpRoot
	}
	tmpRoot = filepath.Clean(tmpRoot)
	if tmpRoot == "." || tmpRoot == string(filepath.Separator) {
		return "", xfmt.Errorf("tmpRoot must be a non-root directory")
	}
	return tmpRoot, nil
}

// ResolveWorkspaceTmpDir returns a workspace-scoped tmp directory under
// <tmp-root>/workspaces/<workspace-key>.
func ResolveWorkspaceTmpDir(workspaceRoot string, tmpRoot string) (string, error) {
	normalizedTmpRoot, err := normalizeTmpRoot(tmpRoot)
	if err != nil {
		return "", err
	}
	return filepath.Join(normalizedTmpRoot, "workspaces", workspaceKey(workspaceRoot)), nil
}

// ResolveTestingTmpDir returns a testing-scoped tmp directory under
// <tmp-root>/testing/<workspace-key>/<kind>.
func ResolveTestingTmpDir(workspaceRoot string, tmpRoot string, kind string) (string, error) {
	return ResolveTestingTmpDirWithRunID(workspaceRoot, tmpRoot, kind, "")
}

// ResolveTestingTmpDirWithRunID returns a testing-scoped tmp directory under
// <tmp-root>/testing/<workspace-key>/<run-id>/<kind> when run-id is
// provided; otherwise it falls back to <...>/testing/<workspace-key>/<kind>.
func ResolveTestingTmpDirWithRunID(workspaceRoot string, tmpRoot string, kind string, runID string) (string, error) {
	kind = filepath.Clean(strings.TrimSpace(kind))
	if kind == "" || kind == "." || kind == string(filepath.Separator) {
		return "", xfmt.Errorf("kind is required")
	}
	if strings.HasPrefix(kind, ".."+string(filepath.Separator)) || kind == ".." {
		return "", xfmt.Errorf("kind must stay within testing root")
	}

	runID, err := normalizeTestingRunID(runID)
	if err != nil {
		return "", err
	}

	normalizedTmpRoot, err := normalizeTmpRoot(tmpRoot)
	if err != nil {
		return "", err
	}
	if runID == "" {
		return filepath.Join(normalizedTmpRoot, "testing", workspaceKey(workspaceRoot), kind), nil
	}
	return filepath.Join(normalizedTmpRoot, "testing", workspaceKey(workspaceRoot), runID, kind), nil
}

// ResolveTestingTmpDirFromContext resolves a testing tmp directory using
// run-id from context when available.
func ResolveTestingTmpDirFromContext(ctx context.Context, workspaceRoot string, tmpRoot string, kind string) (string, error) {
	runID := TestingRunIDFromContext(ctx)
	return ResolveTestingTmpDirWithRunID(workspaceRoot, tmpRoot, kind, runID)
}

const (
	// EnvCLITestTMP overrides the CLI test temporary root.
	EnvCLITestTMP = "CHOYSUM_TEST_TMP"
	// defaultCLITestTmpDirName is the directory name under os.TempDir() used
	// when CHOYSUM_TEST_TMP is unset.
	defaultCLITestTmpDirName = "choysum-testing"
	// CLITestingRunHomeKind is the testing kind for the shared DefaultChoysumPath
	// of one CLI test invocation (pkg/esm warm cache across apps).
	CLITestingRunHomeKind = "home"
)

// CLITestTmpRoot returns the temporary root for CLI test commands
// (unit / e2e / typecheck). Prefer CHOYSUM_TEST_TMP; otherwise use
// <os.TempDir()>/choysum-testing. This intentionally does not use the
// production config TmpPath (~/.choysum/tmp).
func CLITestTmpRoot() (string, error) {
	root := strings.TrimSpace(os.Getenv(EnvCLITestTMP))
	if root == "" {
		root = filepath.Join(os.TempDir(), defaultCLITestTmpDirName)
	}
	normalized, err := normalizeTmpRoot(root)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(normalized, 0o755); err != nil {
		return "", xfmt.Errorf("create CLI test tmp root: %w", err)
	}
	return normalized, nil
}

// ResolveCLITestingRunHome returns the shared DefaultChoysumPath for one CLI
// test run: <tmpRoot>/testing/<workspace>/<run-id>/home.
func ResolveCLITestingRunHome(ctx context.Context, workspaceRoot string, tmpRoot string) (string, error) {
	home, err := ResolveTestingTmpDirFromContext(ctx, workspaceRoot, tmpRoot, CLITestingRunHomeKind)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(home, 0o755); err != nil {
		return "", xfmt.Errorf("create CLI testing run home: %w", err)
	}
	return home, nil
}
