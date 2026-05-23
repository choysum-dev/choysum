// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package task

import (
	"time"

	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
)

type runtimeOptions struct {
	initialized bool

	dbDialect         string
	serverEnvironment string
	authInternalKey   string

	dispatchMaxConcurrency       int
	dispatchPollInterval         time.Duration
	dispatchFetchBatchSize       int
	dispatchReadyQueueMax        int
	dispatchMaxConcurrencyPerApp int
	dispatchDefaultMaxAttempts   int
	dispatchRetryAfterMsCap      int64
	dispatchBackoffMaxMs         int64
	dispatchDefaultJobTimeoutMs  int64
	dispatchJobTokenTTLms        int64

	scheduleMaxScheduleEnqueuesPerMinute int

	sanitizePayloadMaxBytes int
	sanitizeResultMaxBytes  int
	sanitizeErrorMaxBytes   int

	retentionGCInterval time.Duration
	retention           *config.TaskRetentionConfig
}

func newRuntimeOptions(dbOpts scope.DatabaseRuntimeOptions, hasDBOpts bool, serverOpts scope.ServerRuntimeOptions, hasServerOpts bool, authOpts scope.AuthRuntimeOptions, hasAuthOpts bool, taskOpts scope.TaskRuntimeOptions, hasTaskOpts bool) runtimeOptions {
	taskDefaults := config.NewDefaultTaskConfig()
	serverDefaults := config.NewDefaultServerConfig()

	opts := runtimeOptions{
		initialized: true,

		serverEnvironment: serverDefaults.Environment,

		dispatchMaxConcurrency:               20,
		dispatchPollInterval:                 time.Second,
		dispatchDefaultMaxAttempts:           1,
		dispatchRetryAfterMsCap:              300000,
		dispatchBackoffMaxMs:                 60000,
		dispatchDefaultJobTimeoutMs:          0,
		dispatchJobTokenTTLms:                int64((2 * time.Minute) / time.Millisecond),
		sanitizePayloadMaxBytes:              taskDefaults.Sanitize.PayloadMaxBytes,
		sanitizeResultMaxBytes:               taskDefaults.Sanitize.ResultMaxBytes,
		sanitizeErrorMaxBytes:                taskDefaults.Sanitize.ErrorMaxBytes,
		retentionGCInterval:                  time.Hour,
		retention:                            cloneTaskRetentionConfig(nil, taskDefaults.Retention),
		scheduleMaxScheduleEnqueuesPerMinute: 0,
	}

	if hasDBOpts {
		opts.dbDialect = dbOpts.Dialect
	}
	if hasServerOpts {
		opts.serverEnvironment = serverOpts.Environment
	}
	if hasAuthOpts {
		opts.authInternalKey = authOpts.InternalKey
	}
	if !hasTaskOpts || taskOpts.Task == nil {
		return opts
	}
	taskCfg := taskOpts.Task

	if taskCfg.Dispatch != nil {
		if taskCfg.Dispatch.MaxConcurrency > 0 {
			opts.dispatchMaxConcurrency = taskCfg.Dispatch.MaxConcurrency
		}
		if taskCfg.Dispatch.PollIntervalMs > 0 {
			opts.dispatchPollInterval = time.Duration(taskCfg.Dispatch.PollIntervalMs) * time.Millisecond
		}
		opts.dispatchFetchBatchSize = taskCfg.Dispatch.FetchBatchSize
		opts.dispatchReadyQueueMax = taskCfg.Dispatch.ReadyQueueMax
		opts.dispatchMaxConcurrencyPerApp = taskCfg.Dispatch.MaxConcurrencyPerApp
		if taskCfg.Dispatch.DefaultMaxAttempts > 0 {
			opts.dispatchDefaultMaxAttempts = taskCfg.Dispatch.DefaultMaxAttempts
		}
		if taskCfg.Dispatch.RetryAfterMsCap > 0 {
			opts.dispatchRetryAfterMsCap = taskCfg.Dispatch.RetryAfterMsCap
		}
		if taskCfg.Dispatch.BackoffMaxMs > 0 {
			opts.dispatchBackoffMaxMs = taskCfg.Dispatch.BackoffMaxMs
		}
		if taskCfg.Dispatch.DefaultJobTimeoutMs > 0 {
			opts.dispatchDefaultJobTimeoutMs = taskCfg.Dispatch.DefaultJobTimeoutMs
		}
		if taskCfg.Dispatch.JobTokenTTLms > 0 {
			opts.dispatchJobTokenTTLms = taskCfg.Dispatch.JobTokenTTLms
		}
	}

	if taskCfg.Schedule != nil {
		opts.scheduleMaxScheduleEnqueuesPerMinute = taskCfg.Schedule.MaxScheduleEnqueuesPerMinute
	}
	if taskCfg.Sanitize != nil {
		if taskCfg.Sanitize.PayloadMaxBytes > 0 {
			opts.sanitizePayloadMaxBytes = taskCfg.Sanitize.PayloadMaxBytes
		}
		if taskCfg.Sanitize.ResultMaxBytes > 0 {
			opts.sanitizeResultMaxBytes = taskCfg.Sanitize.ResultMaxBytes
		}
		if taskCfg.Sanitize.ErrorMaxBytes > 0 {
			opts.sanitizeErrorMaxBytes = taskCfg.Sanitize.ErrorMaxBytes
		}
	}
	if taskCfg.Retention != nil && taskCfg.Retention.GCIntervalMs > 0 {
		opts.retentionGCInterval = time.Duration(taskCfg.Retention.GCIntervalMs) * time.Millisecond
	}
	opts.retention = cloneTaskRetentionConfig(taskCfg.Retention, taskDefaults.Retention)

	return opts
}

