// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"bytes"
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

// http2LogFilter drops log lines produced by x/net/http2 that unconditionally
// write protocol errors to stderr (e.g. "received DATA after END_STREAM").
// All other log output is forwarded to the underlying writer unchanged.
type http2LogFilter struct {
	w io.Writer
}

func (f *http2LogFilter) Write(p []byte) (int, error) {
	if bytes.Contains(p, []byte("protocol error: received DATA after END_STREAM")) {
		return len(p), nil
	}
	if f.w == nil {
		return len(p), nil
	}
	return f.w.Write(p)
}

var exitFunc = os.Exit

func main() {
	// Filter out x/net/http2 protocol-error noise from the standard library
	// logger while preserving legitimate log output from other dependencies.
	// Choysum does not use the standard log package; all application logging
	// goes through internal/logger.
	log.SetOutput(&http2LogFilter{w: os.Stderr})

	command := newCommander(context.Background())
	if err := command.Execute(); err != nil {
		exitFunc(1)
	}
}
