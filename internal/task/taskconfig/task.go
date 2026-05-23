// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package taskconfig

import "github.com/spf13/viper"

type TaskConfig struct {
	Dispatch  *TaskDispatchConfig  `mapstructure:"dispatch"`
	Schedule  *TaskScheduleConfig  `mapstructure:"schedule"`
	Worker    *TaskWorkerConfig    `mapstructure:"worker"`
	Sanitize  *TaskSanitizeConfig  `mapstructure:"sanitize"`
	Retention *TaskRetentionConfig `mapstructure:"retention"`
}

type TaskDispatchConfig struct {
	MaxConcurrency       int   `mapstructure:"max_concurrency"`
	MaxConcurrencyPerApp int   `mapstructure:"max_concurrency_per_app"`
	PollIntervalMs       int64 `mapstructure:"poll_interval_ms"`
	FetchBatchSize       int   `mapstructure:"fetch_batch_size"`
	ReadyQueueMax        int   `mapstructure:"ready_queue_max"`
	DefaultMaxAttempts   int   `mapstructure:"default_max_attempts"`
	RetryAfterMsCap      int64 `mapstructure:"retry_after_ms_cap"`
	BackoffMaxMs         int64 `mapstructure:"backoff_max_ms"`
	DefaultJobTimeoutMs  int64 `mapstructure:"default_job_timeout_ms"`
	JobTokenTTLms        int64 `mapstructure:"job_token_ttl_ms"`
}

type TaskScheduleConfig struct {
	MaxScheduleEnqueuesPerMinute int `mapstructure:"max_schedule_enqueues_per_minute"`
}

type TaskWorkerConfig struct {
	HeartbeatIntervalMs           int64 `mapstructure:"heartbeat_interval_ms"`
	LeaseDurationMs               int64 `mapstructure:"lease_duration_ms"`
	CancelPollIntervalMs          int64 `mapstructure:"cancel_poll_interval_ms"`
	AlreadyRunningRetryAfterMaxMs int64 `mapstructure:"already_running_retry_after_max_ms"`
}

type TaskSanitizeConfig struct {
	PayloadMaxBytes int `mapstructure:"payload_json_max_bytes"`
	ResultMaxBytes  int `mapstructure:"result_json_max_bytes"`
	ErrorMaxBytes   int `mapstructure:"error_json_max_bytes"`
}

type TaskRetentionConfig struct {
	GCIntervalMs  int64               `mapstructure:"gc_interval_ms"`
	TaskJob       *TaskRetentionEntry `mapstructure:"task_job"`
	TaskExecution *TaskRetentionEntry `mapstructure:"task_job_execution"`
}

type TaskRetentionEntry struct {
	TaskRetentionPolicy `mapstructure:",squash"`
	Overrides           map[string]*TaskRetentionPolicy `mapstructure:"overrides"`
}

type TaskRetentionPolicy struct {
	SucceededDays int `mapstructure:"succeeded_days"`
	FailedDays    int `mapstructure:"failed_days"`
	CancelledDays int `mapstructure:"cancelled_days"`
}

func NewDefaultTaskConfig() *TaskConfig {
	return &TaskConfig{
		Dispatch: &TaskDispatchConfig{
			MaxConcurrency:       20,
			MaxConcurrencyPerApp: 0,
			PollIntervalMs:       5000,
			FetchBatchSize:       100,
			ReadyQueueMax:        1000,
			DefaultMaxAttempts:   1,
			RetryAfterMsCap:      300000,
			BackoffMaxMs:         60000,
			DefaultJobTimeoutMs:  0,
			JobTokenTTLms:        120000,
		},
		Schedule: &TaskScheduleConfig{MaxScheduleEnqueuesPerMinute: 60},
		Worker: &TaskWorkerConfig{
			HeartbeatIntervalMs:           20000,
			LeaseDurationMs:               60000,
			CancelPollIntervalMs:          2000,
			AlreadyRunningRetryAfterMaxMs: 60000,
		},
		Sanitize: &TaskSanitizeConfig{
			PayloadMaxBytes: 16 * 1024,
			ResultMaxBytes:  64 * 1024,
			ErrorMaxBytes:   16 * 1024,
		},
		Retention: &TaskRetentionConfig{
			GCIntervalMs: 3600000,
			TaskJob: &TaskRetentionEntry{
				TaskRetentionPolicy: TaskRetentionPolicy{SucceededDays: 30, FailedDays: 90, CancelledDays: 90},
				Overrides:           map[string]*TaskRetentionPolicy{},
			},
			TaskExecution: &TaskRetentionEntry{
				TaskRetentionPolicy: TaskRetentionPolicy{SucceededDays: 7, FailedDays: 30, CancelledDays: 30},
				Overrides:           map[string]*TaskRetentionPolicy{},
			},
		},
	}
}

