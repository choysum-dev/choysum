// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestPrintErrorBlock_OutputFormat(t *testing.T) {
	output := captureStderr(t, func() {
		printErrorBlock("failed", "because", "do next")
	})

	expected := "ERROR: failed\nREASON: because\nNEXT: do next\n"
	if output != expected {
		t.Fatalf("unexpected output:\n%s", output)
	}
}

func TestPrintCLIWarning_OutputFormat(t *testing.T) {
	output := captureStderr(t, func() {
		printCLIWarning("check config")
	})

	expected := "WARN: check config\n"
	if output != expected {
		t.Fatalf("unexpected output:\n%s", output)
	}
}

func TestPrintCLIError_OutputFormat(t *testing.T) {
	output := captureStderr(t, func() {
		printCLIError("scope is not initialized")
	})

	expected := "ERROR: scope is not initialized\n"
	if output != expected {
		t.Fatalf("unexpected output:\n%s", output)
	}
}

func captureStderr(t *testing.T, fn func()) string {
	old := os.Stderr
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = writer
	t.Cleanup(func() {
		os.Stderr = old
	})

	fn()
	_ = writer.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		t.Fatalf("read stderr: %v", err)
	}
	return buf.String()
}
