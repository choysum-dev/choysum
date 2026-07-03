// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	clioutput "github.com/choysum-dev/choysum/internal/cli/output"
	cliruntime "github.com/choysum-dev/choysum/internal/cli/runtime"
	"github.com/choysum-dev/choysum/internal/config/snapshot"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/server/defaultserver"
	"github.com/spf13/cobra"
)

const (
	runConfigGenerateNext  = "create a valid config file and rerun 'choysum run --config <path>'"
	runConfigFixValuesNext = "fix config values and rerun 'choysum run'"
	runConfigFixFormatNext = "fix the config format and rerun 'choysum run'"
)

var runServerFactory = defaultserver.NewServer
var runExit = os.Exit
var runRuntimeScopeFactory = cliruntime.NewScopeForRun

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
			dbOptions := loadedConfig.scopeInput.DBOptions()
			if dbErr := cliruntime.PrepareRunDatabase(dbOptions); dbErr != nil {
				(&runError{exitCode: dbErr.ExitCode, errMsg: dbErr.ErrMsg, reason: dbErr.Reason, next: dbErr.Next}).exit()
			}

			runtimeScope, envErr := runRuntimeScopeFactory(loadedConfig.scopeInput, loadedConfig.logConfig)
			if envErr != nil {
				printRunScopeInitError(envErr, dbOptions.Dialect)
				runExit(4)
				return
			}

			baseCtx := cmd.Context()
			if baseCtx == nil {
				baseCtx = context.Background()
			}
			ctx, stop := signal.NotifyContext(baseCtx, os.Interrupt, syscall.SIGTERM)
			defer stop()

			// Re-bind scope with the runtime context (for server side gating/options).
			runtimeScope = runtimeScope.WithContext(ctx)

			choysumServer := runServerFactory(runtimeScope)
			if err := choysumServer.Serve(ctx, args...); err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				clioutput.PrintErrorBlock(
					"server exited unexpectedly",
					err.Error(),
					"fix the underlying issue and rerun 'choysum run'",
				)
				runExit(1)
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
	clioutput.PrintErrorBlock(e.errMsg, e.reason, e.next)
	runExit(e.exitCode)
}

func printRunScopeInitError(err error, dbDialect string) {
	if isLikelyRunScopeDBInitError(err, dbDialect) {
		clioutput.PrintErrorBlock(
			fmt.Sprintf("cannot connect to database (dialect=%s)", strings.TrimSpace(dbDialect)),
			"network unreachable / authentication failed / permission denied / database not found (DSN redacted)",
			"verify database reachability and credentials; rerun 'choysum run' to update config if needed",
		)
		return
	}

	reason := "scope initialization failed"
	if err != nil && strings.TrimSpace(err.Error()) != "" {
		reason = err.Error()
	}
	clioutput.PrintErrorBlock(
		"failed to initialize runtime scope",
		reason,
		"fix configuration/runtime initialization and rerun 'choysum run'",
	)
}

func isLikelyRunScopeDBInitError(err error, dbDialect string) bool {
	message := ""
	if err != nil {
		message = strings.ToLower(err.Error())
	}

	for _, keyword := range []string{"database", "dsn", "sqlite", "postgres", "mysql", "sql"} {
		if strings.Contains(message, keyword) {
			return true
		}
	}

	dialect := strings.ToLower(strings.TrimSpace(dbDialect))
	if dialect == "sqlite" || dialect == "postgres" || dialect == "mysql" {
		return strings.Contains(message, dialect)
	}

	return false
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

	if cliruntime.ContainsControl(cfgPath) {
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

type runDBRuntimeOptions = cliruntime.RunDBOptions

type runLoadedConfig struct {
	scopeInput cliruntime.RunScopeInput
	logConfig  *config.LogConfig
}

func resolveRunStartupOptions(scopeInput *cliruntime.RunScopeInput) *runError {
	if scopeInput == nil {
		return &runError{
			exitCode: 3,
			errMsg:   "invalid config",
			reason:   "missing required fields",
			next:     runConfigFixValuesNext,
		}
	}

	cliOptions := scopeInput.CLIOptions()
	if err := cliOptions.Validate(); err != nil {
		return &runError{
			exitCode: 3,
			errMsg:   "invalid config",
			reason:   "invalid config values",
			next:     runConfigFixValuesNext,
		}
	}

	serverOptions := scopeInput.ServerOptions()
	if err := serverOptions.Validate(); err != nil {
		return &runError{
			exitCode: 3,
			errMsg:   "invalid config",
			reason:   "invalid config values",
			next:     runConfigFixValuesNext,
		}
	}

	dbOptions := scopeInput.DBOptions()
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
	cfgOptions := cliruntime.NewScopeInputConfigOptions(snapshot.New(cfg))
	if cfgOptions == nil {
		return runLoadedConfig{}, &runError{
			exitCode: 3,
			errMsg:   "invalid config",
			reason:   "failed to parse config options",
			next:     runConfigFixValuesNext,
		}
	}
	cliOptions := cliruntime.Options{
		DefaultChoysumPath:    cfgOptions.DefaultChoysumPath,
		ModulesPath:           cfgOptions.ModulesPath,
		TmpPath:               cfgOptions.TmpPath,
		ModuleCatalogIndexURL: strings.TrimSpace(cfgOptions.ModuleCatalogIndexURL),
	}
	serverOptions := cliruntime.NewRunServerOptions(cfg.Server)
	dbOptions := cliruntime.NewRunDBOptions(cfg)

	return runLoadedConfig{
		scopeInput: cliruntime.NewRunScopeInput(cfgOptions, cliOptions, serverOptions, dbOptions),
		logConfig:  cliruntime.CloneLogConfig(cfg.Log),
	}, nil
}

func validateRunConfig(scopeInput *cliruntime.RunScopeInput) *runError {
	if scopeInput == nil {
		return &runError{
			exitCode: 3,
			errMsg:   "invalid config",
			reason:   "missing required fields",
			next:     runConfigFixValuesNext,
		}
	}
	cliOptions := scopeInput.CLIOptions()
	normalizedModulesPath, moduleErr := cliruntime.ValidateRunModulesPath(cliOptions.ModulesPath)
	if moduleErr != nil {
		return &runError{exitCode: moduleErr.ExitCode, errMsg: moduleErr.ErrMsg, reason: moduleErr.Reason, next: moduleErr.Next}
	}
	cliOptions.ModulesPath = normalizedModulesPath
	*scopeInput = cliruntime.NewRunScopeInput(scopeInput.ConfigOptions(), cliOptions, scopeInput.ServerOptions(), scopeInput.DBOptions())
	dbErr := cliruntime.ValidateRunDBWithWarning(scopeInput.DBOptions(), clioutput.PrintWarning)
	if dbErr != nil {
		return &runError{exitCode: dbErr.ExitCode, errMsg: dbErr.ErrMsg, reason: dbErr.Reason, next: dbErr.Next}
	}
	return nil
}
