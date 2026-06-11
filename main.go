// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/choysum-dev/choysum/cmd"
	_ "github.com/choysum-dev/choysum/internal/initialize"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"

	readBuildInfo = debug.ReadBuildInfo
)

func shortCommitHash(value string) string {
	if len(value) > 7 {
		return value[:7]
	}
	return value
}

func getBuildVersion() string {
	v := version
	if version == "dev" {
		if info, ok := readBuildInfo(); ok {
			for _, setting := range info.Settings {
				if setting.Key == "vcs.revision" {
					commit = shortCommitHash(setting.Value)
				}
				if setting.Key == "vcs.time" {
					date = setting.Value
				}
			}
			v = fmt.Sprintf("dev-%s", commit)
		}
	}
	return fmt.Sprintf("%s (commit: %s, date: %s)", v, commit, date)
}

var newCommander = func(ctx context.Context) interface{ Execute() error } {
	return cmd.NewCommander(ctx, getBuildVersion())
}

var exitFunc = os.Exit

func main() {
	command := newCommander(context.Background())
	if err := command.Execute(); err != nil {
		exitFunc(1)
	}
}
