// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/choysum-dev/choysum/internal/config/snapshot"
	"github.com/choysum-dev/choysum/internal/logger"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/choysum-dev/choysum/pkg/server/defaultserver"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

const (
	runConfigGenerateNext  = "create a valid config file and rerun 'choysum run --config <path>'"
	runConfigFixValuesNext = "fix config values and rerun 'choysum run'"
	runConfigFixFormatNext = "fix the config format and rerun 'choysum run'"
)

var runServerFactory = defaultserver.NewServer

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run Choysum Application",
		Run: func(cmd *cobra.Command, args []string) {
			cfgPath, runErr := resolveRunConfigPath(cmd)
			if runErr != nil {
				runErr.exit()
			}
			loadedConfig, runErr := loadRunConfig(cfgPath)
			if runErr != nil {
				runErr.exit()
			}
			if runErr := validateRunConfig(&loadedConfig.scopeInput); runErr != nil {
				runErr.exit()
			}
			if runErr := resolveRunStartupOptions(&loadedConfig.scopeInput); runErr != nil {
				runErr.exit()
			}
			if runErr := prepareRunDatabase(loadedConfig.scopeInput.dbOptions); runErr != nil {
				runErr.exit()
			}

			dbOptions := loadedConfig.scopeInput.dbOptions

			runtimeScope, envErr := newRuntimeScopeForRun(loadedConfig.scopeInput, loadedConfig.logConfig)
			if envErr != nil {
				printErrorBlock(
					fmt.Sprintf("cannot connect to database (dialect=%s)", dbOptions.dialect),
					"network unreachable / authentication failed / permission denied / database not found (DSN redacted)",
					"verify database reachability and credentials; rerun 'choysum run' to update config if needed",
				)
				os.Exit(4)
			}

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			// Re-bind scope with the runtime context (for server side gating/options).
			runtimeScope = runtimeScope.WithContext(ctx)

			choysumServer := runServerFactory(runtimeScope)
			if err := choysumServer.Serve(ctx, args...); err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				printErrorBlock(
					"server exited unexpectedly",
					err.Error(),
					"fix the underlying issue and rerun 'choysum run'",
				)
				os.Exit(1)
			}
		},
	}
	return cmd
}

type runError struct {
	exitCode int
	errMsg   string
	reason   string
	next     string
}

func (e *runError) exit() {
	printErrorBlock(e.errMsg, e.reason, e.next)
	os.Exit(e.exitCode)
}

func resolveRunConfigPath(cmd *cobra.Command) (string, *runError) {
	var cfgPath string
	if cmd != nil {
		var err error
		cfgPath, err = cmd.Flags().GetString("config")
		if err != nil {
			return "", &runError{
				exitCode: 2,
				errMsg:   "invalid config flag",
				reason:   "failed to read --config",
				next:     "fix --config and retry",
			}
		}
	}

	if cfgPath == "" {
		path, err := config.DefaultConfigPath()
		if err != nil {
			return "", &runError{
				exitCode: 3,
				errMsg:   "cannot resolve default config file",
				reason:   err.Error(),
				next:     "fix the default config path permissions and retry",
			}
		}
		if path == "" {
			return "", nil
		}
		cfgPath = path
	}

	if containsControl(cfgPath) {
		return "", &runError{
			exitCode: 2,
			errMsg:   "invalid config flag",
			reason:   "path contains NUL (\\x00) or newline (\\n/\\r)",
			next:     "fix --config and retry",
		}
	}
	if strings.TrimSpace(cfgPath) == "" {
		return "", &runError{
			exitCode: 2,
			errMsg:   "invalid config flag",
			reason:   "path must not be empty or whitespace",
			next:     "fix --config and retry",
		}
	}
	if strings.TrimSpace(cfgPath) != cfgPath {
		return "", &runError{
			exitCode: 2,
			errMsg:   "invalid config flag",
			reason:   "path has leading or trailing whitespace",
			next:     "fix --config and retry",
		}
	}

	if !filepath.IsAbs(cfgPath) {
		absPath, err := filepath.Abs(cfgPath)
		if err != nil {
			return "", &runError{
				exitCode: 3,
				errMsg:   "invalid config",
				reason:   "invalid config path",
				next:     "fix the path and retry",
			}
		}
		cfgPath = absPath
	}
	info, err := os.Lstat(cfgPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", &runError{
				exitCode: 3,
				errMsg:   "config file not found",
				reason:   "file not found",
				next:     runConfigGenerateNext,
			}
		}
		return "", &runError{
			exitCode: 3,
			errMsg:   fmt.Sprintf("cannot read config file: %s", cfgPath),
			reason:   "file not found or permission denied",
			next:     runConfigGenerateNext,
		}
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", &runError{
			exitCode: 3,
			errMsg:   "invalid config",
			reason:   "config path is a symlink",
			next:     "use a regular config file path and retry",
		}
	}
	if info.IsDir() {
		return "", &runError{
			exitCode: 3,
			errMsg:   "invalid config",
			reason:   "config path is a directory",
			next:     "use a regular config file path and retry",
		}
	}

	return cfgPath, nil
}

