// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package client

import (
	"regexp"
)

var serviceNameRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*(\.[A-Za-z][A-Za-z0-9]*)*$`)

func ValidateServiceName(serviceName string) error {
	if serviceName == "" {
		return &InvalidServiceNameError{ServiceName: serviceName}
	}
	if len(serviceName) > MaxServiceNameLen {
		return &InvalidServiceNameError{ServiceName: serviceName}
	}
	if !serviceNameRe.MatchString(serviceName) {
		return &InvalidServiceNameError{ServiceName: serviceName}
	}
	return nil
}
