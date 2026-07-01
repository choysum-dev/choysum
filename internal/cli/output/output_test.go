// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package output

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestPrintErrorBlock_OutputFormat(t *testing.T) {
	output := captureStderr(t, func() {
		PrintErrorBlock("failed", "because", "do next")
	})

	expected := "ERROR: failed\nREASON: because\nNEXT: do next\n"
	if output != expected {
		t.Fatalf("unexpected output:\n%s", output)
	}
}

func TestPrintWarning_OutputFormat(t *testing.T) {
	output := captureStderr(t, func() {
		PrintWarning("check config")
	})

	expected := "WARN: check config\n"
	if output != expected {
		t.Fatalf("unexpected output:\n%s", output)
	}
}

func TestPrintError_OutputFormat(t *testing.T) {
	output := captureStderr(t, func() {
		PrintError("scope is not initialized")
	})

	expected := "ERROR: scope is not initialized\n"
	if output != expected {
		t.Fatalf("unexpected output:\n%s", output)
	}
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stderr
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = writer

	fn()
	os.Stderr = old
	_ = writer.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		t.Fatalf("read stderr: %v", err)
	}
	return buf.String()
}
