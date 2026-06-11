//go:build !windows

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package e2e

import (
	"os/exec"
	"syscall"
)

func setServerProcessAttrs(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func signalServerProcess(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	if cmd.Process.Pid > 0 {
		if cmd.SysProcAttr != nil && cmd.SysProcAttr.Setpgid {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
			return
		}
		_ = cmd.Process.Signal(syscall.SIGTERM)
	}
}
