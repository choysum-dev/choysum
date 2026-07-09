// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Resolve backend environment variables from import.meta or global fallback.
 */
export function getBackendEnv(): Record<string, unknown> {
  return (((import.meta as any)?.env || (globalThis as any)?.__choysumBackendEnv || {}) as Record<string, unknown>) || {};
}

/**
 * Read the first defined environment variable value from the given keys.
 */
export function getBackendEnvText(...keys: string[]): string {
  const env = getBackendEnv();
  for (const key of keys) {
    const value = String((env as any)?.[key] ?? '').trim();
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
 */
export function getBackendEnvPositiveInt(key: string, defaultValue: number): number {
  const globalEnv = ((globalThis as any)?.__choysumBackendEnv || {}) as Record<string, unknown>;
  const metaEnv = (((import.meta as any)?.env || {}) as Record<string, unknown>) || {};
  const raw = (globalEnv as any)?.[key] ?? (metaEnv as any)?.[key];
  const parsed = typeof raw === 'number' ? raw : typeof raw === 'string' ? Number(raw) : NaN;
  if (!Number.isFinite(parsed) || parsed <= 0) return defaultValue;
  return Math.floor(parsed);
}
