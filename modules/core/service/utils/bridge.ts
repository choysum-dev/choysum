// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getUserId } from '@/core/service/api/context';

/**
 * Return the current operator user ID.
 *
 * The user ID is injected by the Go service layer into the request identity
 * snapshot at call time.  For E2E test scenarios the Go layer also applies
 * a CHOYSUM_E2E_OPERATOR_USER_ID env-var fallback before the request reaches
 * JS, so callers no longer need their own env-var escape hatch.
 */
export function getOperatorUserId(): string {
  const userId = String(getUserId() || '').trim();
  if (!userId) {
    throw new Error('current user is required');
  }
  return userId;
}

/**
 * Return the module-management bridge injected by the Go QuickJS runtime plugin.
 * Throws with a clear diagnostic when the bridge has not been registered.
 */
export function getModuleManagement(): any {
  const root: any = (globalThis as any)?.$choysum;
  if (!root?.moduleManagement) {
    throw new Error('moduleManagement bridge is not injected');
  }
  return root.moduleManagement;
}

/**
 * Return the backend env snapshot injected by the Go service layer.
 * The snapshot is a subset of os.Environ keys prefixed with CHOYSUM_
 * (case‑insensitive), captured once per request in buildJsContext.
 */
export function getBackendEnv(): Record<string, unknown> {
  // Read from the request-scoped context injected by Go.
  // Fall back to import.meta.env / globalThis for backwards compatibility
  // during the transition period.
  const root: any = (globalThis as any)?.$choysum;
  const jsCtx: any = root?.request?.context ?? root?.context ?? root;
  const env = (jsCtx?.env as Record<string, unknown>) ?? {};
  if (Object.keys(env).length > 0) return env;

  // Legacy fallback — remove once all callers receive the injected env snapshot.
  return ((import.meta as any)?.env || (globalThis as any)?.__choysumBackendEnv || {}) as Record<string, unknown>;
}

/**
 * Return the first non‑empty value for the given env keys (searched in order).
 * Empty string if none of the keys contain a value.
 */
export function getBackendEnvText(...keys: string[]): string {
  const env = getBackendEnv();
  for (const key of keys) {
    const value = String((env as any)?.[key] || '').trim();
    if (value) return value;
  }
  return '';
}