func runtimeOptionsFromScope(runtimeScope scope.Scope) runtimeOptions {
	if runtimeScope == nil {
		return newRuntimeOptions(scope.DatabaseRuntimeOptions{}, false, scope.ServerRuntimeOptions{}, false, scope.AuthRuntimeOptions{}, false, scope.TaskRuntimeOptions{}, false)
	}
	dbOpts, hasDBOpts := scope.DatabaseRuntimeOptionsFromScope(runtimeScope)
	serverOpts, hasServerOpts := scope.ServerRuntimeOptionsFromScope(runtimeScope)
	authOpts, hasAuthOpts := scope.AuthRuntimeOptionsFromScope(runtimeScope)
	taskOpts, hasTaskOpts := scope.TaskRuntimeOptionsFromScope(runtimeScope)
	return newRuntimeOptions(dbOpts, hasDBOpts, serverOpts, hasServerOpts, authOpts, hasAuthOpts, taskOpts, hasTaskOpts)
}

func cloneTaskRetentionConfig(cfg *config.TaskRetentionConfig, defaults *config.TaskRetentionConfig) *config.TaskRetentionConfig {
	if defaults == nil {
		defaults = config.NewDefaultTaskConfig().Retention
	}
	if cfg == nil {
		cfg = defaults
	}

	out := &config.TaskRetentionConfig{
		GCIntervalMs:  cfg.GCIntervalMs,
		TaskJob:       cloneTaskRetentionEntry(cfg.TaskJob, defaults.TaskJob),
		TaskExecution: cloneTaskRetentionEntry(cfg.TaskExecution, defaults.TaskExecution),
	}
	return out
}

func cloneTaskRetentionEntry(entry *config.TaskRetentionEntry, defaults *config.TaskRetentionEntry) *config.TaskRetentionEntry {
	if defaults == nil {
		defaults = &config.TaskRetentionEntry{Overrides: map[string]*config.TaskRetentionPolicy{}}
	}
	if entry == nil {
		entry = defaults
	}

	out := &config.TaskRetentionEntry{
		TaskRetentionPolicy: entry.TaskRetentionPolicy,
		Overrides:           map[string]*config.TaskRetentionPolicy{},
	}

	overrides := entry.Overrides
	if overrides == nil {
		overrides = defaults.Overrides
	}
	for method, policy := range overrides {
		if policy == nil {
			continue
		}
		copied := *policy
		out.Overrides[method] = &copied
	}
	return out
}
