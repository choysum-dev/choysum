// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export type ObjectRecord = Record<string, unknown>;
/** @deprecated Prefer ObjectRecord. Kept for incremental migration. */
export type UnknownRecord = ObjectRecord;
