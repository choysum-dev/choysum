// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import (
	"testing"
)

type providerSpy struct {
	called     bool
	configPath string
	optsCount  int
	cfg        *Config
	err        error
}

func (p *providerSpy) Load(configPath string, opts ...Option) (*Config, error) {
	p.called = true
	p.configPath = configPath
	p.optsCount = len(opts)
	if p.err != nil {
		return nil, p.err
	}
	if p.cfg != nil {
		return p.cfg, nil
	}
	return &Config{}, nil
}

func TestLoadWithProviderUsesCustomProvider(t *testing.T) {
	spy := &providerSpy{cfg: &Config{AddonsPath: "/tmp/addons"}}

	cfg, err := LoadWithProvider(spy, "/tmp/config.yaml", WithEnvPrefix("CHOYSUM_TEST"))
	if err != nil {
		t.Fatalf("LoadWithProvider returned error: %v", err)
	}
	if !spy.called {
		t.Fatal("expected custom provider to be called")
	}
	if spy.configPath != "/tmp/config.yaml" {
		t.Fatalf("provider config path = %q, want /tmp/config.yaml", spy.configPath)
	}
	if spy.optsCount != 1 {
		t.Fatalf("provider opts count = %d, want 1", spy.optsCount)
	}
	if cfg != spy.cfg {
		t.Fatal("expected config pointer returned by provider")
	}
}

func TestLoadWithProviderNilUsesFileProvider(t *testing.T) {
	cfgPath := writeTestConfig(t, "default_choysum_path: ./.choysum-bootstrap")

	cfg, err := LoadWithProvider(nil, cfgPath)
	if err != nil {
		t.Fatalf("LoadWithProvider(nil) returned error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config from default file provider")
	}
}
