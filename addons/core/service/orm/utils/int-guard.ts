// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export function assertSafeInt(v: unknown, field: string) {
  if (v == null) return;
  if (typeof v !== 'number' || !Number.isInteger(v)) {
    throw new Error(`Field ${field} expects integer, got ${v}`);
  }
  if (!Number.isSafeInteger(v)) {
    throw new Error(`Field ${field} exceeds safe integer range: ${v}`);
  }
}
