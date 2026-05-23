// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"context"
	"os"

	"github.com/choysum-dev/choysum/cmd"
	_ "github.com/choysum-dev/choysum/internal/initialize"
)

var newCommander = func(ctx context.Context) interface{ Execute() error } {
	return cmd.NewCommander(ctx)
}

var exitFunc = os.Exit

func main() {
	command := newCommander(context.Background())
	if err := command.Execute(); err != nil {
		exitFunc(1)
	}
}
