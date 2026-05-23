// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package client

import (
	"fmt"
)

const (
	MaxServiceNameLen     = 256
	MaxServiceNameEchoLen = 128
)

type InvalidServiceNameError struct {
	ServiceName string
}

func (e *InvalidServiceNameError) Error() string {
	return fmt.Sprintf("invalid service name: %q", truncateForEcho(e.ServiceName))
}

type MissingServiceDialerError struct{}

func (e *MissingServiceDialerError) Error() string {
	return "missing grpc service dialer in context"
}

type ConnCacheFullError struct {
	ServiceName string
	Max         int
	Current     int
}

func (e *ConnCacheFullError) Error() string {
	return fmt.Sprintf(
		"grpc client conn cache full for service %q (max=%d current=%d)",
		truncateForEcho(e.ServiceName),
		e.Max,
		e.Current,
	)
}

func truncateForEcho(s string) string {
	if len(s) <= MaxServiceNameEchoLen {
		return s
	}
	return s[:MaxServiceNameEchoLen]
}