type runServerRuntimeOptions struct {
	bindAddress string
	port        int
	enabledTLS  bool
}

type runDBRuntimeOptions struct {
	dialect     string
	dsn         string
	allowCreate bool
}

type runLoadedConfig struct {
	scopeInput runRuntimeScopeInput
	logConfig  *config.LogConfig
}

func newRunServerRuntimeOptions(serverCfg *config.ServerConfig) runServerRuntimeOptions {
	defaults := config.NewDefaultServerConfig()
	options := runServerRuntimeOptions{
		bindAddress: defaults.BindAddress,
		port:        defaults.Port,
		enabledTLS:  defaults.EnabledTLS,
	}
	if serverCfg == nil {
		return options
	}
	if strings.TrimSpace(serverCfg.BindAddress) != "" {
		options.bindAddress = serverCfg.BindAddress
	}
	if serverCfg.Port > 0 {
		options.port = serverCfg.Port
	}
	options.enabledTLS = serverCfg.EnabledTLS
	return options
}

func (o runServerRuntimeOptions) Validate() error {
	if strings.TrimSpace(o.bindAddress) == "" {
		return fmt.Errorf("run server options: bindAddress is required")
	}
	if o.port <= 0 {
		return fmt.Errorf("run server options: port must be positive")
	}
	return nil
}

func newRunDBRuntimeOptions(cfg *config.Config) runDBRuntimeOptions {
	if cfg == nil || cfg.Db == nil {
		return runDBRuntimeOptions{}
	}
	dialect := strings.ToLower(strings.TrimSpace(cfg.Db.Dialect))
	dsn := cfg.Db.DSN
	return runDBRuntimeOptions{
		dialect:     dialect,
		dsn:         dsn,
		allowCreate: dialect == "sqlite" && isDefaultRunSqlitePath(dsn, config.DefaultSQLitePath(cfg.DefaultChoysumPath)),
	}
}

func (o runDBRuntimeOptions) Validate() error {
	if strings.TrimSpace(o.dialect) == "" {
		return fmt.Errorf("run db options: dialect is required")
	}
	return nil
}

func resolveRunStartupOptions(scopeInput *runRuntimeScopeInput) *runError {
	if scopeInput == nil {
		return &runError{
			exitCode: 3,
			errMsg:   "invalid config",
			reason:   "missing required fields",
			next:     runConfigFixValuesNext,
		}
	}

	cliOptions := scopeInput.cliOptions
	if err := cliOptions.Validate(); err != nil {
		return &runError{
			exitCode: 3,
			errMsg:   "invalid config",
			reason:   "invalid config values",
			next:     runConfigFixValuesNext,
		}
	}

	serverOptions := scopeInput.serverOptions
	if err := serverOptions.Validate(); err != nil {
		return &runError{
			exitCode: 3,
			errMsg:   "invalid config",
			reason:   "invalid config values",
			next:     runConfigFixValuesNext,
		}
	}

	dbOptions := scopeInput.dbOptions
	if err := dbOptions.Validate(); err != nil {
		return &runError{
			exitCode: 3,
			errMsg:   "invalid config",
			reason:   "invalid config values",
			next:     runConfigFixValuesNext,
		}
	}

	return nil
}

