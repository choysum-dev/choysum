// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	testingpathing "github.com/choysum-dev/choysum/internal/testing/tmpdir"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type testStubScope struct {
	ctx context.Context
	cfg *config.Config
}

type testPassthroughTransactor struct{ rootScope scope.Scope }

func (t testPassthroughTransactor) Do(ctx context.Context, opts scope.TransactionOptions, fn scope.TxFunc) error {
	switch opts.Propagation {
	case scope.PropagationRequired:
		txScope := t.rootScope
		if ctx != nil {
			txScope = t.rootScope.WithContext(ctx)
		}
		return fn(txScope, nil)
	case scope.PropagationRequiresNew:
		return scope.ErrRequiresNewUnsupported
	case scope.PropagationNested:
		return scope.ErrNestedUnsupported
	default:
		return fmt.Errorf("%w: %q", scope.ErrInvalidTransactionPropagation, opts.Propagation)
	}
}

func (t testPassthroughTransactor) Required(ctx context.Context, fn scope.TxFunc) error {
	return t.Do(ctx, scope.TransactionOptions{Propagation: scope.PropagationRequired}, fn)
}

func (t testPassthroughTransactor) RequiresNew(ctx context.Context, fn scope.TxFunc) error {
	return t.Do(ctx, scope.TransactionOptions{Propagation: scope.PropagationRequiresNew}, fn)
}

func (t testPassthroughTransactor) Nested(ctx context.Context, fn scope.TxFunc) error {
	return t.Do(ctx, scope.TransactionOptions{Propagation: scope.PropagationNested}, fn)
}

func (e *testStubScope) Run(fn func(scope.Scope) error) error { return fn(e) }
func (e *testStubScope) Transactor() scope.Transactor {
	return testPassthroughTransactor{rootScope: e}
}
func (e *testStubScope) Session() *scope.Session { return nil }
func (e *testStubScope) WithContext(ctx context.Context) scope.Scope {
	clone := *e
	clone.ctx = ctx
	return &clone
}
func (e *testStubScope) Context() context.Context { return e.ctx }
func (e *testStubScope) Logger() *slog.Logger     { return slog.New(slog.NewTextHandler(io.Discard, nil)) }
func (e *testStubScope) Config() *config.Config   { return e.cfg }
func (e *testStubScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.cfg)
}

