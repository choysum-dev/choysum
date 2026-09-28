// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export type ChoyNumberMode = 'integer' | 'float' | 'decimal' | 'bigint';

/**
 * Host model is already a number — validate mode against the value itself.
 * `String(1e-22)` is exponential and `parseChoyNumber` rejects it, but the host
 * number is still a valid float/decimal.
 */
export function isChoyNumberHostCompatibleWithMode(
  value: number,
  mode: ChoyNumberMode,
): boolean {
  if (!Number.isFinite(value)) {
    return false;
  }
  if (mode === 'integer' || mode === 'bigint') {
    return Number.isSafeInteger(value);
  }
  return Number.isSafeInteger(Math.trunc(value));
}

/**
 * Chrome parse mode collapses bigint onto integer digit rules.
 * Undefined mode defaults to decimal (matches store-mode ODecimalField fallback).
 */
export function resolveChoyNumberChromeParseMode(
  mode?: ChoyNumberMode,
): 'integer' | 'float' | 'decimal' {
  if (mode === 'bigint') return 'integer';
  return mode ?? 'decimal';
}

/** Inputmode for the chrome text control. */
export function resolveChoyNumberInputMode(mode?: ChoyNumberMode): 'numeric' | 'decimal' {
  return mode === 'integer' || mode === 'bigint' ? 'numeric' : 'decimal';
}
