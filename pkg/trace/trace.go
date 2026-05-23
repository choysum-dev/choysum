// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package trace

import (
	"context"

	"google.golang.org/grpc"
)

type Telemetry interface {
	ServerOptions() []grpc.ServerOption
	Shutdown(ctx context.Context) error
}
