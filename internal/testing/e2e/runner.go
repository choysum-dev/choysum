// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package e2e

import (
	"bufio"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	metadata "github.com/choysum-dev/choysum/internal/module/metadata"

	"github.com/choysum-dev/choysum/internal/config/snapshot"
	_ "github.com/choysum-dev/choysum/internal/defaultengine"
	_ "github.com/choysum-dev/choysum/internal/defaultjsexecutor"
	_ "github.com/choysum-dev/choysum/internal/defaultscope"
	"github.com/choysum-dev/choysum/internal/logger"
	dataloader "github.com/choysum-dev/choysum/internal/module/evolution/data"
	"github.com/choysum-dev/choysum/internal/module/lifecycle"
	testingpathing "github.com/choysum-dev/choysum/internal/testing/tmpdir"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/jsexecutor"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
	"gorm.io/datatypes"
	"gorm.io/gorm/clause"
)

type RunOptions struct {
	ModulesPath string
	NpmPath     string
	TmpPath     string
	// ChoysumBinaryPath, when set, is used to start `choysum run`.
	// When empty, the runner will default to os.Args[0] (works when invoked from the choysum CLI binary).
	ChoysumBinaryPath string

	Module    string
	Scenarios []string
	WithDemo  bool
	Keep      bool
	Timeout   time.Duration

	StartupTimeout  time.Duration
	Port            int
	Verbose         bool
	RuntimeLogLevel string

	PlaywrightArgs []string
	WorkDir        string

	Stdout io.Writer
	Stderr io.Writer
}

type runtimeInfo struct {
	PID        int      `json:"pid"`
	Port       int      `json:"port"`
	BaseURL    string   `json:"baseURL"`
	ConfigPath string   `json:"configPath"`
	DBPath     string   `json:"dbPath"`
	Module     string   `json:"module"`
	Scenario   string   `json:"scenario"`
	SpecsDir   string   `json:"specsDir"`
	Fixtures   []string `json:"fixtures,omitempty"`
}

var scenarioNameRE = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)

var (
	installForE2EHook         = installForE2E
	applyScenarioFixturesHook = applyScenarioFixtures
	seedModuleIndexHook       = seedModuleIndexForE2E
	startServerHook           = startServer
	stopServerHook            = stopServer
	waitForHTTP200Hook        = waitForHTTP200
	runPlaywrightHook         = runPlaywright
	runOneScenarioHook        = runOneScenario
)

type e2eRuntimeOptions struct {
	modulesPath string
}

const e2EConfigEnvPrefix = "CHOYSUM_E2E_CONFIG"

func writeE2EProgress(w io.Writer, format string, args ...any) {
	if w == nil {
		return
	}
	_, _ = fmt.Fprintf(w, format, args...)
}

func newE2ERuntimeOptions(pathOpts scope.PathsRuntimeOptions, hasPathOpts bool) e2eRuntimeOptions {
	if !hasPathOpts {
		return e2eRuntimeOptions{}
	}
	return e2eRuntimeOptions{modulesPath: pathOpts.ModulesPath}
}

func (o e2eRuntimeOptions) Validate() error {
	if strings.TrimSpace(o.modulesPath) == "" {
		return xfmt.Errorf("e2e runtime options: modulesPath is required")
	}
	return nil
}

func newE2ERuntimeScope(ctx context.Context, configPath string) (scope.Scope, e2eRuntimeOptions, error) {
	cfg, err := config.LoadWithProvider(nil, configPath, config.WithEnvPrefix(e2EConfigEnvPrefix))
	if err != nil {
		return nil, e2eRuntimeOptions{}, err
	}
	cfgSnapshot := snapshot.New(cfg)

	runtimeInput := newRuntimeScopeInput(cfgSnapshot, e2eRuntimeOptions{})
	pathOpts, hasPathOpts := scope.PathsRuntimeOptionsFromInput(runtimeInput)
	runtimeOptions := newE2ERuntimeOptions(pathOpts, hasPathOpts)
	if err := runtimeOptions.Validate(); err != nil {
		return nil, e2eRuntimeOptions{}, err
	}

	l := logger.NewLogger(cfg.Log)
	runtimeScope := scope.NewScope(ctx, newRuntimeScopeInput(cfgSnapshot, runtimeOptions), l)
	if runtimeScope == nil {
		return nil, e2eRuntimeOptions{}, xfmt.Errorf("failed to initialize scope")
	}

	return runtimeScope, runtimeOptions, nil
}

