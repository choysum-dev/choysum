// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package client

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
)

func Dial(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
	if err := ValidateServiceName(serviceName); err != nil {
		return nil, err
	}

	dialer, ok := ServiceDialerFromContext(ctx)
	if !ok {
		return nil, &MissingServiceDialerError{}
	}

	cc, err := dialer(ctx, serviceName)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", serviceName, err)
	}
	return cc, nil
}
