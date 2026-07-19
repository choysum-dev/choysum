// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package service

import (
	i18nservice "github.com/choysum-dev/choysum/internal/i18n/service"
	"google.golang.org/grpc"
)

func (s *ApplicationService) i18nServiceDesc() (*grpc.ServiceDesc, error) {
	return i18nservice.New(s.name, s.runtimeScope).ServiceDesc()
}
