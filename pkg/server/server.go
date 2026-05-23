// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package server

import "context"

type Server interface {
	Serve(ctx context.Context, services ...string) error
}
