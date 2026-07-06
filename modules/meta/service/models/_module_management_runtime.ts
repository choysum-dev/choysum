// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel } from '@/core/service';

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

/**
 * Return the current authenticated user Id, falling back to the E2E operator
 * env override. Throws when no user identity can be resolved.
 */
export function ensureCurrentUserId(): string {
  // Primary: use BaseModel's context-aware resolution.
  try {
    return BaseModel.ensureUserId();
  } catch {
    // Fall through to E2E operator fallback.
  }
  const fallback = getBackendEnvText('CHOYSUM_E2E_OPERATOR_USER_ID', 'choysum_e2e_operator_user_id');
  if (fallback) return fallback;
  throw new Error('current user is required');
}

/**
 * Return the moduleManagement bridge injected in the global $choysum namespace.
 * Throws when the bridge has not been registered.
 */
export function getModuleManagementBridge(): any {
  const root: any = (globalThis as any)?.$choysum;
  if (!root?.moduleManagement) {
    throw new Error('moduleManagement bridge is not injected');
  }
  return root.moduleManagement;
}
