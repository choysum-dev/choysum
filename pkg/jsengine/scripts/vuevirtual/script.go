// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

// Package vuevirtual embeds the language-core createServiceScript IIFE for
// Go-native Vue typecheck (QuickJSCoder).
package vuevirtual

import (
	_ "embed"
)

//go:generate go run gen.go

//go:embed dist/index.js
var VueVirtualScript string