func filteredE2EEnv(baseEnv []string) []string {
	if len(baseEnv) == 0 {
		return nil
	}
	filtered := make([]string, 0, len(baseEnv))
	for _, entry := range baseEnv {
		key, _, found := strings.Cut(entry, "=")
		if !found {
			filtered = append(filtered, entry)
			continue
		}
		if strings.HasPrefix(key, "CHOYSUM_") && !strings.HasPrefix(key, "CHOYSUM_E2E_") {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}

func RunModule(ctx context.Context, opts RunOptions) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(testingpathing.TestingRunIDFromContext(ctx)) == "" {
		ctx = testingpathing.ContextWithTestingRunID(ctx, testingpathing.NewTestingRunID())
	}
	if strings.TrimSpace(opts.Module) == "" {
		return xfmt.Errorf("module is required")
	}
	if strings.TrimSpace(opts.ModulesPath) == "" {
		return xfmt.Errorf("modules_path is required")
	}
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	if opts.WorkDir == "" {
		wd, _ := os.Getwd()
		opts.WorkDir = wd
	}
	if opts.StartupTimeout <= 0 {
		opts.StartupTimeout = 3 * time.Minute
	}

	packages, err := discoverSourcePackages(opts.ModulesPath)
	if err != nil {
		return err
	}
	targetPackage, ok := packages[opts.Module]
	if !ok {
		return xfmt.Errorf("unknown module %q (no package.json under %s)", opts.Module, opts.ModulesPath)
	}
	if targetPackage.E2E == nil || strings.TrimSpace(targetPackage.E2E.Specs) == "" {
		return xfmt.Errorf("module %q has no package.json choysum.e2e.specs", opts.Module)
	}

	scenarioList := opts.Scenarios
	if len(scenarioList) == 0 {
		scenarioList = []string{"default"}
	}
	for _, s := range scenarioList {
		if strings.TrimSpace(s) == "" {
			return xfmt.Errorf("scenario name cannot be empty")
		}
		if !scenarioNameRE.MatchString(s) {
			return xfmt.Errorf("invalid scenario %q (must match %s)", s, scenarioNameRE.String())
		}
	}

	for _, scenario := range scenarioList {
		if opts.Timeout > 0 {
			perRunCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
			err := runOneScenarioHook(perRunCtx, opts, packages, scenario)
			cancel()
			if err != nil {
				return err
			}
			continue
		}
		if err := runOneScenarioHook(ctx, opts, packages, scenario); err != nil {
			return err
		}
	}
	return nil
}

// ResolveE2EModules returns module directory names that are runnable by `choysum test e2e`,
// i.e. modules that have a package.json with a non-empty choysum.e2e.specs.
func ResolveE2EModules(modulesPath string) ([]string, error) {
	if strings.TrimSpace(modulesPath) == "" {
		return nil, xfmt.Errorf("modules_path is required")
	}
	packages, err := discoverSourcePackages(modulesPath)
	if err != nil {
		return nil, err
	}
	mods := make([]string, 0)
	for name, p := range packages {
		if p == nil || p.E2E == nil || strings.TrimSpace(p.E2E.Specs) == "" {
			continue
		}
		mods = append(mods, name)
	}
	sort.Strings(mods)
	return mods, nil
}

func runOneScenario(ctx context.Context, opts RunOptions, packages map[string]*sourceModulePackage, scenario string) error {
	prepareStarted := time.Now()
	writeE2EProgress(opts.Stderr, "# prepare runtime %s\n", opts.Module)

	closure, err := topoClosure(opts.Module, packages)
	if err != nil {
		return err
	}

	workspaceTmpDir, err := testingpathing.ResolveTestingTmpDirFromContext(ctx, opts.WorkDir, opts.TmpPath, "e2e")
	if err != nil {
		return xfmt.Errorf("resolve e2e tmp dir: %w", err)
	}
	rootTmp := workspaceTmpDir
	if err := os.MkdirAll(rootTmp, 0o755); err != nil {
		return xfmt.Errorf("create tmp dir: %w", err)
	}
	runID := randHex(3)
	runDir := filepath.Join(rootTmp, opts.Module, scenario+"-"+runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return xfmt.Errorf("create run dir: %w", err)
	}
	cleanup := func() {
		if opts.Keep {
			return
		}
		_ = os.RemoveAll(runDir)
	}
	defer cleanup()

	port := opts.Port
	if port == 0 {
		port, err = pickFreePort()
		if err != nil {
			return err
		}
	}

	resolvedRuntimeLogLevel, err := resolveRuntimeLogLevel(opts.RuntimeLogLevel, opts.Verbose)
	if err != nil {
		return err
	}

	logPath := filepath.Join(runDir, "server.log")
	runtimePath := filepath.Join(runDir, "runtime.json")
	configPath := filepath.Join(runDir, "config.yaml")
	dbPath := filepath.Join(runDir, "db.sqlite")
	distDir := filepath.Join(runDir, "dist")
	defaultChoysumPath := filepath.Join(runDir, ".choysum")
	jwtKeysDir := filepath.Join(runDir, "jwtkeys")
	privateKeyPath := filepath.Join(jwtKeysDir, "private.pem")
	publicKeyPath := filepath.Join(jwtKeysDir, "public.pem")
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		return xfmt.Errorf("create dist dir: %w", err)
	}
	if err := os.MkdirAll(jwtKeysDir, 0o755); err != nil {
		return xfmt.Errorf("create jwt keys dir: %w", err)
	}

	sqliteDSN := fmt.Sprintf("file:%s?mode=rwc&_fk=1&_busy_timeout=60000&_journal_mode=WAL", dbPath)
	npmPath := opts.NpmPath
	if strings.TrimSpace(npmPath) == "" {
		// Best effort: prefer repo-local node_modules.
		candidate := filepath.Join(opts.WorkDir, "node_modules")
		if st, err := os.Stat(candidate); err == nil && st.IsDir() {
			npmPath = candidate
		}
	}

	authEnabled := false
	authInClosure := false
	metaInClosure := false
	for _, mod := range closure {
		switch strings.TrimSpace(mod) {
		case "auth":
			authEnabled = true
			authInClosure = true
		case "meta":
			metaInClosure = true
		}
	}
	if opts.Module == "meta" {
		authEnabled = true
	}
	if packages["auth"] != nil {
		authEnabled = true
	}
	internalKey := randHex(16)
	authBlock := fmt.Sprintf("auth:\n  enabled: true\n  jwt:\n    revokeStore: \"database\"\n    privateKeyFile: %q\n    publicKeyFile: %q\n  internalKey: %q\n", privateKeyPath, publicKeyPath, internalKey)
	if !authEnabled {
		authBlock = fmt.Sprintf("auth:\n  enabled: false\n  grpcAuthentication: false\n  grpcMethodAccess: false\n  grpcRecordRule: false\n  grpcCompanyFilter: false\n  grpcFieldRule: false\n  jwt:\n    revokeStore: \"database\"\n    privateKeyFile: %q\n    publicKeyFile: %q\n", privateKeyPath, publicKeyPath)
	}

	extraBackendEnv := ""
	switch strings.TrimSpace(strings.ToLower(scenario)) {
	case "lock-conflict":
		extraBackendEnv = "  CHOYSUM_E2E_FORCE_LOCK_CONFLICT: \"true\"\n"
	case "result-failed":
		extraBackendEnv = "  CHOYSUM_E2E_FORCE_RESULT_STATUS: \"FAILED\"\n"
	case "reload-failed":
		extraBackendEnv = "  CHOYSUM_E2E_FORCE_RELOAD_FAILED: \"true\"\n"
	}

	configYAML := fmt.Sprintf(`# Auto-generated by choysum test e2e
default_choysum_path: %q
modules_path: %q
dist_path: %q
npm_path: %q

log:
  level: %q

db:
  dialect: "sqlite"
  dsn: %q

%s
backendEnv:
	CHOYSUM_E2E_OPERATOR_USER_ID: "e2e-admin"
	CHOYSUM_E2E_SKIP_RELOAD: "true"
%s
task:
	dispatch:
		max_concurrency: 1
		fetch_batch_size: 1
		poll_interval_ms: 1000
server:
  bindAddress: "127.0.0.1"
  port: %d
  hotReload: false

compile:
  production: true
  minify: false
  treeShaking: true
  sourcemap: false
`, defaultChoysumPath, opts.ModulesPath, distDir, npmPath, resolvedRuntimeLogLevel, sqliteDSN, authBlock, extraBackendEnv, port)
	configYAML = strings.ReplaceAll(configYAML, "\t", "  ")

	if err := os.WriteFile(configPath, []byte(configYAML), 0o644); err != nil {
		return xfmt.Errorf("write config: %w", err)
	}

	// Prepare specs dir for target module.
	targetPackage := packages[opts.Module]
	if targetPackage.E2E == nil || strings.TrimSpace(targetPackage.E2E.Specs) == "" {
		return xfmt.Errorf("module %q has no package.json choysum.e2e.specs", opts.Module)
	}
	specsRel := filepath.Clean(strings.TrimSpace(targetPackage.E2E.Specs))
	if specsRel == "." || filepath.IsAbs(specsRel) || specsRel == ".." || strings.HasPrefix(specsRel, ".."+string(filepath.Separator)) {
		return xfmt.Errorf("invalid package.json choysum.e2e.specs for %q: %q", opts.Module, targetPackage.E2E.Specs)
	}
	specsDir := filepath.Join(opts.ModulesPath, targetPackage.DirName, specsRel)

	// Install dependency closure by installing the target module (planner handles depends).
	if err := installForE2EHook(ctx, configPath, opts.Module, opts.WithDemo); err != nil {
		return err
	}
	// Module management E2E relies on task job tables; install task module when testing meta.
	if opts.Module == "meta" {
		if err := installForE2EHook(ctx, configPath, "task", opts.WithDemo); err != nil {
			return err
		}
		if err := installForE2EHook(ctx, configPath, "auth", opts.WithDemo); err != nil {
			return err
		}
	}
	// Auth E2E needs meta services for permission bootstrap (IrModel/IrService lookups).
	if opts.Module == "auth" {
		if err := installForE2EHook(ctx, configPath, "meta", opts.WithDemo); err != nil {
			return err
		}
	}
	if authEnabled && !authInClosure && opts.Module != "auth" {
		if err := installForE2EHook(ctx, configPath, "auth", opts.WithDemo); err != nil {
			return err
		}
	}
	// If auth is in the dependency closure but meta is not, we still need meta services for auth bootstrapping.
	if authEnabled && !metaInClosure && opts.Module != "auth" {
		if err := installForE2EHook(ctx, configPath, "meta", opts.WithDemo); err != nil {
			return err
		}
	}

	// Apply fixtures for closure (each module may contribute its own fixtures for this scenario).
	fixtureClosure := append([]string{}, closure...)
	if authEnabled {
		fixtureClosure = append(fixtureClosure, "auth")
	}
	uniqueFixtures := make([]string, 0, len(fixtureClosure))
	seen := map[string]bool{}
	for _, mod := range fixtureClosure {
		if mod == "" || seen[mod] {
			continue
		}
		seen[mod] = true
		uniqueFixtures = append(uniqueFixtures, mod)
	}
	loadedFixtures := []string{}
	if err := applyScenarioFixturesHook(ctx, configPath, uniqueFixtures, packages, scenario, opts.Module, opts.Verbose, opts.Stderr, &loadedFixtures); err != nil {
		return err
	}
	if err := seedModuleIndexHook(ctx, configPath, packages); err != nil {
		return err
	}

	// Start server.
	serverCmd, err := startServerHook(opts.WorkDir, configPath, logPath, opts.ChoysumBinaryPath)
	if err != nil {
		return err
	}
	defer stopServerHook(serverCmd)

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	writeRuntime(runtimePath, runtimeInfo{PID: serverCmd.Process.Pid, Port: port, BaseURL: baseURL, ConfigPath: configPath, DBPath: dbPath, Module: opts.Module, Scenario: scenario, SpecsDir: specsDir, Fixtures: loadedFixtures})

	if err := waitForHTTP200Hook(ctx, baseURL+"/readyz", opts.StartupTimeout); err != nil {
		return includeLogTail(err, logPath)
	}
	writeE2EProgress(opts.Stderr, "# prepare runtime %s ok (%s)\n", opts.Module, time.Since(prepareStarted).Round(100*time.Millisecond))

	// Run Playwright (no global setup; Go has already orchestrated env).
	// Ensure the resolved npmPath is available to the Playwright runner.
	opts2 := opts
	opts2.NpmPath = npmPath
	if err := runPlaywrightHook(ctx, opts2, specsDir, baseURL, runtimePath); err != nil {
		return err
	}

	if opts.Keep {
		fmt.Fprintf(opts.Stderr, "[choysum test e2e] kept run dir: %s\n", runDir)
	}
	return nil
}

