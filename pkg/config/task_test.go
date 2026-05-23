// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import "testing"

func TestNewDefaultTaskConfig(t *testing.T) {
	cfg := NewDefaultTaskConfig()
	if cfg.Dispatch == nil || cfg.Schedule == nil || cfg.Worker == nil || cfg.Sanitize == nil || cfg.Retention == nil {
		t.Fatalf("expected nested task defaults: %#v", cfg)
	}
	if cfg.Dispatch.MaxConcurrency != 20 || cfg.Dispatch.FetchBatchSize != 100 || cfg.Dispatch.JobTokenTTLms != 120000 {
		t.Fatalf("unexpected dispatch defaults: %#v", cfg.Dispatch)
	}
	if cfg.Schedule.MaxScheduleEnqueuesPerMinute != 60 {
		t.Fatalf("unexpected schedule defaults: %#v", cfg.Schedule)
	}
	if cfg.Worker.HeartbeatIntervalMs != 20000 || cfg.Worker.LeaseDurationMs != 60000 {
		t.Fatalf("unexpected worker defaults: %#v", cfg.Worker)
	}
	if cfg.Sanitize.PayloadMaxBytes != 16*1024 || cfg.Sanitize.ResultMaxBytes != 64*1024 || cfg.Sanitize.ErrorMaxBytes != 16*1024 {
		t.Fatalf("unexpected sanitize defaults: %#v", cfg.Sanitize)
	}
	if cfg.Retention.GCIntervalMs != 3600000 || cfg.Retention.TaskJob == nil || cfg.Retention.TaskExecution == nil {
		t.Fatalf("unexpected retention defaults: %#v", cfg.Retention)
	}
}

func TestMergeTaskConfig(t *testing.T) {
	defaults := NewDefaultTaskConfig()

	t.Run("returns defaults when cfg is nil", func(t *testing.T) {
		merged := MergeTaskConfig(nil, defaults)
		if merged != defaults {
			t.Fatal("expected defaults pointer when cfg is nil")
		}
	})

	t.Run("returns cfg when defaults is nil", func(t *testing.T) {
		cfg := &TaskConfig{}
		merged := MergeTaskConfig(cfg, nil)
		if merged != cfg {
			t.Fatal("expected cfg pointer when defaults is nil")
		}
	})

	t.Run("fills nested defaults and zero values", func(t *testing.T) {
		cfg := &TaskConfig{
			Dispatch: &TaskDispatchConfig{MaxConcurrency: 5, MaxConcurrencyPerApp: 0, PollIntervalMs: 0, FetchBatchSize: 0, ReadyQueueMax: 0, DefaultMaxAttempts: 0, RetryAfterMsCap: 0, BackoffMaxMs: 0, DefaultJobTimeoutMs: -1, JobTokenTTLms: 0},
			Schedule: &TaskScheduleConfig{},
			Worker:   &TaskWorkerConfig{HeartbeatIntervalMs: 0, LeaseDurationMs: 0, CancelPollIntervalMs: 0, AlreadyRunningRetryAfterMaxMs: 0},
			Sanitize: &TaskSanitizeConfig{},
			Retention: &TaskRetentionConfig{
				GCIntervalMs: 0,
				TaskJob: &TaskRetentionEntry{
					TaskRetentionPolicy: TaskRetentionPolicy{SucceededDays: 0, FailedDays: 1, CancelledDays: 0},
					Overrides: map[string]*TaskRetentionPolicy{
						"app": {SucceededDays: 0, FailedDays: 0, CancelledDays: 9},
					},
				},
			},
		}

		merged := MergeTaskConfig(cfg, defaults)
		if merged.Dispatch.MaxConcurrency != 5 {
			t.Fatalf("expected explicit dispatch max concurrency to survive, got %d", merged.Dispatch.MaxConcurrency)
		}
		if merged.Dispatch.MaxConcurrencyPerApp != defaults.Dispatch.MaxConcurrencyPerApp || merged.Dispatch.PollIntervalMs != defaults.Dispatch.PollIntervalMs || merged.Dispatch.DefaultJobTimeoutMs != defaults.Dispatch.DefaultJobTimeoutMs {
			t.Fatalf("expected zero/negative dispatch fields to merge defaults, got %#v", merged.Dispatch)
		}
		if merged.Schedule.MaxScheduleEnqueuesPerMinute != defaults.Schedule.MaxScheduleEnqueuesPerMinute {
			t.Fatalf("expected schedule defaults, got %#v", merged.Schedule)
		}
		if merged.Worker.HeartbeatIntervalMs != defaults.Worker.HeartbeatIntervalMs || merged.Worker.AlreadyRunningRetryAfterMaxMs != defaults.Worker.AlreadyRunningRetryAfterMaxMs {
			t.Fatalf("expected worker defaults, got %#v", merged.Worker)
		}
		if merged.Sanitize.PayloadMaxBytes != defaults.Sanitize.PayloadMaxBytes || merged.Sanitize.ResultMaxBytes != defaults.Sanitize.ResultMaxBytes || merged.Sanitize.ErrorMaxBytes != defaults.Sanitize.ErrorMaxBytes {
			t.Fatalf("expected sanitize defaults, got %#v", merged.Sanitize)
		}
		if merged.Retention.GCIntervalMs != defaults.Retention.GCIntervalMs {
			t.Fatalf("expected retention GC defaults, got %#v", merged.Retention)
		}
		if merged.Retention.TaskExecution != defaults.Retention.TaskExecution {
			t.Fatalf("expected nil TaskExecution to adopt default pointer, got %#v", merged.Retention.TaskExecution)
		}
		if merged.Retention.TaskJob.SucceededDays != defaults.Retention.TaskJob.SucceededDays || merged.Retention.TaskJob.FailedDays != 1 || merged.Retention.TaskJob.CancelledDays != defaults.Retention.TaskJob.CancelledDays {
			t.Fatalf("expected task job policy merge, got %#v", merged.Retention.TaskJob)
		}
		if merged.Retention.TaskJob.Overrides["app"].SucceededDays != defaults.Retention.TaskJob.SucceededDays || merged.Retention.TaskJob.Overrides["app"].FailedDays != defaults.Retention.TaskJob.FailedDays || merged.Retention.TaskJob.Overrides["app"].CancelledDays != 9 {
			t.Fatalf("expected override policy merge, got %#v", merged.Retention.TaskJob.Overrides["app"])
		}
	})
}

