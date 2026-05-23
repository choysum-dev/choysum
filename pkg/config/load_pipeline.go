// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import "errors"

// LoadStage defines the static config loading stages.
type LoadStage string

const (
	LoadStageDecode   LoadStage = "decode"
	LoadStageValidate LoadStage = "validate"
	LoadStageApply    LoadStage = "apply"
)

// LoadStageError marks which stage failed during static config loading.
type LoadStageError struct {
	Stage LoadStage
	Err   error
}

func (e *LoadStageError) Error() string {
	if e == nil || e.Err == nil {
		return "config load failed"
	}
	return string(e.Stage) + " stage failed: " + e.Err.Error()
}

func (e *LoadStageError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func stageError(stage LoadStage, err error) error {
	if err == nil {
		return nil
	}
	var existing *LoadStageError
	if errors.As(err, &existing) {
		return err
	}
	return &LoadStageError{Stage: stage, Err: err}
}

// IsLoadStage reports whether err originates from the given static load stage.
func IsLoadStage(err error, stage LoadStage) bool {
	var target *LoadStageError
	if !errors.As(err, &target) {
		return false
	}
	return target != nil && target.Stage == stage
}