func installForE2E(ctx context.Context, configPath string, moduleName string, withDemo bool) error {
	runtimeScope, _, err := newE2ERuntimeScope(ctx, configPath)
	if err != nil {
		return xfmt.Errorf("load temp config: %w", err)
	}

	// Let module installation manage its own transaction/lease lifecycle.
	compilerExecutor, err := jsexecutor.NewCompilerExecutor(runtimeScope)
	if err != nil {
		return xfmt.Errorf("create compiler executor: %w", err)
	}
	if err := compilerExecutor.Start(); err != nil {
		return xfmt.Errorf("start compiler executor: %w", err)
	}
	defer compilerExecutor.Stop()

	moduleLifecycle := lifecycle.NewService(runtimeScope, compilerExecutor)
	return moduleLifecycle.Install(ctx, lifecycle.InstallRequest{Name: strings.TrimSpace(moduleName), WithDemo: withDemo})
}

func applyScenarioFixtures(
	ctx context.Context,
	configPath string,
	closure []string,
	packages map[string]*sourceModulePackage,
	scenario string,
	targetModule string,
	verbose bool,
	stderr io.Writer,
	loadedFixtures *[]string,
) error {
	runtimeScope, runtimeOptions, err := newE2ERuntimeScope(ctx, configPath)
	if err != nil {
		return xfmt.Errorf("load temp config: %w", err)
	}
	if stderr == nil {
		stderr = os.Stderr
	}

	for _, modName := range closure {
		sm := packages[modName]
		paths, defined, err := resolveScenarioFixtures(sm, scenario)
		if err != nil {
			return xfmt.Errorf("resolve fixtures: module=%s scenario=%s: %w", modName, scenario, err)
		}
		if !defined {
			// Scenario not defined for this module: allowed (skip fixtures).
			// Per docs (6.2), we should make this visible for the target module.
			if modName == targetModule {
				fmt.Fprintf(stderr, "[choysum test e2e] no fixtures loaded for module=%s scenario=%s (scenario not defined)\n", modName, scenario)
			} else if verbose {
				// Keep dependency-module noise behind --verbose.
				fmt.Fprintf(stderr, "[choysum test e2e] module=%s scenario=%s not defined (fixtures skipped)\n", modName, scenario)
			}
			continue
		}
		if len(paths) == 0 {
			// Scenario defined but empty: allowed.
			if modName == targetModule {
				fmt.Fprintf(stderr, "[choysum test e2e] module=%s scenario=%s defined but has no fixtures\n", modName, scenario)
			} else if verbose {
				fmt.Fprintf(stderr, "[choysum test e2e] module=%s scenario=%s defined but has no fixtures (skipped)\n", modName, scenario)
			}
			continue
		}
		// Apply in its own transaction to reduce lock contention.
		if err := runtimeScope.Transactor().Required(ctx, func(txScope scope.Scope, _ scope.Transaction) error {
			loader := dataloader.New(txScope)
			owner := &meta.IrModule{Name: modName, Path: filepath.Join(runtimeOptions.modulesPath, sm.DirName)}
			return loader.ApplyFiles(ctx, owner, paths)
		}); err != nil {
			return err
		}
		for _, p := range paths {
			*loadedFixtures = append(*loadedFixtures, filepath.Join(sm.DirName, p))
		}
	}
	return nil
}