func TestRun(t *testing.T) {
	newEnv := func(addonsPath string) *testStubScope {
		return &testStubScope{
			ctx: context.Background(),
			cfg: &config.Config{AddonsPath: addonsPath},
		}
	}

	t.Run("returns early when context already canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := Run(ctx, RunOptions{})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context canceled, got %v", err)
		}
	})

	t.Run("no tests selected exits early", func(t *testing.T) {
		var stdout strings.Builder
		err := Run(context.Background(), RunOptions{
			Stdout: &stdout,
			Stderr: io.Discard,
		})
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
		if !strings.Contains(stdout.String(), "# no tests selected") {
			t.Fatalf("expected no tests selected message, got %q", stdout.String())
		}
	})

	t.Run("requires callbacks after base validation", func(t *testing.T) {
		err := Run(context.Background(), RunOptions{
			Env:        newEnv(t.TempDir()),
			AddonsPath: t.TempDir(),
			Target:     "auth",
			RunBE:      true,
			Stdout:     io.Discard,
			Stderr:     io.Discard,
		})
		if err == nil || !strings.Contains(err.Error(), "missing required callbacks") {
			t.Fatalf("expected missing callbacks error, got %v", err)
		}
	})

	t.Run("validates scope and required fields", func(t *testing.T) {
		err := Run(context.Background(), RunOptions{RunBE: true, AddonsPath: t.TempDir(), Target: "auth"})
		if err == nil || !strings.Contains(err.Error(), "scope is not initialized") {
			t.Fatalf("expected missing scope error, got %v", err)
		}

		err = Run(context.Background(), RunOptions{Env: newEnv(t.TempDir()), RunBE: true, Target: "auth"})
		if err == nil || !strings.Contains(err.Error(), "config missing addons_path") {
			t.Fatalf("expected missing addons_path error, got %v", err)
		}

		err = Run(context.Background(), RunOptions{Env: newEnv(t.TempDir()), AddonsPath: t.TempDir(), RunBE: true})
		if err == nil || !strings.Contains(err.Error(), "missing app") {
			t.Fatalf("expected missing app error, got %v", err)
		}
	})

	t.Run("returns no tests found when nothing resolves", func(t *testing.T) {
		var stdout strings.Builder
		err := Run(context.Background(), RunOptions{
			Env:        newEnv(t.TempDir()),
			AddonsPath: t.TempDir(),
			Target:     "all",
			RunBE:      true,
			Stdout:     &stdout,
			Stderr:     io.Discard,
			ResolveApps: func(runtimeScope scope.Scope, arg string, runBE bool, runFE bool) ([]string, error) {
				return nil, nil
			},
			HasBackendTests:  func(addonsPath string, app string) (bool, error) { return false, nil },
			HasFrontendTests: func(addonsPath string, app string) (bool, error) { return false, nil },
			RunBackend: func(context.Context, scope.Scope, string, string, string, string, string, bool, string, string, bool, bool) (bool, error) {
				return false, nil
			},
			RunFrontend: func(context.Context, string, string, string, bool, bool, bool, bool, string, int, int, int, int) (bool, error) {
				return false, nil
			},
		})
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
		if !strings.Contains(stdout.String(), "no tests found") {
			t.Fatalf("expected no tests found message, got %q", stdout.String())
		}
	})

	t.Run("resolve apps error is propagated", func(t *testing.T) {
		resolveErr := errors.New("resolve failed")
		err := Run(context.Background(), RunOptions{
			Env:        newEnv(t.TempDir()),
			AddonsPath: t.TempDir(),
			Target:     "all",
			RunBE:      true,
			Stdout:     io.Discard,
			Stderr:     io.Discard,
			ResolveApps: func(runtimeScope scope.Scope, arg string, runBE bool, runFE bool) ([]string, error) {
				return nil, resolveErr
			},
			HasBackendTests:  func(addonsPath string, app string) (bool, error) { return false, nil },
			HasFrontendTests: func(addonsPath string, app string) (bool, error) { return false, nil },
			RunBackend: func(context.Context, scope.Scope, string, string, string, string, string, bool, string, string, bool, bool) (bool, error) {
				return false, nil
			},
			RunFrontend: func(context.Context, string, string, string, bool, bool, bool, bool, string, int, int, int, int) (bool, error) {
				return false, nil
			},
		})
		if !errors.Is(err, resolveErr) {
			t.Fatalf("expected resolve apps error, got %v", err)
		}
	})

	t.Run("fails when no tests found and fail if no tests is set", func(t *testing.T) {
		err := Run(context.Background(), RunOptions{
			Env:           newEnv(t.TempDir()),
			AddonsPath:    t.TempDir(),
			Target:        "all",
			RunBE:         true,
			FailIfNoTests: true,
			Stdout:        io.Discard,
			Stderr:        io.Discard,
			ResolveApps: func(runtimeScope scope.Scope, arg string, runBE bool, runFE bool) ([]string, error) {
				return nil, nil
			},
			HasBackendTests:  func(addonsPath string, app string) (bool, error) { return false, nil },
			HasFrontendTests: func(addonsPath string, app string) (bool, error) { return false, nil },
			RunBackend: func(context.Context, scope.Scope, string, string, string, string, string, bool, string, string, bool, bool) (bool, error) {
				return false, nil
			},
			RunFrontend: func(context.Context, string, string, string, bool, bool, bool, bool, string, int, int, int, int) (bool, error) {
				return false, nil
			},
		})
		if err == nil || !strings.Contains(err.Error(), "no tests found") {
			t.Fatalf("expected no tests found error, got %v", err)
		}
	})

	t.Run("returns context error after resolving apps", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		resolved := false
		err := Run(ctx, RunOptions{
			Env:        newEnv(t.TempDir()),
			AddonsPath: t.TempDir(),
			Target:     "auth",
			RunBE:      true,
			Stdout:     io.Discard,
			Stderr:     io.Discard,
			ResolveApps: func(runtimeScope scope.Scope, arg string, runBE bool, runFE bool) ([]string, error) {
				resolved = true
				cancel()
				return []string{"auth"}, nil
			},
			HasBackendTests:  func(addonsPath string, app string) (bool, error) { return true, nil },
			HasFrontendTests: func(addonsPath string, app string) (bool, error) { return false, nil },
			RunBackend: func(context.Context, scope.Scope, string, string, string, string, string, bool, string, string, bool, bool) (bool, error) {
				return false, nil
			},
			RunFrontend: func(context.Context, string, string, string, bool, bool, bool, bool, string, int, int, int, int) (bool, error) {
				return false, nil
			},
		})
		if !resolved {
			t.Fatal("expected resolve callback to run")
		}
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context canceled, got %v", err)
		}
	})

	t.Run("requires typecheck callback when requested", func(t *testing.T) {
		err := Run(context.Background(), RunOptions{
			Env:           newEnv(t.TempDir()),
			AddonsPath:    t.TempDir(),
			Target:        "auth",
			RunBE:         true,
			WithTypecheck: true,
			Stdout:        io.Discard,
			Stderr:        io.Discard,
			ResolveApps: func(runtimeScope scope.Scope, arg string, runBE bool, runFE bool) ([]string, error) {
				return []string{"auth"}, nil
			},
			HasBackendTests:  func(addonsPath string, app string) (bool, error) { return true, nil },
			HasFrontendTests: func(addonsPath string, app string) (bool, error) { return false, nil },
			RunBackend: func(context.Context, scope.Scope, string, string, string, string, string, bool, string, string, bool, bool) (bool, error) {
				return false, nil
			},
			RunFrontend: func(context.Context, string, string, string, bool, bool, bool, bool, string, int, int, int, int) (bool, error) {
				return false, nil
			},
		})
		if err == nil || !strings.Contains(err.Error(), "typecheck requested but callback missing") {
			t.Fatalf("expected missing typecheck callback error, got %v", err)
		}
	})

	t.Run("typecheck failure without fail fast is aggregated", func(t *testing.T) {
		var stderr strings.Builder
		err := Run(context.Background(), RunOptions{
			Env:           newEnv(t.TempDir()),
			AddonsPath:    t.TempDir(),
			Target:        "all",
			RunBE:         true,
			WithTypecheck: true,
			Stdout:        io.Discard,
			Stderr:        &stderr,
			ResolveApps: func(runtimeScope scope.Scope, arg string, runBE bool, runFE bool) ([]string, error) {
				return []string{"auth"}, nil
			},
			HasBackendTests:  func(addonsPath string, app string) (bool, error) { return true, nil },
			HasFrontendTests: func(addonsPath string, app string) (bool, error) { return false, nil },
			Typecheck: func(context.Context, scope.Scope, string, string) error {
				return errors.New("typecheck boom")
			},
			RunBackend: func(context.Context, scope.Scope, string, string, string, string, string, bool, string, string, bool, bool) (bool, error) {
				return false, nil
			},
			RunFrontend: func(context.Context, string, string, string, bool, bool, bool, bool, string, int, int, int, int) (bool, error) {
				return false, nil
			},
		})
		if err == nil || !strings.Contains(err.Error(), "tests failed") {
			t.Fatalf("expected aggregated test failure, got %v", err)
		}
		if !strings.Contains(stderr.String(), "typecheck boom") {
			t.Fatalf("expected typecheck error on stderr, got %q", stderr.String())
		}
	})

	t.Run("typecheck failure with fail fast returns immediately", func(t *testing.T) {
		typecheckErr := errors.New("typecheck stop now")
		err := Run(context.Background(), RunOptions{
			Env:           newEnv(t.TempDir()),
			AddonsPath:    t.TempDir(),
			Target:        "auth",
			RunBE:         true,
			WithTypecheck: true,
			FailFast:      true,
			Stdout:        io.Discard,
			Stderr:        io.Discard,
			ResolveApps: func(runtimeScope scope.Scope, arg string, runBE bool, runFE bool) ([]string, error) {
				return []string{"auth"}, nil
			},
			HasBackendTests:  func(addonsPath string, app string) (bool, error) { return true, nil },
			HasFrontendTests: func(addonsPath string, app string) (bool, error) { return false, nil },
			Typecheck: func(context.Context, scope.Scope, string, string) error {
				return typecheckErr
			},
			RunBackend: func(context.Context, scope.Scope, string, string, string, string, string, bool, string, string, bool, bool) (bool, error) {
				return false, nil
			},
			RunFrontend: func(context.Context, string, string, string, bool, bool, bool, bool, string, int, int, int, int) (bool, error) {
				return false, nil
			},
		})
		if !errors.Is(err, typecheckErr) {
			t.Fatalf("expected typecheck failfast error, got %v", err)
		}
	})

	t.Run("backend fail fast stops after first failing app", func(t *testing.T) {
		calls := 0
		err := Run(context.Background(), RunOptions{
			Env:        newEnv(t.TempDir()),
			AddonsPath: t.TempDir(),
			Target:     "all",
			RunBE:      true,
			FailFast:   true,
			Stdout:     io.Discard,
			Stderr:     io.Discard,
			ResolveApps: func(runtimeScope scope.Scope, arg string, runBE bool, runFE bool) ([]string, error) {
				return []string{"auth", "partner"}, nil
			},
			HasBackendTests:  func(addonsPath string, app string) (bool, error) { return true, nil },
			HasFrontendTests: func(addonsPath string, app string) (bool, error) { return false, nil },
			RunBackend: func(context.Context, scope.Scope, string, string, string, string, string, bool, string, string, bool, bool) (bool, error) {
				calls++
				return true, nil
			},
			RunFrontend: func(context.Context, string, string, string, bool, bool, bool, bool, string, int, int, int, int) (bool, error) {
				return false, nil
			},
		})
		if err == nil || !strings.Contains(err.Error(), "tests failed") {
			t.Fatalf("expected tests failed error, got %v", err)
		}
		if calls != 1 {
			t.Fatalf("expected one backend call before fail-fast break, got %d", calls)
		}
	})

	t.Run("propagates keep option to backend runner", func(t *testing.T) {
		keepSeen := false
		err := Run(context.Background(), RunOptions{
			Env:        newEnv(t.TempDir()),
			AddonsPath: t.TempDir(),
			Target:     "auth",
			RunBE:      true,
			Keep:       true,
			Stdout:     io.Discard,
			Stderr:     io.Discard,
			ResolveApps: func(runtimeScope scope.Scope, arg string, runBE bool, runFE bool) ([]string, error) {
				return []string{"auth"}, nil
			},
			HasBackendTests:  func(addonsPath string, app string) (bool, error) { return true, nil },
			HasFrontendTests: func(addonsPath string, app string) (bool, error) { return false, nil },
			RunBackend: func(_ context.Context, _ scope.Scope, _ string, _ string, _ string, _ string, _ string, keep bool, _ string, _ string, _ bool, _ bool) (bool, error) {
				keepSeen = keep
				return false, nil
			},
			RunFrontend: func(context.Context, string, string, string, bool, bool, bool, bool, string, int, int, int, int) (bool, error) {
				return false, nil
			},
		})
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
		if !keepSeen {
			t.Fatalf("expected backend runner to receive keep=true")
		}
	})

	t.Run("backend and frontend runner errors are propagated", func(t *testing.T) {
		backendErr := errors.New("backend exploded")
		err := Run(context.Background(), RunOptions{
			Env:        newEnv(t.TempDir()),
			AddonsPath: t.TempDir(),
			Target:     "auth",
			RunBE:      true,
			Stdout:     io.Discard,
			Stderr:     io.Discard,
			ResolveApps: func(runtimeScope scope.Scope, arg string, runBE bool, runFE bool) ([]string, error) {
				return []string{"auth"}, nil
			},
			HasBackendTests:  func(addonsPath string, app string) (bool, error) { return true, nil },
			HasFrontendTests: func(addonsPath string, app string) (bool, error) { return false, nil },
			RunBackend: func(context.Context, scope.Scope, string, string, string, string, string, bool, string, string, bool, bool) (bool, error) {
				return false, backendErr
			},
			RunFrontend: func(context.Context, string, string, string, bool, bool, bool, bool, string, int, int, int, int) (bool, error) {
				return false, nil
			},
		})
		if !errors.Is(err, backendErr) {
			t.Fatalf("expected backend error, got %v", err)
		}

		frontendErr := errors.New("frontend exploded")
		err = Run(context.Background(), RunOptions{
			Env:        newEnv(t.TempDir()),
			AddonsPath: t.TempDir(),
			Target:     "auth",
			RunFE:      true,
			Stdout:     io.Discard,
			Stderr:     io.Discard,
			ResolveApps: func(runtimeScope scope.Scope, arg string, runBE bool, runFE bool) ([]string, error) {
				return []string{"auth"}, nil
			},
			HasBackendTests:  func(addonsPath string, app string) (bool, error) { return false, nil },
			HasFrontendTests: func(addonsPath string, app string) (bool, error) { return true, nil },
			RunBackend: func(context.Context, scope.Scope, string, string, string, string, string, bool, string, string, bool, bool) (bool, error) {
				return false, nil
			},
			RunFrontend: func(context.Context, string, string, string, bool, bool, bool, bool, string, int, int, int, int) (bool, error) {
				return false, frontendErr
			},
		})
		if !errors.Is(err, frontendErr) {
			t.Fatalf("expected frontend error, got %v", err)
		}
	})

	t.Run("test discovery callback errors are propagated", func(t *testing.T) {
		hasBEErr := errors.New("has backend tests failed")
		err := Run(context.Background(), RunOptions{
			Env:        newEnv(t.TempDir()),
			AddonsPath: t.TempDir(),
			Target:     "auth",
			RunBE:      true,
			Stdout:     io.Discard,
			Stderr:     io.Discard,
			ResolveApps: func(runtimeScope scope.Scope, arg string, runBE bool, runFE bool) ([]string, error) {
				return []string{"auth"}, nil
			},
			HasBackendTests: func(addonsPath string, app string) (bool, error) {
				return false, hasBEErr
			},
			HasFrontendTests: func(addonsPath string, app string) (bool, error) { return false, nil },
			RunBackend: func(context.Context, scope.Scope, string, string, string, string, string, bool, string, string, bool, bool) (bool, error) {
				return false, nil
			},
			RunFrontend: func(context.Context, string, string, string, bool, bool, bool, bool, string, int, int, int, int) (bool, error) {
				return false, nil
			},
		})
		if !errors.Is(err, hasBEErr) {
			t.Fatalf("expected has backend tests error, got %v", err)
		}

		hasFEErr := errors.New("has frontend tests failed")
		err = Run(context.Background(), RunOptions{
			Env:        newEnv(t.TempDir()),
			AddonsPath: t.TempDir(),
			Target:     "auth",
			RunFE:      true,
			Stdout:     io.Discard,
			Stderr:     io.Discard,
			ResolveApps: func(runtimeScope scope.Scope, arg string, runBE bool, runFE bool) ([]string, error) {
				return []string{"auth"}, nil
			},
			HasBackendTests:  func(addonsPath string, app string) (bool, error) { return false, nil },
			HasFrontendTests: func(addonsPath string, app string) (bool, error) { return false, hasFEErr },
			RunBackend: func(context.Context, scope.Scope, string, string, string, string, string, bool, string, string, bool, bool) (bool, error) {
				return false, nil
			},
			RunFrontend: func(context.Context, string, string, string, bool, bool, bool, bool, string, int, int, int, int) (bool, error) {
				return false, nil
			},
		})
		if !errors.Is(err, hasFEErr) {
			t.Fatalf("expected has frontend tests error, got %v", err)
		}
	})

	t.Run("coverage report and check branches return errors when tooling missing", func(t *testing.T) {
		t.Setenv("PATH", "")
		err := Run(context.Background(), RunOptions{
			Env:               newEnv(t.TempDir()),
			AddonsPath:        t.TempDir(),
			Target:            "auth",
			RepoRoot:          t.TempDir(),
			RunBE:             true,
			Coverage:          true,
			CoverageReport:    true,
			CoverageReporters: []string{"text"},
			Stdout:            io.Discard,
			Stderr:            io.Discard,
			ResolveApps: func(runtimeScope scope.Scope, arg string, runBE bool, runFE bool) ([]string, error) {
				return []string{"auth"}, nil
			},
			HasBackendTests:  func(addonsPath string, app string) (bool, error) { return true, nil },
			HasFrontendTests: func(addonsPath string, app string) (bool, error) { return false, nil },
			RunBackend: func(context.Context, scope.Scope, string, string, string, string, string, bool, string, string, bool, bool) (bool, error) {
				return false, nil
			},
			RunFrontend: func(context.Context, string, string, string, bool, bool, bool, bool, string, int, int, int, int) (bool, error) {
				return false, nil
			},
		})
		if err == nil || !strings.Contains(err.Error(), "--coverage-report requires node") {
			t.Fatalf("expected coverage report tooling error, got %v", err)
		}

		err = Run(context.Background(), RunOptions{
			Env:           newEnv(t.TempDir()),
			AddonsPath:    t.TempDir(),
			Target:        "auth",
			RepoRoot:      t.TempDir(),
			RunBE:         true,
			Coverage:      true,
			CoverageCheck: true,
			CoverageLines: 80,
			Stdout:        io.Discard,
			Stderr:        io.Discard,
			ResolveApps: func(runtimeScope scope.Scope, arg string, runBE bool, runFE bool) ([]string, error) {
				return []string{"auth"}, nil
			},
			HasBackendTests:  func(addonsPath string, app string) (bool, error) { return true, nil },
			HasFrontendTests: func(addonsPath string, app string) (bool, error) { return false, nil },
			RunBackend: func(context.Context, scope.Scope, string, string, string, string, string, bool, string, string, bool, bool) (bool, error) {
				return false, nil
			},
			RunFrontend: func(context.Context, string, string, string, bool, bool, bool, bool, string, int, int, int, int) (bool, error) {
				return false, nil
			},
		})
		if err == nil || !strings.Contains(err.Error(), "--coverage-check requires node") {
			t.Fatalf("expected coverage check tooling error, got %v", err)
		}
	})
}

