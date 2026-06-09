// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export function normalizePrefetchedRows<T>(rows: T[] | undefined): T[] {
  return rows ?? [];
}
