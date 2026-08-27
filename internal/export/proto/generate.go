// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package proto

//go:generate protoc -I . --go_out=paths=source_relative:./exportpb --go-grpc_out=paths=source_relative:./exportpb ./export.proto