func TestRunWithDefaults(t *testing.T) {
	err := RunWithDefaults(context.Background(), RunOptions{
		RunBE:      true,
		AddonsPath: t.TempDir(),
		Target:     "auth",
		Stdout:     io.Discard,
		Stderr:     io.Discard,
	})
	if err == nil || !strings.Contains(err.Error(), "scope is not initialized") {
		t.Fatalf("expected scope error, got %v", err)
	}
}

func TestRunWithDefaultsUsesInjectedTypecheck(t *testing.T) {
	err := RunWithDefaults(context.Background(), RunOptions{
		Env:           nil,
		RunBE:         true,
		WithTypecheck: true,
		AddonsPath:    t.TempDir(),
		Target:        "auth",
		Stdout:        io.Discard,
		Stderr:        io.Discard,
	})
	if err == nil || !strings.Contains(err.Error(), "scope is not initialized") {
		t.Fatalf("expected scope error, got %v", err)
	}

	// Ensure os import remains used in tests across toolchains.
	_ = os.PathSeparator
}

func TestRunWithDefaultsPropagatesTmpPathToTypecheck(t *testing.T) {
	repoRoot := t.TempDir()
	addonsPath := filepath.Join(repoRoot, "addons")
	if err := os.MkdirAll(filepath.Join(addonsPath, "auth", "service"), 0o755); err != nil {
		t.Fatalf("MkdirAll auth service: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repoRoot, "node_modules", "vue-tsc"), 0o755); err != nil {
		t.Fatalf("MkdirAll vue-tsc: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "node_modules", "vue-tsc", "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("WriteFile vue-tsc package.json: %v", err)
	}

	tmpRoot := filepath.Join(t.TempDir(), "tmp-root")
	binDir := t.TempDir()
	npmPath := filepath.Join(binDir, "npm")
	if err := os.WriteFile(npmPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("WriteFile npm stub: %v", err)
	}
	capturePath := filepath.Join(t.TempDir(), "captured-tsconfig-path.txt")
	npxPath := filepath.Join(binDir, "npx")
	npxScript := "#!/bin/sh\nprev=\"\"\nfor arg in \"$@\"; do\n  if [ \"$prev\" = \"-p\" ]; then\n    printf '%s' \"$arg\" > \"$CHOYSUM_CAPTURE_TSCONFIG_PATH\"\n    break\n  fi\n  prev=\"$arg\"\ndone\nexit 0\n"
	if err := os.WriteFile(npxPath, []byte(npxScript), 0o755); err != nil {
		t.Fatalf("WriteFile npx stub: %v", err)
	}
	t.Setenv("CHOYSUM_CAPTURE_TSCONFIG_PATH", capturePath)

	runtimeScope := &testStubScope{ctx: context.Background(), cfg: &config.Config{AddonsPath: addonsPath, NpmPath: npmPath, TmpPath: tmpRoot}}
	err := RunWithDefaults(context.Background(), RunOptions{
		Env:           runtimeScope,
		AddonsPath:    addonsPath,
		RepoRoot:      repoRoot,
		Target:        "auth",
		RunBE:         true,
		WithTypecheck: true,
		Stdout:        io.Discard,
		Stderr:        io.Discard,
		ResolveApps: func(runtimeScope scope.Scope, arg string, runBE bool, runFE bool) ([]string, error) {
			return []string{"auth"}, nil
		},
		HasBackendTests:  func(addonsPath string, app string) (bool, error) { return true, nil },
		HasFrontendTests: func(addonsPath string, app string) (bool, error) { return false, nil },
		RunBackend: func(context.Context, scope.Scope, string, string, string, string, string, bool, string, string, bool, bool) (bool, error) {
			return false, nil
		},
		RunFrontend: func(context.Context, string, string, string, bool, bool, bool, bool, string, int, int, int, int) (bool, error) {
			return false, nil
		},
	})
	if err != nil {
		t.Fatalf("RunWithDefaults: %v", err)
	}

	captured, err := os.ReadFile(capturePath)
	if err != nil {
		t.Fatalf("ReadFile captured tsconfig path: %v", err)
	}
	tsconfigPath := strings.TrimSpace(string(captured))
	if tsconfigPath == "" {
		t.Fatalf("expected captured tsconfig path")
	}

	typecheckLegacyTmpDir, err := testingpathing.ResolveTestingTmpDir(repoRoot, tmpRoot, "typecheck")
	if err != nil {
		t.Fatalf("ResolveTestingTmpDir(typecheck): %v", err)
	}
	workspaceTestingRoot := filepath.Dir(typecheckLegacyTmpDir)
	if filepath.Base(tsconfigPath) == "" {
		t.Fatalf("expected non-empty tsconfig basename")
	}
	gotTmpDir := filepath.Dir(tsconfigPath)
	if filepath.Base(gotTmpDir) != "auth" {
		t.Fatalf("expected typecheck app leaf dir, got %q", gotTmpDir)
	}
	if filepath.Base(filepath.Dir(gotTmpDir)) != "typecheck" {
		t.Fatalf("expected typecheck parent dir, got %q", filepath.Dir(gotTmpDir))
	}
	if !strings.HasPrefix(filepath.Clean(gotTmpDir), filepath.Clean(workspaceTestingRoot)+string(filepath.Separator)) {
		t.Fatalf("typecheck tmp dir = %q, want prefix %q", gotTmpDir, workspaceTestingRoot)
	}
	runID := filepath.Base(filepath.Dir(filepath.Dir(gotTmpDir)))
	if strings.TrimSpace(runID) == "" || runID == "testing" || runID == filepath.Base(workspaceTestingRoot) {
		t.Fatalf("expected non-empty run-id segment in typecheck tmp dir, got %q", runID)
	}
}
