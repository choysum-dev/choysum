// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/* eslint-disable */

// Export all stores.
{{- range $model := .Models}}
export * from './{{$model.Name | ToSnakeCase}}';
{{- end}}