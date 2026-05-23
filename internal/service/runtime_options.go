// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package service

import (
	"strings"

	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
)

type runtimeOptions struct {
	distPath        string
	bundleMode      string
	webBaseURL      string
	rootRedirectURL string
	serverEnv       string

	authEnabled           bool
	authGrpcMethodAccess  bool
	authGrpcRecordRule    bool
	authGrpcCompanyFilter bool
	authGrpcFieldRule     bool
	authGrpcEntryPolicy   map[string]*config.EntryMethodConfig
	authInternalKey       string
	authzDecisionLog      string
	authzDecisionAudit    bool

	taskWorkerHeartbeatIntervalMs           int64
	taskWorkerLeaseDurationMs               int64
	taskWorkerCancelPollIntervalMs          int64
	taskWorkerAlreadyRunningRetryAfterMaxMs int64
	taskSanitizePayloadMaxBytes             int
	taskSanitizeResultMaxBytes              int
	taskSanitizeErrorMaxBytes               int
}

func newRuntimeOptions(pathOpts scope.PathsRuntimeOptions, hasPathOpts bool, compileOpts scope.CompileRuntimeOptions, hasCompileOpts bool, serverOpts scope.ServerRuntimeOptions, hasServerOpts bool, authOpts scope.AuthRuntimeOptions, hasAuthOpts bool, taskOpts scope.TaskRuntimeOptions, hasTaskOpts bool) runtimeOptions {
	compileDefaults := config.NewDefaultCompileConfig()
	serverDefaults := config.NewDefaultServerConfig()
	taskDefaults := config.NewDefaultTaskConfig()

	opts := runtimeOptions{
		bundleMode:                              compileDefaults.BundleMode,
		webBaseURL:                              serverDefaults.WebBaseURL,
		serverEnv:                               serverDefaults.Environment,
		taskWorkerHeartbeatIntervalMs:           taskDefaults.Worker.HeartbeatIntervalMs,
		taskWorkerLeaseDurationMs:               taskDefaults.Worker.LeaseDurationMs,
		taskWorkerCancelPollIntervalMs:          taskDefaults.Worker.CancelPollIntervalMs,
		taskWorkerAlreadyRunningRetryAfterMaxMs: taskDefaults.Worker.AlreadyRunningRetryAfterMaxMs,
		taskSanitizePayloadMaxBytes:             taskDefaults.Sanitize.PayloadMaxBytes,
		taskSanitizeResultMaxBytes:              taskDefaults.Sanitize.ResultMaxBytes,
		taskSanitizeErrorMaxBytes:               taskDefaults.Sanitize.ErrorMaxBytes,
	}

	if hasPathOpts {
		opts.distPath = pathOpts.DistPath
	}
	if hasCompileOpts && strings.TrimSpace(compileOpts.BundleMode) != "" {
		opts.bundleMode = compileOpts.BundleMode
	}
	if hasServerOpts {
		opts.webBaseURL = serverOpts.WebBaseURL
		opts.rootRedirectURL = serverOpts.RootRedirectURL
		opts.serverEnv = serverOpts.Environment
	}
	if hasAuthOpts {
		opts.authEnabled = authOpts.Enabled
		opts.authGrpcMethodAccess = authOpts.GrpcMethodAccess
		opts.authGrpcRecordRule = authOpts.GrpcRecordRule
		opts.authGrpcCompanyFilter = authOpts.GrpcCompanyFilter
		opts.authGrpcFieldRule = authOpts.GrpcFieldRule
		opts.authGrpcEntryPolicy = authOpts.GrpcEntryPolicy
		opts.authInternalKey = authOpts.InternalKey
		opts.authzDecisionLog = authOpts.AuthzDecisionLog
		opts.authzDecisionAudit = authOpts.AuthzDecisionAudit
	}

	if hasTaskOpts && taskOpts.Task != nil {
		taskCfg := taskOpts.Task
		if taskCfg.Worker != nil {
			if taskCfg.Worker.HeartbeatIntervalMs > 0 {
				opts.taskWorkerHeartbeatIntervalMs = taskCfg.Worker.HeartbeatIntervalMs
			}
			if taskCfg.Worker.LeaseDurationMs > 0 {
				opts.taskWorkerLeaseDurationMs = taskCfg.Worker.LeaseDurationMs
			}
			if taskCfg.Worker.CancelPollIntervalMs > 0 {
				opts.taskWorkerCancelPollIntervalMs = taskCfg.Worker.CancelPollIntervalMs
			}
			if taskCfg.Worker.AlreadyRunningRetryAfterMaxMs > 0 {
				opts.taskWorkerAlreadyRunningRetryAfterMaxMs = taskCfg.Worker.AlreadyRunningRetryAfterMaxMs
			}
		}
		if taskCfg.Sanitize != nil {
			if taskCfg.Sanitize.PayloadMaxBytes > 0 {
				opts.taskSanitizePayloadMaxBytes = taskCfg.Sanitize.PayloadMaxBytes
			}
			if taskCfg.Sanitize.ResultMaxBytes > 0 {
				opts.taskSanitizeResultMaxBytes = taskCfg.Sanitize.ResultMaxBytes
			}
			if taskCfg.Sanitize.ErrorMaxBytes > 0 {
				opts.taskSanitizeErrorMaxBytes = taskCfg.Sanitize.ErrorMaxBytes
			}
		}
	}

	return opts
}

