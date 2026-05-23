// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import (
	"testing"

	"github.com/spf13/viper"
)

func TestOptionHelpers(t *testing.T) {
	var cfg Config
	v := viper.New()

	if err := (Option{}).applyPre(v, &cfg); err != nil {
		t.Fatalf("empty applyPre() error = %v", err)
	}
	if err := (Option{}).applyPost(v, &cfg); err != nil {
		t.Fatalf("empty applyPost() error = %v", err)
	}

	WithDefaults(func(cfg *Config) {
		cfg.AddonsPath = "/tmp/addons"
	}).applyPre(v, &cfg)
	if cfg.AddonsPath != "/tmp/addons" {
		t.Fatalf("WithDefaults did not apply override, got %q", cfg.AddonsPath)
	}

	WithDefaults(nil).applyPre(v, &cfg)
	WithViper(func(v *viper.Viper) {
		v.Set("custom.enabled", true)
	}).applyPre(v, &cfg)
	if !v.GetBool("custom.enabled") {
		t.Fatal("WithViper did not mutate viper instance")
	}

	WithEnvPrefix("CHOYSUM_TEST").applyPre(v, &cfg)
	if got := v.GetEnvPrefix(); got != "CHOYSUM_TEST" {
		t.Fatalf("env prefix = %q, want %q", got, "CHOYSUM_TEST")
	}
	WithEnvPrefix("").applyPre(v, &cfg)
	if got := v.GetEnvPrefix(); got != "CHOYSUM_TEST" {
		t.Fatalf("empty env prefix should preserve existing prefix, got %q", got)
	}

	type custom struct {
		Enabled bool `mapstructure:"enabled"`
	}
	v.Set("custom", map[string]any{"enabled": true})
	var section custom
	if err := UnmarshalKey("custom", &section).applyPost(v, &cfg); err != nil {
		t.Fatalf("UnmarshalKey.applyPost() error = %v", err)
	}
	if !section.Enabled {
		t.Fatal("expected custom section to unmarshal")
	}
	if err := UnmarshalKey("", &section).applyPost(v, &cfg); err != nil {
		t.Fatalf("empty key UnmarshalKey.applyPost() error = %v", err)
	}
	if err := UnmarshalKey[custom]("custom", nil).applyPost(v, &cfg); err != nil {
		t.Fatalf("nil out UnmarshalKey.applyPost() error = %v", err)
	}

	called := false
	if err := AfterUnmarshal(func(v *viper.Viper, cfg *Config) error {
		called = v.GetBool("custom.enabled")
		cfg.DistPath = "/tmp/dist"
		return nil
	}).applyPost(v, &cfg); err != nil {
		t.Fatalf("AfterUnmarshal.applyPost() error = %v", err)
	}
	if !called || cfg.DistPath != "/tmp/dist" {
		t.Fatalf("unexpected AfterUnmarshal result: called=%v dist=%q", called, cfg.DistPath)
	}
}