func seedModuleIndexForE2E(ctx context.Context, configPath string, packages map[string]*sourceModulePackage) error {
	runtimeScope, runtimeOptions, err := newE2ERuntimeScope(ctx, configPath)
	if err != nil {
		return xfmt.Errorf("load temp config: %w", err)
	}

	return runtimeScope.Transactor().Required(ctx, func(txScope scope.Scope, _ scope.Transaction) error {
		sess := txScope.Session()
		if sess == nil || sess.DB == nil {
			return xfmt.Errorf("missing db session")
		}
		if err := sess.DB.AutoMigrate(&metadata.IrModuleIndex{}); err != nil {
			return xfmt.Errorf("auto-migrate metadata.IrModuleIndex: %w", err)
		}

		now := time.Now()
		for name, sm := range packages {
			if sm == nil || strings.TrimSpace(sm.DirName) == "" {
				continue
			}
			if shouldSkipModuleDir(name) {
				continue
			}
			packageJSONPath := filepath.Join(runtimeOptions.modulesPath, sm.DirName, "package.json")
			info, err := os.Stat(packageJSONPath)
			if err != nil {
				continue
			}
			data, version, err := readPackageVersion(packageJSONPath)
			if err != nil {
				continue
			}
			revision := fmt.Sprintf("%d:%d", info.ModTime().UnixNano(), info.Size())
			localPath := filepath.Join(runtimeOptions.modulesPath, sm.DirName)

			rec := metadata.IrModuleIndex{
				ModuleName:      name,
				OriginType:      "local",
				OriginRef:       "local",
				Available:       true,
				Version:         sql.NullString{String: version, Valid: version != ""},
				ManifestJson:    datatypes.JSON(data),
				LocalPath:       sql.NullString{String: localPath, Valid: localPath != ""},
				LastSyncAt:      &now,
				LastBatchSyncAt: &now,
				SyncRevision:    sql.NullString{String: revision, Valid: revision != ""},
			}

			if err := sess.DB.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "module_name"}, {Name: "origin_type"}, {Name: "origin_ref"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"available",
					"version",
					"manifest_json",
					"local_path",
					"last_sync_at",
					"last_batch_sync_at",
					"sync_revision",
					"last_error_message",
					"updated_at",
					"deleted_at",
				}),
			}).Create(&rec).Error; err != nil {
				return xfmt.Errorf("upsert module index: %w", err)
			}
		}
		return nil
	})
}

