// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** Escape a value for use as a single-quoted SQL string literal. */
export function quoteSqlStringLiteral(value: string): string {
  return `'${String(value).replace(/'/g, "''")}'`;
}

/**
 * Build a JSON path for one object key that works with MySQL / SQLite / MSSQL
 * extractors when the key contains `-`, `.`, spaces, or other non-identifiers.
 *
 * Always uses the quoted-member form `$."key"` (double quotes inside the path).
 */
export function quoteJsonObjectPath(key: string): string {
  const escaped = String(key).replace(/\\/g, '\\\\').replace(/"/g, '\\"');
  return `$."${escaped}"`;
}
