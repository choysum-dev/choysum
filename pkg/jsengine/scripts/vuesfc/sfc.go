// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vuesfc

import (
	_ "embed"
)

//go:generate go run gen.go

//go:embed dist/index.js
var VueSfcScript string