func shouldSkipModuleDir(name string) bool {
	if name == "" {
		return true
	}
	if strings.HasPrefix(name, ".") {
		return true
	}
	switch name {
	case "tmp", "node_modules", "dist":
		return true
	default:
		return false
	}
}

func readPackageVersion(path string) ([]byte, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	packageJSON := make(map[string]any)
	if err := json.Unmarshal(data, &packageJSON); err != nil {
		return data, "", err
	}
	version := ""
	if raw, ok := packageJSON["version"]; ok {
		if s, ok := raw.(string); ok {
			version = strings.TrimSpace(s)
		}
	}
	if version == "" {
		return data, "", nil
	}
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	return data, version, nil
}

func startServer(workDir, configPath, logPath string, choysumBinaryPath string) (*exec.Cmd, error) {
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, xfmt.Errorf("open log file: %w", err)
	}
	bin := strings.TrimSpace(choysumBinaryPath)
	if bin == "" {
		bin = os.Args[0]
	}
	cmd := exec.Command(bin, "--config", configPath, "run")
	cmd.Dir = workDir
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Env = append(filteredE2EEnv(os.Environ()), "CHOYSUM_E2E_SKIP_INIT=true")
	setServerProcessAttrs(cmd)
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return nil, xfmt.Errorf("start server: %w", err)
	}
	return cmd, nil
}

