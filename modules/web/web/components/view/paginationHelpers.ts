// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Pure helpers for 1-based page indexing used by ChoyPagination.
 */

/** Clamps page into [1, totalPages], treating non-finite as 1. */
export function clampChoyPage(page: number, totalPages: number): number {
  const max = Math.max(1, Math.floor(Number.isFinite(totalPages) ? totalPages : 1));
  const raw = Number.isFinite(page) ? Math.floor(page) : 1;
  if (raw < 1) {
    return 1;
  }
  if (raw > max) {
    return max;
  }
  return raw;
}

/** Zero-based row offset for a 1-based page and page size. */
export function choyPageOffset(page: number, pageSize: number): number {
  const size =
    Number.isFinite(pageSize) && pageSize > 0 ? Math.floor(pageSize) : 0;
  const p = Number.isFinite(page) && page > 0 ? Math.floor(page) : 1;
  return Math.max(0, (p - 1) * size);
}

/** Total page count for a result set (at least 1 when pageSize is valid). */
export function choyTotalPages(total: number, pageSize: number): number {
  const size =
    Number.isFinite(pageSize) && pageSize > 0 ? Math.floor(pageSize) : 0;
  if (size <= 0) {
    return 1;
  }
  const n = Number.isFinite(total) && total > 0 ? Math.floor(total) : 0;
  return Math.max(1, Math.ceil(n / size));
}
