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
    const value = String((env as any)?.[key] || '').trim();
    if (value) return value;
  }
  return '';
}

/**
 * Return whether a string environment-flag value evaluates to truthy.
 */
export function isTruthyFlag(value: string): boolean {
  const raw = value.trim().toLowerCase();
  return raw === '1' || raw === 'true' || raw === 'yes' || raw === 'on';
}