func stopServer(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	signalServerProcess(cmd)
	_, _ = cmd.Process.Wait()
}

func runPlaywright(ctx context.Context, opts RunOptions, specsDir string, baseURL string, runtimePath string) error {
	// Playwright defaults testDir to "tests" when not configured, which would not discover
	// colocated module specs under modules/. Generate a minimal per-run config that sets
	// testDir to the specs dir, and pass explicit spec file paths to avoid ambiguous directory matching.
	runDir := filepath.Dir(runtimePath)
	configPath := filepath.Join(runDir, "playwright.e2e.config.cjs")
	configJS := fmt.Sprintf(`/** Auto-generated by choysum test e2e (do not edit). */
module.exports = {
  testDir: %q,
  outputDir: %q,
	// E2E uses a single sqlite DB for the whole scenario run. Parallel workers can
	// cause concurrent writes (e.g. login token creation) and intermittently hit
	// "database is locked". Keep it serial by default; callers may override via
	// Playwright args (e.g. -- --workers=2).
	workers: 1,
  timeout: 60_000,
  expect: { timeout: 10_000 },
  retries: 0,
  use: { trace: 'retain-on-failure' },
};
`, specsDir, filepath.Join(runDir, ".playwright", "test-results"))
	if err := os.WriteFile(configPath, []byte(configJS), 0o644); err != nil {
		return xfmt.Errorf("write playwright config: %w", err)
	}

	specFiles := make([]string, 0)
	if err := filepath.WalkDir(specsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := strings.ToLower(d.Name())
		if strings.HasSuffix(name, ".spec.ts") || strings.HasSuffix(name, ".spec.js") {
			specFiles = append(specFiles, path)
		}
		return nil
	}); err != nil {
		return xfmt.Errorf("discover playwright specs: %w", err)
	}
	sort.Strings(specFiles)
	if len(specFiles) == 0 {
		return xfmt.Errorf("no playwright specs found under %s", specsDir)
	}

	args := []string{"playwright", "test", "--config", configPath}
	args = append(args, specFiles...)
	args = append(args, opts.PlaywrightArgs...)
	playwrightBin := "playwright"
	binDir := ""
	tryBinDir := func(modulesDir string) {
		if strings.TrimSpace(modulesDir) == "" {
			return
		}
		candidateBinDir := filepath.Join(modulesDir, ".bin")
		candidate := filepath.Join(candidateBinDir, "playwright")
		if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
			playwrightBin = candidate
			binDir = candidateBinDir
			return
		}
		candidateCmd := candidate + ".cmd"
		if st, err := os.Stat(candidateCmd); err == nil && !st.IsDir() {
			playwrightBin = candidateCmd
			binDir = candidateBinDir
			return
		}
	}

	if _, err := exec.LookPath("playwright"); err != nil {
		// Fall back to local node_modules search.
		tryBinDir(opts.NpmPath)
		if playwrightBin == "playwright" {
			tryBinDir(filepath.Join(opts.WorkDir, "node_modules"))
		}
		if playwrightBin == "playwright" {
			return xfmt.Errorf("playwright not found. Run: npm install -g @playwright/test && npx playwright install")
		}
	}

	cmd := exec.CommandContext(ctx, playwrightBin, args[1:]...)
	cmd.Dir = opts.WorkDir
	cmd.Stdout = opts.Stdout
	cmd.Stderr = opts.Stderr
	cmd.Env = append(os.Environ(),
		"CHOYSUM_E2E_BASE_URL="+baseURL,
		"CHOYSUM_E2E_RUNTIME_JSON="+runtimePath,
	)
	// Disable Playwright's legacy TS ESM loader path to avoid DEP0205
	// module.register() warnings on newer Node releases.
	cmd.Env = append(cmd.Env, "PW_DISABLE_TS_ESM=1")
	// Generated API files can live outside opts.WorkDir (e.g. under ~/.choysum/generated).
	// Ensure Node can still resolve workspace dependencies from the repository node_modules.
	nodeModulesDir := filepath.Join(opts.WorkDir, "node_modules")
	nodePath := nodeModulesDir
	if existing := strings.TrimSpace(os.Getenv("NODE_PATH")); existing != "" {
		nodePath = nodeModulesDir + string(os.PathListSeparator) + existing
	}
	cmd.Env = append(cmd.Env, "NODE_PATH="+nodePath)
	if binDir != "" {
		// Ensure any helper binaries under node_modules/.bin are discoverable.
		cmd.Env = append(cmd.Env, "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	}
	if err := cmd.Run(); err != nil {
		return xfmt.Errorf("playwright failed: %w", err)
	}
	return nil
}

func waitForHTTP200(ctx context.Context, url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	for time.Now().Before(deadline) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == 200 {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
	return xfmt.Errorf("timeout waiting for %s", url)
}

func includeLogTail(err error, logPath string) error {
	tail := readLogTail(logPath, 32*1024)
	if tail == "" {
		return err
	}
	return xfmt.Errorf("%w\n--- server log tail ---\n%s", err, tail)
}

func readLogTail(path string, maxBytes int64) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return ""
	}
	start := st.Size() - maxBytes
	if start < 0 {
		start = 0
	}
	if _, err := f.Seek(start, 0); err != nil {
		return ""
	}
	b := strings.Builder{}
	s := bufio.NewScanner(f)
	for s.Scan() {
		b.WriteString(s.Text())
		b.WriteString("\n")
	}
	return b.String()
}

