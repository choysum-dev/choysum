// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package pagehost

import (
	"os"
	"testing"

	"github.com/choysum-dev/choysum/internal/testing/e2e/cdp"
)

func TestMain(m *testing.M) {
	code := m.Run()
	cdp.CloseSharedTestSession()
	os.Exit(code)
}
