// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package output

import (
	"fmt"
	"os"
)

func PrintErrorBlock(errMsg, reason, next string) {
	PrintLine("ERROR", errMsg)
	PrintLine("REASON", reason)
	PrintLine("NEXT", next)
}

func PrintWarning(message string) {
	PrintLine("WARN", message)
}

func PrintError(message string) {
	PrintLine("ERROR", message)
}

func PrintLine(label, message string) {
	fmt.Fprintf(os.Stderr, "%s: %s\n", label, message)
}
