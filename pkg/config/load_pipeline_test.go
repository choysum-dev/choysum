// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestNewConfigStageDecodeFailure(t *testing.T) {
	_, err := NewConfig(filepath.Join(t.TempDir(), "missing-config.yaml"))
	if err == nil {
		t.Fatal("expected decode stage error for missing config file")
	}
	if !IsLoadStage(err, LoadStageDecode) {
		t.Fatalf("expected decode stage error, got %v", err)
	}
	if !strings.Contains(err.Error(), "read config file failed") {
		t.Fatalf("unexpected decode error message: %v", err)
	}
}

func TestNewConfigWithoutConfigFileUsesDefaults(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("CHOYSUM_DEFAULT_CHOYSUM_PATH", "")
	t.Setenv("CHOYSUM_DB_DIALECT", "")
	t.Setenv("CHOYSUM_DB_DSN", "")
	t.Setenv("CHOYSUM_AUTH_INTERNAL_KEY", "dev-internal-key")

	cfg, err := NewConfig("")
	if err != nil {
		t.Fatalf("NewConfig(\"\") returned error: %v", err)
	}

	wantDefaultChoysumPath, _ := filepath.Abs(filepath.Join(homeDir, ".choysum"))
	if filepath.Clean(cfg.DefaultChoysumPath) != filepath.Clean(wantDefaultChoysumPath) {
		t.Fatalf("default_choysum_path = %q, want %q", cfg.DefaultChoysumPath, wantDefaultChoysumPath)
	}
	wantDistPath, _ := filepath.Abs(filepath.Join(homeDir, ".choysum", "dist"))
	if filepath.Clean(cfg.DistPath) != filepath.Clean(wantDistPath) {
		t.Fatalf("dist_path = %q, want %q", cfg.DistPath, wantDistPath)
	}
	if got, want := cfg.Db.Dialect, "sqlite"; got != want {
		t.Fatalf("db.dialect = %q, want %q", got, want)
	}
	wantDBPath, _ := filepath.Abs(filepath.Join(homeDir, ".choysum", "choysum.sqlite"))
	if filepath.Clean(cfg.Db.DSN) != filepath.Clean(wantDBPath) {
		t.Fatalf("db.dsn = %q, want %q", cfg.Db.DSN, wantDBPath)
	}
	if got, want := cfg.Auth.InternalKey, "dev-internal-key"; got != want {
		t.Fatalf("auth.internalKey = %q, want %q", got, want)
	}
}

func TestNewConfigStageValidateFailure(t *testing.T) {
	cfgPath := writeTestConfig(t, strings.Join([]string{
		"default_choysum_path: ./.choysum-bootstrap",
		"compile:",
		"  bundleMode: invalid",
	}, "\n"))

	_, err := NewConfig(cfgPath)
	if err == nil {
		t.Fatal("expected validate stage error")
	}
	if !IsLoadStage(err, LoadStageValidate) {
		t.Fatalf("expected validate stage error, got %v", err)
	}
	if !strings.Contains(err.Error(), "invalid compile.bundleMode") {
		t.Fatalf("unexpected validate error message: %v", err)
	}
}

func TestNewConfigStageApplyFailure(t *testing.T) {
	cfgPath := writeTestConfig(t, "default_choysum_path: ./.choysum-bootstrap")

	_, err := NewConfig(cfgPath, AfterUnmarshal(func(*viper.Viper, *Config) error {
		return errors.New("apply hook failed")
	}))
	if err == nil {
		t.Fatal("expected apply stage error")
	}
	if !IsLoadStage(err, LoadStageApply) {
		t.Fatalf("expected apply stage error, got %v", err)
	}
	if !strings.Contains(err.Error(), "apply hook failed") {
		t.Fatalf("unexpected apply error message: %v", err)
	}
}