func ApplyViperDefaults(v *viper.Viper) {
	if v == nil {
		return
	}

	v.SetDefault("task.dispatch.max_concurrency", 20)
	v.SetDefault("task.dispatch.max_concurrency_per_app", 0)
	v.SetDefault("task.dispatch.default_max_attempts", 1)
	v.SetDefault("task.dispatch.retry_after_ms_cap", 300000)
	v.SetDefault("task.dispatch.backoff_max_ms", 60000)
	v.SetDefault("task.dispatch.default_job_timeout_ms", 0)
	v.SetDefault("task.dispatch.job_token_ttl_ms", 120000)
	v.SetDefault("task.schedule.max_schedule_enqueues_per_minute", 60)
	v.SetDefault("task.worker.cancel_poll_interval_ms", 2000)
	v.SetDefault("task.worker.already_running_retry_after_max_ms", 60000)
	v.SetDefault("task.sanitize.payload_json_max_bytes", 16384)
	v.SetDefault("task.sanitize.result_json_max_bytes", 65536)
	v.SetDefault("task.sanitize.error_json_max_bytes", 16384)
	v.SetDefault("task.retention.gc_interval_ms", 3600000)
	v.SetDefault("task.retention.task_job.succeeded_days", 30)
	v.SetDefault("task.retention.task_job.failed_days", 90)
	v.SetDefault("task.retention.task_job.cancelled_days", 90)
	v.SetDefault("task.retention.task_job.overrides", map[string]any{})
	v.SetDefault("task.retention.task_job_execution.succeeded_days", 7)
	v.SetDefault("task.retention.task_job_execution.failed_days", 30)
	v.SetDefault("task.retention.task_job_execution.cancelled_days", 30)
	v.SetDefault("task.retention.task_job_execution.overrides", map[string]any{})
}

