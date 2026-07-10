// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Calculate exponential backoff with a ceiling.
 *
 * @param attempts    Current attempt count (≥ 1).
 * @param baseSeconds Base delay in seconds for the first retry.
 * @returns Backoff delay in seconds, capped at 6 hours.
 */
export function computeRetryBackoffSeconds(attempts: number, baseSeconds: number): number {
  const exponent = Math.max(0, Math.min(10, attempts - 1));
  const backoff = baseSeconds * 2 ** exponent;
  return Math.min(backoff, 6 * 60 * 60);
}
