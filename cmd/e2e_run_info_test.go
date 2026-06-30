// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"
	"testing"
)

func TestCLIRunInfoShowsActualAddress(t *testing.T) {
	configPath, addr, _ := writeTempInitializedRunConfig(t, false)
	expected := fmt.Sprintf("http://%s", addr)

	output, _ := runCLIUntilLine(t, func(line string) bool {
		return strings.Contains(line, "http server listening")
	}, "run", "--config", configPath)
	if strings.Contains(output, "server starting; NEXT: open") {
		t.Fatalf("did not expect CLI startup hint, got %s", output)
	}
	if !strings.Contains(output, expected) {
		t.Fatalf("expected output to contain %q, got %s", expected, output)
	}
}
