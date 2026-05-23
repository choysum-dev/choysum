// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package client

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ToStatusError(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := status.FromError(err); ok {
		return err
	}

	var invalid *InvalidServiceNameError
	if errors.As(err, &invalid) {
		return status.Error(codes.InvalidArgument, invalid.Error())
	}

	var missing *MissingServiceDialerError
	if errors.As(err, &missing) {
		return status.Error(codes.FailedPrecondition, missing.Error())
	}

	var full *ConnCacheFullError
	if errors.As(err, &full) {
		return status.Error(codes.ResourceExhausted, full.Error())
	}

	return status.Error(codes.Unknown, err.Error())
}