func loadRunConfig(cfgPath string) (runLoadedConfig, *runError) {
	cfg, err := config.LoadWithProvider(nil, cfgPath)
	if err != nil {
		lower := strings.ToLower(err.Error())
		if strings.Contains(lower, "read config file failed") &&
			(strings.Contains(lower, "permission denied") || strings.Contains(lower, "no such file") || strings.Contains(lower, "file not found")) {
			return runLoadedConfig{}, &runError{
				exitCode: 3,
				errMsg:   fmt.Sprintf("cannot read config file: %s", cfgPath),
				reason:   "file not found or permission denied",
				next:     runConfigGenerateNext,
			}
		}
		reason := "invalid config values"
		if strings.Contains(lower, "read config file failed") || strings.Contains(lower, "yaml:") {
			reason = "invalid config format (YAML parse failed)"
		}
		next := runConfigFixValuesNext
		if reason == "invalid config format (YAML parse failed)" {
			next = runConfigFixFormatNext
		}
		return runLoadedConfig{}, &runError{
			exitCode: 3,
			errMsg:   "invalid config",
			reason:   reason,
			next:     next,
		}
	}
	if !configHasAddonsPath(cfgPath) {
		cfg.AddonsPath = "./addons"
	}
	cfgOptions := newScopeInputConfigOptions(snapshot.New(cfg))
	cliOptions := newCliRuntimeOptionsFromScopeInputOptions(cfgOptions)
	serverOptions := newRunServerRuntimeOptions(cfg.Server)
	dbOptions := newRunDBRuntimeOptions(cfg)

	return runLoadedConfig{
		scopeInput: newRunRuntimeScopeInput(cfgOptions, cliOptions, serverOptions, dbOptions),
		logConfig:  cloneRunLogConfig(cfg.Log),
	}, nil
}

func cloneRunLogConfig(cfg *config.LogConfig) *config.LogConfig {
	if cfg == nil {
		return config.NewDefaultLogConfig()
	}
	cloned := *cfg
	return &cloned
}

func configHasAddonsPath(cfgPath string) bool {
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return true
	}
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return true
	}
	_, ok := raw["addons_path"]
	return ok
}

func validateRunConfig(scopeInput *runRuntimeScopeInput) *runError {
	if scopeInput == nil {
		return &runError{
			exitCode: 3,
			errMsg:   "invalid config",
			reason:   "missing required fields",
			next:     runConfigFixValuesNext,
		}
	}
	if err := validateRunAddonsPath(&scopeInput.cliOptions); err != nil {
		return err
	}
	return validateRunDb(scopeInput.dbOptions)
}

func validateRunAddonsPath(options *cliRuntimeOptions) *runError {
	if options == nil {
		return &runError{
			exitCode: 3,
			errMsg:   "invalid config",
			reason:   "missing required fields",
			next:     runConfigFixValuesNext,
		}
	}

	path := options.addonsPath
	if path == "" {
		path = "./addons"
		options.addonsPath = path
	}
	if strings.TrimSpace(path) == "" {
		return &runError{
			exitCode: 3,
			errMsg:   "invalid config",
			reason:   "missing required fields",
			next:     runConfigFixValuesNext,
		}
	}
	if strings.TrimSpace(path) != path {
		return &runError{
			exitCode: 3,
			errMsg:   "invalid config",
			reason:   "invalid value",
			next:     runConfigFixValuesNext,
		}
	}
	if containsControl(path) {
		return &runError{
			exitCode: 3,
			errMsg:   "invalid config",
			reason:   "invalid value",
			next:     runConfigFixValuesNext,
		}
	}
	if hasPathListSeparator(path) {
		return &runError{
			exitCode: 3,
			errMsg:   "invalid config",
			reason:   "invalid value",
			next:     runConfigFixValuesNext,
		}
	}
	if !filepath.IsAbs(path) {
		absPath, err := filepath.Abs(path)
		if err == nil {
			path = absPath
			options.addonsPath = absPath
		}
	}
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &runError{
				exitCode: 3,
				errMsg:   "invalid config",
				reason:   "path does not exist",
				next:     runConfigFixValuesNext,
			}
		}
		return &runError{
			exitCode: 3,
			errMsg:   "invalid config",
			reason:   "permission denied or not accessible",
			next:     runConfigFixValuesNext,
		}
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return &runError{
			exitCode: 3,
			errMsg:   "invalid config",
			reason:   "invalid value",
			next:     runConfigFixValuesNext,
		}
	}
	if !info.IsDir() {
		return &runError{
			exitCode: 3,
			errMsg:   "invalid config",
			reason:   "invalid value",
			next:     runConfigFixValuesNext,
		}
	}
	if _, err := os.ReadDir(path); err != nil {
		return &runError{
			exitCode: 3,
			errMsg:   "invalid config",
			reason:   "permission denied or not accessible",
			next:     runConfigFixValuesNext,
		}
	}
	return nil
}

