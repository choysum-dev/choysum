//go:build windows

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package e2e

import (
	"os/exec"
)

func setServerProcessAttrs(*exec.Cmd) {}

func signalServerProcess(cmd *exec.Cmd) {
	_ = cmd.Process.Kill()
}
