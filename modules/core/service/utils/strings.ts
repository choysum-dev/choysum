// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Normalize an unknown value into a trimmed non-empty string.
 * Returns empty string for null/undefined/empty inputs.
 */
export function normalizeString(value: unknown): string {
  return String(value ?? '').trim();
}

/**
 * Normalize an unknown value into a trimmed string, or undefined if empty.
 */
export function normalizeOptionalString(value: unknown): string | undefined {
  const normalized = String(value ?? '').trim();
  return normalized || undefined;
}

/**
 * Normalize a possibly mixed array into unique non-empty trimmed strings.
 * Non-array inputs are treated as empty.
 */
export function normalizeStringArray(value: unknown): string[] {
  const arr = Array.isArray(value) ? value : [];
  return Array.from(new Set(arr.map(item => String(item ?? '').trim()).filter(Boolean)));
}