func validateRunDb(dbOptions runDBRuntimeOptions) *runError {
	dialect := strings.ToLower(strings.TrimSpace(dbOptions.dialect))
	if dialect == "" {
		return &runError{
			exitCode: 3,
			errMsg:   "invalid config",
			reason:   "missing required fields",
			next:     runConfigFixValuesNext,
		}
	}
	if dialect != "sqlite" && dialect != "postgres" && dialect != "mysql" {
		return &runError{
			exitCode: 3,
			errMsg:   "invalid config",
			reason:   "invalid value",
			next:     runConfigFixValuesNext,
		}
	}
	if dbOptions.dsn == "" || strings.TrimSpace(dbOptions.dsn) == "" {
		return &runError{
			exitCode: 3,
			errMsg:   "invalid config",
			reason:   "missing required fields",
			next:     runConfigFixValuesNext,
		}
	}

	if dialect == "sqlite" {
		return validateRunSqlite(dbOptions.dsn, dbOptions.allowCreate)
	}
	return validateRunDatabaseDsn(dialect, dbOptions.dsn)
}

func prepareRunDatabase(dbOptions runDBRuntimeOptions) *runError {
	if strings.ToLower(strings.TrimSpace(dbOptions.dialect)) != "sqlite" || !dbOptions.allowCreate {
		return nil
	}
	return prepareRunSqlite(dbOptions.dsn)
}

func prepareRunSqlite(dsn string) *runError {
	path, err := sqlitePathFromDsn(dsn)
	if err != nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return &runError{
			exitCode: 3,
			errMsg:   "invalid sqlite path",
			reason:   "permission denied or not accessible",
			next:     runSqlitePathNext(true),
		}
	}
	return nil
}

func validateRunDatabaseDsn(dialect, dsn string) *runError {
	if containsControl(dsn) {
		return &runError{
			exitCode: 3,
			errMsg:   "invalid database dsn",
			reason:   "dsn contains NUL (\\x00) or newline (\\n/\\r)",
			next:     "remove control characters from dsn and retry; if it comes from config, fix the config and rerun",
		}
	}
	if strings.TrimSpace(dsn) == "" {
		return &runError{
			exitCode: 3,
			errMsg:   "invalid config",
			reason:   "missing required fields",
			next:     runConfigFixValuesNext,
		}
	}
	if strings.TrimSpace(dsn) != dsn {
		return &runError{
			exitCode: 3,
			errMsg:   "invalid database dsn",
			reason:   "dsn has leading or trailing whitespace",
			next:     "remove leading/trailing whitespace from dsn and retry; if it comes from config, fix the config and rerun",
		}
	}

	if scheme := urlScheme(dsn); scheme != "" {
		scheme = strings.ToLower(scheme)
		expected := dialect
		if scheme == "postgresql" {
			scheme = "postgres"
		}
		if scheme != expected {
			return &runError{
				exitCode: 3,
				errMsg:   "invalid config",
				reason:   "db.dialect conflicts with dsn scheme",
				next:     "make db.dialect and dsn scheme consistent and retry",
			}
		}
	}

	return nil
}