func writeRuntime(path string, info runtimeInfo) {
	b, _ := json.MarshalIndent(info, "", "  ")
	_ = os.WriteFile(path, b, 0o644)
}

func pickFreePort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, xfmt.Errorf("listen :0: %w", err)
	}
	defer ln.Close()
	addr := ln.Addr().(*net.TCPAddr)
	return addr.Port, nil
}

func randHex(nBytes int) string {
	b := make([]byte, nBytes)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func resolveRuntimeLogLevel(explicit string, verbose bool) (string, error) {
	level := strings.ToLower(strings.TrimSpace(explicit))
	if level == "" {
		if verbose {
			return "debug", nil
		}
		return "warn", nil
	}
	switch level {
	case "debug", "info", "warn", "error":
		return level, nil
	default:
		return "", xfmt.Errorf("e2e: invalid runtime log level %q (expected debug|info|warn|error)", explicit)
	}
}

// --- source package parsing + scenario resolution ---

type sourceModulePackage struct {
	Name    string              `json:"name"`
	Choysum sourceModuleChoysum `json:"choysum"`
	Depends []string            `json:"-"`
	E2E     *packageE2E         `json:"-"`
	RawPath string              `json:"-"`
	DirName string              `json:"-"`
}

type sourceModuleChoysum struct {
	Depends []string    `json:"depends"`
	E2E     *packageE2E `json:"e2e"`
}

type packageE2E struct {
	Specs     string                  `json:"specs"`
	Scenarios map[string]packageScene `json:"scenarios"`
}

type packageScene struct {
	Extends  string   `json:"extends"`
	Fixtures []string `json:"fixtures"`
}

func discoverSourcePackages(modulesPath string) (map[string]*sourceModulePackage, error) {
	entries, err := os.ReadDir(modulesPath)
	if err != nil {
		return nil, xfmt.Errorf("read modules dir: %w", err)
	}
	out := map[string]*sourceModulePackage{}
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		name := ent.Name()
		packagePath := filepath.Join(modulesPath, name, "package.json")
		b, err := os.ReadFile(packagePath)
		if err != nil {
			continue
		}
		var m sourceModulePackage
		if err := json.Unmarshal(b, &m); err != nil {
			return nil, xfmt.Errorf("parse %s: %w", packagePath, err)
		}
		m.RawPath = packagePath
		m.DirName = name
		m.Depends = m.Choysum.Depends
		m.E2E = m.Choysum.E2E
		// Module identity for dependency resolution is the module directory name.
		// package.json name is treated as metadata and is not used as the key.
		out[name] = &m
	}
	return out, nil
}

