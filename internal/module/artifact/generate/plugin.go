// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package generator

import (
	"google.golang.org/protobuf/types/pluginpb"
)

type GrpcPlugin interface {
	Name() string
	Generate(*pluginpb.CodeGeneratorRequest) (*pluginpb.CodeGeneratorResponse, error)
}