func validateRunSqlite(dsn string, allowCreate bool) *runError {
	if containsControl(dsn) {
		return &runError{
			exitCode: 3,
			errMsg:   "invalid sqlite path",
			reason:   "path contains NUL (\\x00) or newline (\\n/\\r)",
			next:     runSqlitePathNext(allowCreate),
		}
	}
	if strings.TrimSpace(dsn) == "" {
		return &runError{
			exitCode: 3,
			errMsg:   "invalid sqlite dsn",
			reason:   "sqlite dsn must not be empty or whitespace",
			next:     "use a sqlite file path or file: URI and not :memory:; for run, the path must be absolute",
		}
	}
	if strings.EqualFold(strings.TrimSpace(dsn), ":memory:") {
		return &runError{
			exitCode: 3,
			errMsg:   "invalid sqlite dsn",
			reason:   "sqlite dsn must not be :memory:",
			next:     "use a sqlite file path or file: URI and not :memory:; for run, the path must be absolute",
		}
	}
	if strings.TrimSpace(dsn) != dsn {
		return &runError{
			exitCode: 3,
			errMsg:   "invalid sqlite path",
			reason:   "path has leading or trailing whitespace",
			next:     "choose a valid sqlite file path and retry; for run, ensure the path is absolute and the DB file already exists and is a regular file",
		}
	}
	path, parseErr := sqlitePathFromDsn(dsn)
	if parseErr != nil {
		return &runError{
			exitCode: 3,
			errMsg:   "invalid sqlite dsn",
			reason:   "sqlite dsn must be a file: URI or plain file path (other URI schemes are not allowed)",
			next:     "use a sqlite file path or file: URI and not :memory:; for run, the path must be absolute",
		}
	}
	if strings.TrimSpace(path) == "" {
		return &runError{
			exitCode: 3,
			errMsg:   "invalid sqlite dsn",
			reason:   "sqlite dsn must not be empty or whitespace",
			next:     "use a sqlite file path or file: URI and not :memory:; for run, the path must be absolute",
		}
	}
	if strings.EqualFold(strings.TrimSpace(path), ":memory:") {
		return &runError{
			exitCode: 3,
			errMsg:   "invalid sqlite dsn",
			reason:   "sqlite dsn must not be :memory:",
			next:     "use a sqlite file path or file: URI and not :memory:; for run, the path must be absolute",
		}
	}
	if strings.TrimSpace(path) != path {
		return &runError{
			exitCode: 3,
			errMsg:   "invalid sqlite path",
			reason:   "path has leading or trailing whitespace",
			next:     runSqlitePathNext(allowCreate),
		}
	}
	if !filepath.IsAbs(path) {
		return &runError{
			exitCode: 3,
			errMsg:   "invalid sqlite path",
			reason:   "path is not absolute",
			next:     runSqlitePathNext(allowCreate),
		}
	}
	if hasParentSymlink(path) {
		printCLIWarning("sqlite parent directory is a symlink")
	}
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			if allowCreate {
				return nil
			}
			return &runError{
				exitCode: 3,
				errMsg:   "invalid sqlite path",
				reason:   "path does not exist",
				next:     runSqlitePathNext(false),
			}
		}
		return &runError{
			exitCode: 3,
			errMsg:   "invalid sqlite path",
			reason:   "permission denied or not accessible",
			next:     runSqlitePathNext(allowCreate),
		}
	}
	if info.IsDir() {
		return &runError{
			exitCode: 3,
			errMsg:   "invalid sqlite path",
			reason:   "path is a directory",
			next:     runSqlitePathNext(allowCreate),
		}
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return &runError{
			exitCode: 3,
			errMsg:   "invalid sqlite path",
			reason:   "path is a symlink",
			next:     runSqlitePathNext(allowCreate),
		}
	}
	if !info.Mode().IsRegular() {
		return &runError{
			exitCode: 3,
			errMsg:   "invalid sqlite path",
			reason:   "path is not a regular file",
			next:     runSqlitePathNext(allowCreate),
		}
	}
	return nil
}