func runtimeOptionsFromScope(runtimeScope scope.Scope) runtimeOptions {
	if runtimeScope == nil {
		return newRuntimeOptions(scope.PathsRuntimeOptions{}, false, scope.CompileRuntimeOptions{}, false, scope.ServerRuntimeOptions{}, false, scope.AuthRuntimeOptions{}, false, scope.TaskRuntimeOptions{}, false)
	}
	pathOpts, hasPathOpts := scope.PathsRuntimeOptionsFromScope(runtimeScope)
	compileOpts, hasCompileOpts := scope.CompileRuntimeOptionsFromScope(runtimeScope)
	serverOpts, hasServerOpts := scope.ServerRuntimeOptionsFromScope(runtimeScope)
	authOpts, hasAuthOpts := scope.AuthRuntimeOptionsFromScope(runtimeScope)
	taskOpts, hasTaskOpts := scope.TaskRuntimeOptionsFromScope(runtimeScope)
	return newRuntimeOptions(pathOpts, hasPathOpts, compileOpts, hasCompileOpts, serverOpts, hasServerOpts, authOpts, hasAuthOpts, taskOpts, hasTaskOpts)
}

func hasRuntimeOptions(opts runtimeOptions) bool {
	return strings.TrimSpace(opts.bundleMode) != "" || strings.TrimSpace(opts.webBaseURL) != ""
}

func (s *ApplicationService) resolvedRuntimeOptions() runtimeOptions {
	if s != nil && hasRuntimeOptions(s.runtimeOptions) {
		return s.runtimeOptions
	}
	if s != nil && s.runtimeScope != nil {
		return runtimeOptionsFromScope(s.runtimeScope)
	}
	if s != nil {
		return s.runtimeOptions
	}
	return newRuntimeOptions(scope.PathsRuntimeOptions{}, false, scope.CompileRuntimeOptions{}, false, scope.ServerRuntimeOptions{}, false, scope.AuthRuntimeOptions{}, false, scope.TaskRuntimeOptions{}, false)
}

func (o runtimeOptions) Validate() error {
	if strings.TrimSpace(o.distPath) == "" {
		return xfmt.Errorf("service runtime options: distPath is required")
	}
	if strings.TrimSpace(o.bundleMode) == "" {
		return xfmt.Errorf("service runtime options: bundleMode is required")
	}
	if strings.TrimSpace(o.webBaseURL) == "" {
		return xfmt.Errorf("service runtime options: webBaseURL is required")
	}
	return nil
}
