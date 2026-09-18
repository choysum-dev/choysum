// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Resolve backend environment variables from runtime global or import.meta fallback.
 */
export function getBackendEnv(): Record<string, unknown> {
  const g = globalThis as { __choysumBackendEnv?: Record<string, unknown> };
  const meta = import.meta as { env?: Record<string, unknown> };
  return g.__choysumBackendEnv || meta.env || {};
}

/**
 * Read the first defined environment variable value from the given keys.
 */
export function getBackendEnvText(...keys: string[]): string {
  const env = getBackendEnv();
  for (const key of keys) {
    const value = String(env[key] ?? '').trim();
    if (value) return value;
  }
  return '';
}

/**
 * Return whether a string environment-flag value evaluates to truthy.
 */
export function isTruthyFlag(value: unknown): boolean {
  const raw = String(value || '')
    .trim()
    .toLowerCase();
  return raw === '1' || raw === 'true' || raw === 'yes' || raw === 'on';
}

/**
 * Parse a positive integer backend env value with default fallback.
 *
 * Accepts a single key string or an array of keys tried in order.
 * The single-key signature remains backward-compatible with all existing callers.
 */
export function getBackendEnvPositiveInt(keyOrKeys: string | readonly string[], defaultValue: number): number {
  const keys = typeof keyOrKeys === 'string' ? [keyOrKeys] : keyOrKeys;
  const env = getBackendEnv();
  for (const k of keys) {
    const raw = env[k];
    const parsed = typeof raw === 'number' ? raw : typeof raw === 'string' ? Number(raw) : NaN;
    if (Number.isFinite(parsed) && parsed > 0) {
      return Math.floor(parsed);
    }
  }
  return defaultValue;
}