func runSqlitePathNext(allowCreate bool) string {
	if allowCreate {
		return "choose a valid sqlite file path and retry; for run, ensure the path is absolute and accessible"
	}
	return "choose a valid sqlite file path and retry; for run, ensure the path is absolute and the DB file already exists and is a regular file"
}

func isDefaultRunSqlitePath(dsn string, defaultPath string) bool {
	if strings.TrimSpace(dsn) == "" || strings.TrimSpace(defaultPath) == "" {
		return false
	}
	path, err := sqlitePathFromDsn(dsn)
	if err != nil {
		return false
	}
	return filepath.Clean(path) == filepath.Clean(defaultPath)
}

func sqlitePathFromDsn(dsn string) (string, error) {
	trimmed := strings.TrimSpace(dsn)
	if strings.HasPrefix(strings.ToLower(trimmed), "file:") || strings.Contains(trimmed, "://") {
		parsed, err := url.Parse(trimmed)
		if err != nil {
			return "", err
		}
		scheme := strings.ToLower(parsed.Scheme)
		if scheme != "" && scheme != "file" {
			return "", errors.New("unsupported sqlite dsn scheme")
		}
		if parsed.Path != "" {
			return parsed.Path, nil
		}
		return parsed.Opaque, nil
	}
	return trimmed, nil
}

func newRuntimeScopeForRun(scopeInput runRuntimeScopeInput, logConfig *config.LogConfig) (scope.Scope, error) {
	if err := scopeInput.cliOptions.Validate(); err != nil {
		return nil, fmt.Errorf("invalid cli runtime options: %w", err)
	}
	if err := scopeInput.serverOptions.Validate(); err != nil {
		return nil, fmt.Errorf("invalid run server options: %w", err)
	}
	if err := scopeInput.dbOptions.Validate(); err != nil {
		return nil, fmt.Errorf("invalid run db options: %w", err)
	}
	if scopeInput.options == nil {
		return nil, fmt.Errorf("config is required")
	}

	var runtimeScope scope.Scope
	var panicErr any

	func() {
		defer func() {
			if r := recover(); r != nil {
				panicErr = r
			}
		}()
		l := logger.NewLoggerWithWriter(cloneRunLogConfig(logConfig), os.Stderr)
		runtimeScope = scope.NewScope(context.Background(), scopeInput, l)
	}()

	if panicErr != nil {
		return nil, fmt.Errorf("failed to initialize scope: %v", panicErr)
	}
	if runtimeScope == nil {
		return nil, fmt.Errorf("failed to initialize scope")
	}
	return runtimeScope, nil
}

func hasPathListSeparator(path string) bool {
	if runtime.GOOS == "windows" {
		if strings.Contains(path, ";") {
			return true
		}
		if strings.Contains(path, ":") {
			if len(path) >= 3 && ((path[0] >= 'A' && path[0] <= 'Z') || (path[0] >= 'a' && path[0] <= 'z')) && path[1] == ':' && (path[2] == '\\' || path[2] == '/') {
				return false
			}
			return true
		}
		return false
	}
	return strings.Contains(path, ":")
}

func containsControl(value string) bool {
	return strings.ContainsRune(value, '\x00') || strings.ContainsRune(value, '\n') || strings.ContainsRune(value, '\r')
}

func urlScheme(dsn string) string {
	value := strings.ToLower(dsn)
	if strings.HasPrefix(value, "postgres://") || strings.HasPrefix(value, "postgresql://") || strings.HasPrefix(value, "mysql://") {
		idx := strings.Index(value, "://")
		if idx > 0 {
			return value[:idx]
		}
	}
	return ""
}

func hasParentSymlink(path string) bool {
	current := filepath.Dir(path)
	for {
		info, err := os.Lstat(current)
		if err != nil {
			return false
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return true
		}
		parent := filepath.Dir(current)
		if parent == current {
			return false
		}
		current = parent
	}
}
