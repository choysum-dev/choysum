// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import (
	taskconfig "github.com/choysum-dev/choysum/internal/task/taskconfig"
	"github.com/spf13/viper"
)

type TaskConfig = taskconfig.TaskConfig
type TaskDispatchConfig = taskconfig.TaskDispatchConfig
type TaskScheduleConfig = taskconfig.TaskScheduleConfig
type TaskWorkerConfig = taskconfig.TaskWorkerConfig
type TaskSanitizeConfig = taskconfig.TaskSanitizeConfig
type TaskRetentionConfig = taskconfig.TaskRetentionConfig
type TaskRetentionEntry = taskconfig.TaskRetentionEntry
type TaskRetentionPolicy = taskconfig.TaskRetentionPolicy

func NewDefaultTaskConfig() *TaskConfig {
	return taskconfig.NewDefaultTaskConfig()
}

func applyTaskViperDefaults(v *viper.Viper) {
	taskconfig.ApplyViperDefaults(v)
}

func (c *Config) normalizeAndMergeTaskConfig() {
	if c == nil {
		return
	}
	if c.Task == nil {
		c.Task = NewDefaultTaskConfig()
		return
	}
	c.Task = MergeTaskConfig(c.Task, NewDefaultTaskConfig())
}

func MergeTaskConfig(cfg *TaskConfig, defaults *TaskConfig) *TaskConfig {
	return taskconfig.MergeTaskConfig(cfg, defaults)
}
