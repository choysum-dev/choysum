// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package grpcclient

//go:generate protoc -I ../../../.choysum/generated/proto/auth -I ../../../internal/module/artifact/generate/assets --go_out=paths=source_relative:./authpb --go-grpc_out=paths=source_relative:./authpb ../../../.choysum/generated/proto/auth/auth.proto