func MergeTaskConfig(cfg *TaskConfig, defaults *TaskConfig) *TaskConfig {
	if defaults == nil {
		return cfg
	}
	if cfg == nil {
		return defaults
	}
	if cfg.Dispatch == nil {
		cfg.Dispatch = defaults.Dispatch
		return cfg
	}
	if cfg.Dispatch.MaxConcurrency <= 0 {
		cfg.Dispatch.MaxConcurrency = defaults.Dispatch.MaxConcurrency
	}
	if cfg.Dispatch.MaxConcurrencyPerApp <= 0 {
		cfg.Dispatch.MaxConcurrencyPerApp = defaults.Dispatch.MaxConcurrencyPerApp
	}
	if cfg.Dispatch.PollIntervalMs <= 0 {
		cfg.Dispatch.PollIntervalMs = defaults.Dispatch.PollIntervalMs
	}
	if cfg.Dispatch.FetchBatchSize <= 0 {
		cfg.Dispatch.FetchBatchSize = defaults.Dispatch.FetchBatchSize
	}
	if cfg.Dispatch.ReadyQueueMax <= 0 {
		cfg.Dispatch.ReadyQueueMax = defaults.Dispatch.ReadyQueueMax
	}
	if cfg.Dispatch.DefaultMaxAttempts <= 0 {
		cfg.Dispatch.DefaultMaxAttempts = defaults.Dispatch.DefaultMaxAttempts
	}
	if cfg.Dispatch.RetryAfterMsCap <= 0 {
		cfg.Dispatch.RetryAfterMsCap = defaults.Dispatch.RetryAfterMsCap
	}
	if cfg.Dispatch.BackoffMaxMs <= 0 {
		cfg.Dispatch.BackoffMaxMs = defaults.Dispatch.BackoffMaxMs
	}
	if cfg.Dispatch.DefaultJobTimeoutMs < 0 {
		cfg.Dispatch.DefaultJobTimeoutMs = defaults.Dispatch.DefaultJobTimeoutMs
	}
	if cfg.Dispatch.JobTokenTTLms <= 0 {
		cfg.Dispatch.JobTokenTTLms = defaults.Dispatch.JobTokenTTLms
	}
	if cfg.Schedule == nil {
		cfg.Schedule = defaults.Schedule
	}
	if cfg.Schedule.MaxScheduleEnqueuesPerMinute <= 0 {
		cfg.Schedule.MaxScheduleEnqueuesPerMinute = defaults.Schedule.MaxScheduleEnqueuesPerMinute
	}
	if cfg.Worker == nil {
		cfg.Worker = defaults.Worker
		return cfg
	}
	if cfg.Worker.HeartbeatIntervalMs <= 0 {
		cfg.Worker.HeartbeatIntervalMs = defaults.Worker.HeartbeatIntervalMs
	}
	if cfg.Worker.LeaseDurationMs <= 0 {
		cfg.Worker.LeaseDurationMs = defaults.Worker.LeaseDurationMs
	}
	if cfg.Worker.CancelPollIntervalMs <= 0 {
		cfg.Worker.CancelPollIntervalMs = defaults.Worker.CancelPollIntervalMs
	}
	if cfg.Worker.AlreadyRunningRetryAfterMaxMs <= 0 {
		cfg.Worker.AlreadyRunningRetryAfterMaxMs = defaults.Worker.AlreadyRunningRetryAfterMaxMs
	}
	if cfg.Sanitize == nil {
		cfg.Sanitize = defaults.Sanitize
	} else {
		if cfg.Sanitize.PayloadMaxBytes <= 0 {
			cfg.Sanitize.PayloadMaxBytes = defaults.Sanitize.PayloadMaxBytes
		}
		if cfg.Sanitize.ResultMaxBytes <= 0 {
			cfg.Sanitize.ResultMaxBytes = defaults.Sanitize.ResultMaxBytes
		}
		if cfg.Sanitize.ErrorMaxBytes <= 0 {
			cfg.Sanitize.ErrorMaxBytes = defaults.Sanitize.ErrorMaxBytes
		}
	}
	if cfg.Retention == nil {
		cfg.Retention = defaults.Retention
		return cfg
	}
	if cfg.Retention.GCIntervalMs <= 0 {
		cfg.Retention.GCIntervalMs = defaults.Retention.GCIntervalMs
	}
	if cfg.Retention.TaskJob == nil {
		cfg.Retention.TaskJob = defaults.Retention.TaskJob
	} else {
		cfg.Retention.TaskJob = mergeRetentionEntry(cfg.Retention.TaskJob, defaults.Retention.TaskJob)
	}
	if cfg.Retention.TaskExecution == nil {
		cfg.Retention.TaskExecution = defaults.Retention.TaskExecution
	} else {
		cfg.Retention.TaskExecution = mergeRetentionEntry(cfg.Retention.TaskExecution, defaults.Retention.TaskExecution)
	}
	return cfg
}

func mergeRetentionEntry(entry *TaskRetentionEntry, defaults *TaskRetentionEntry) *TaskRetentionEntry {
	if defaults == nil {
		return entry
	}
	if entry == nil {
		return defaults
	}
	entry.TaskRetentionPolicy = *mergeRetentionPolicy(&entry.TaskRetentionPolicy, &defaults.TaskRetentionPolicy)
	if entry.Overrides == nil {
		entry.Overrides = map[string]*TaskRetentionPolicy{}
	}
	for k, v := range entry.Overrides {
		entry.Overrides[k] = mergeRetentionPolicy(v, &defaults.TaskRetentionPolicy)
	}
	return entry
}

func mergeRetentionPolicy(policy *TaskRetentionPolicy, defaults *TaskRetentionPolicy) *TaskRetentionPolicy {
	if defaults == nil {
		return policy
	}
	if policy == nil {
		return defaults
	}
	if policy.SucceededDays <= 0 {
		policy.SucceededDays = defaults.SucceededDays
	}
	if policy.FailedDays <= 0 {
		policy.FailedDays = defaults.FailedDays
	}
	if policy.CancelledDays <= 0 {
		policy.CancelledDays = defaults.CancelledDays
	}
	return policy
}
