// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Derive a BCP47 tag from a POSIX Language.Code (underscore → hyphen).
 * Used for Intl / Calendar only — not store identity.
 */
export function bcp47FromCode(code: string): string {
  const raw = String(code || '').trim();
  if (!raw) {
    return 'en-US';
  }
  return raw.replace(/_/g, '-');
}
