// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"context"
	"fmt"
	"io"
	"log"
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
	c := commit
	d := date
	if version == "dev" {
		if info, ok := readBuildInfo(); ok && info != nil {
			revision := ""
			dirty := false
			for _, setting := range info.Settings {
				if setting.Key == "vcs.revision" {
					revision = shortCommitHash(setting.Value)
				}
				if setting.Key == "vcs.time" {
					d = setting.Value
				}
				if setting.Key == "vcs.modified" && setting.Value == "true" {
					dirty = true
				}
			}
			if info.Main.Version != "" && info.Main.Version != "(devel)" {
				v = info.Main.Version
				if revision != "" {
					c = revision
				}
			} else if revision != "" {
				c = revision
				v = fmt.Sprintf("dev-%s", revision)
				if dirty {
					v += "-dirty"
				}
			}
		}
	}
	return fmt.Sprintf("%s (commit: %s, date: %s)", v, c, d)
}

var newCommander = func(ctx context.Context) interface{ Execute() error } {
	return cmd.NewCommander(ctx, getBuildVersion())
}

var exitFunc = os.Exit

func main() {
	// Suppress standard-library log output from third-party packages (e.g.
	// x/net/http2 unconditionally writes protocol errors to stderr via log.Printf).
	// Choysum does not use the standard log package; all application logging
	// goes through internal/logger.
	//
	// NOTE: This is process-wide — any dependency that legitimately logs via
	// the stdlib log package will also be silenced. This is an intentional
	// tradeoff: there is no way to scope suppression to http2 alone.
	log.SetOutput(io.Discard)

	command := newCommander(context.Background())
	if err := command.Execute(); err != nil {
		exitFunc(1)
	}
}
