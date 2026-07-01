// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package runtime

import (
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/config"
)

func TestCloneLogConfig(t *testing.T) {
	t.Parallel()

	defaultCfg := CloneLogConfig(nil)
	if defaultCfg == nil {
		t.Fatal("CloneLogConfig(nil) = nil, want default config")
	}

	source := &config.LogConfig{Level: "info"}
	cloned := CloneLogConfig(source)
	if cloned == source {
		t.Fatal("CloneLogConfig() returned the same pointer")
	}
	cloned.Level = "debug"
	if source.Level != "info" {
		t.Fatalf("CloneLogConfig() leaked mutation, source.Level = %q", source.Level)
	}
}

func TestNormalizeRuntimeLogLevelFlag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		command   string
		want      string
		wantError string
	}{
		{name: "default warn", input: "   ", command: "test", want: "warn"},
		{name: "normalize upper", input: " INFO ", command: "run", want: "info"},
		{name: "debug", input: "debug", command: "run", want: "debug"},
		{name: "invalid", input: "trace", command: "test", wantError: "invalid --runtime-log-level"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeRuntimeLogLevelFlag(tt.input, tt.command)
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("NormalizeRuntimeLogLevelFlag(%q) error = %v, want contains %q", tt.input, err, tt.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeRuntimeLogLevelFlag(%q) error = %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("NormalizeRuntimeLogLevelFlag(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