func topoClosure(target string, packages map[string]*sourceModulePackage) ([]string, error) {
	vis := map[string]int{} // 0=unseen,1=visiting,2=done
	order := []string{}
	var dfs func(string) error
	dfs = func(n string) error {
		s := strings.TrimSpace(n)
		if s == "" {
			return nil
		}
		st := vis[s]
		if st == 2 {
			return nil
		}
		if st == 1 {
			return xfmt.Errorf("depends cycle detected at %s", s)
		}
		m := packages[s]
		if m == nil {
			return xfmt.Errorf("missing dependency %q", s)
		}
		vis[s] = 1
		for _, dep := range m.Depends {
			dep = strings.TrimSpace(dep)
			if dep == "" {
				continue
			}
			if err := dfs(dep); err != nil {
				return err
			}
		}
		vis[s] = 2
		order = append(order, s)
		return nil
	}
	if err := dfs(target); err != nil {
		return nil, err
	}
	return order, nil
}

func resolveScenarioFixtures(m *sourceModulePackage, scenario string) ([]string, bool, error) {
	if m == nil || m.E2E == nil || len(m.E2E.Scenarios) == 0 {
		return nil, false, nil
	}
	if _, ok := m.E2E.Scenarios[scenario]; !ok {
		// Scenario missing for this module: allowed (skip fixtures).
		return nil, false, nil
	}

	seen := map[string]bool{}
	var walk func(name string) ([]string, error)
	walk = func(name string) ([]string, error) {
		if seen[name] {
			return nil, xfmt.Errorf("scenario extends cycle: %s", name)
		}
		sc, ok := m.E2E.Scenarios[name]
		if !ok {
			// Parent scenario missing: disallowed.
			return nil, xfmt.Errorf("scenario %q extends missing parent scenario %q", scenario, name)
		}
		seen[name] = true
		fixtures := []string{}
		if strings.TrimSpace(sc.Extends) != "" {
			parent := strings.TrimSpace(sc.Extends)
			pf, err := walk(parent)
			if err != nil {
				return nil, err
			}
			fixtures = append(fixtures, pf...)
		}
		for _, f := range sc.Fixtures {
			f = filepath.Clean(strings.TrimSpace(f))
			if f == "" || f == "." {
				continue
			}
			if filepath.IsAbs(f) || f == ".." || strings.HasPrefix(f, ".."+string(filepath.Separator)) {
				return nil, xfmt.Errorf("invalid fixture path %q", f)
			}
			fixtures = append(fixtures, f)
		}
		return fixtures, nil
	}

	paths, err := walk(scenario)
	if err != nil {
		return nil, true, err
	}
	return paths, true, nil
}

// NOTE: module-scoped E2E intentionally does not depend on any runtime setup endpoint.
