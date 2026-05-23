// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package client

import (
	"strings"
	"testing"
)

func TestClientErrorFormattingHelpers(t *testing.T) {
	long := strings.Repeat("x", MaxServiceNameEchoLen+10)
	if got := truncateForEcho(long); len(got) != MaxServiceNameEchoLen {
		t.Fatalf("truncate len = %d, want %d", len(got), MaxServiceNameEchoLen)
	}
	if got := truncateForEcho("short"); got != "short" {
		t.Fatalf("unexpected short truncate result: %q", got)
	}
	if got := (&InvalidServiceNameError{ServiceName: long}).Error(); !strings.Contains(got, truncateForEcho(long)) {
		t.Fatalf("invalid service name error = %q, want truncated service name", got)
	}
	if got := (&MissingServiceDialerError{}).Error(); got == "" {
		t.Fatal("expected missing dialer error message")
	}
	if got := (&ConnCacheFullError{ServiceName: long, Max: 1, Current: 2}).Error(); !strings.Contains(got, truncateForEcho(long)) || !strings.Contains(got, "max=1") {
		t.Fatalf("unexpected conn cache full error = %q", got)
	}
}
