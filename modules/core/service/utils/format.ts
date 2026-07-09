// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Format a number with optional prefix, suffix, and zero-padding.
 *
 * @param prefix  - string prepended before the padded number
 * @param suffix  - string appended after the padded number
 * @param padding - minimum digit count for zero-padding (0 = no padding)
 * @param n       - the number to format
 */
export function formatPaddedNumber(prefix: string | undefined, suffix: string | undefined, padding: number, n: bigint): string {
  const p = prefix ?? '';
  const s = suffix ?? '';
  const pad = Number.isFinite(padding) && padding > 0 ? Math.floor(padding) : 0;
  const core = n.toString();
  const padded = pad > 0 ? core.padStart(pad, '0') : core;
  return `${p}${padded}${s}`;
}

/**
 * Resolve prefix/suffix/padding from an optional snapshot with fallback values.
 */
export function resolvePaddedNumberFormat(
  snapshot: unknown,
  fallback: { prefix: string | undefined; suffix: string | undefined; padding: number }
): { prefix: string | undefined; suffix: string | undefined; padding: number } {
  const snap = (snapshot && typeof snapshot === 'object' ? snapshot : {}) as Record<string, unknown>;
  const paddingFromSnapshot = Number(snap.Padding);
  return {
    prefix: typeof snap.Prefix === 'string' ? snap.Prefix : fallback.prefix,
    suffix: typeof snap.Suffix === 'string' ? snap.Suffix : fallback.suffix,
    padding: Number.isFinite(paddingFromSnapshot) ? paddingFromSnapshot : fallback.padding,
  };
}

/**
 * Build a contiguous formatted number list from start/count.
 */
export function buildPaddedNumberItems(
  start: bigint,
  count: number,
  prefix: string | undefined,
  suffix: string | undefined,
  padding: number
): Array<{ Value: string; Number: number }> {
  const total = Number.isFinite(count) ? Math.max(0, Math.floor(count)) : 0;
  const items: Array<{ Value: string; Number: number }> = [];
  for (let i = 0; i < total; i++) {
    const n = start + BigInt(i);
    items.push({ Value: formatPaddedNumber(prefix, suffix, padding, n), Number: Number(n) });
  }
  return items;
}
