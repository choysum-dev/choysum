// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"fmt"
	"os"
)

func printErrorBlock(errMsg, reason, next string) {
	printCLIOutputLine("ERROR", errMsg)
	printCLIOutputLine("REASON", reason)
	printCLIOutputLine("NEXT", next)
}

func printCLIWarning(message string) {
	printCLIOutputLine("WARN", message)
}

func printCLIError(message string) {
	printCLIOutputLine("ERROR", message)
}

func printCLIOutputLine(label, message string) {
	fmt.Fprintf(os.Stderr, "%s: %s\n", label, message)
}
