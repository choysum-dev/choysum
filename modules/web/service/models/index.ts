// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Web service model exports.
 *
 * Layout exemplar (thin models): logic stays on the model class files; no models/_*.ts
 * bypasses. Prefer this shape for new small models (alongside partner).
 */
export { default as UserFilter } from './user_filter';
export { default as ExportTemplate } from './export_template';
