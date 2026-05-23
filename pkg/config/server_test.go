// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import "testing"

func TestNewDefaultServerConfig(t *testing.T) {
	cfg := NewDefaultServerConfig()
	if cfg.BindAddress != "0.0.0.0" || cfg.Port != 9527 {
		t.Fatalf("unexpected bind defaults: %#v", cfg)
	}
	if !cfg.EnableGzip || cfg.EnabledTLS || !cfg.EnableGrpcWebProxy || cfg.HotReload {
		t.Fatalf("unexpected server booleans: %#v", cfg)
	}
	if cfg.Register != "local" || cfg.Environment != "default" || cfg.RuntimeEngine != "default" || cfg.WebBaseURL != "/web" || cfg.JsEngineFactory != "quickjs" || cfg.JsExecutorFactory != "default" {
		t.Fatalf("unexpected server string defaults: %#v", cfg)
	}
	if cfg.Security == nil || cfg.GrpcClient == nil {
		t.Fatalf("expected nested defaults: %#v", cfg)
	}
	if cfg.GrpcClient.MaxCachedConns != 128 {
		t.Fatalf("unexpected grpc client defaults: %#v", cfg.GrpcClient)
	}
}