func TestMergeRetentionHelpers(t *testing.T) {
	defaults := NewDefaultTaskConfig()

	t.Run("nil entry falls back to defaults", func(t *testing.T) {
		cfg := &TaskConfig{
			Dispatch:  &TaskDispatchConfig{},
			Worker:    &TaskWorkerConfig{},
			Retention: &TaskRetentionConfig{TaskJob: nil},
		}
		merged := MergeTaskConfig(cfg, defaults)
		if merged.Retention.TaskJob != defaults.Retention.TaskJob {
			t.Fatalf("expected nil task job to adopt default pointer, got %#v", merged.Retention.TaskJob)
		}
	})

	t.Run("entry merge initializes overrides", func(t *testing.T) {
		cfg := &TaskConfig{
			Dispatch:  &TaskDispatchConfig{},
			Worker:    &TaskWorkerConfig{},
			Retention: &TaskRetentionConfig{TaskJob: &TaskRetentionEntry{}},
		}
		merged := MergeTaskConfig(cfg, defaults)
		if merged.Retention.TaskJob.Overrides == nil {
			t.Fatal("expected task job overrides map to be initialized")
		}
	})

	t.Run("policy merge keeps explicit values and fills defaults", func(t *testing.T) {
		cfg := &TaskConfig{
			Dispatch: &TaskDispatchConfig{},
			Worker:   &TaskWorkerConfig{},
			Retention: &TaskRetentionConfig{TaskJob: &TaskRetentionEntry{TaskRetentionPolicy: TaskRetentionPolicy{
				SucceededDays: 0,
				FailedDays:    4,
				CancelledDays: 0,
			}}},
		}
		merged := MergeTaskConfig(cfg, defaults)
		if merged.Retention.TaskJob.SucceededDays != defaults.Retention.TaskJob.SucceededDays || merged.Retention.TaskJob.FailedDays != 4 || merged.Retention.TaskJob.CancelledDays != defaults.Retention.TaskJob.CancelledDays {
			t.Fatalf("unexpected merged retention policy: %#v", merged.Retention.TaskJob)
		}
	})
}
