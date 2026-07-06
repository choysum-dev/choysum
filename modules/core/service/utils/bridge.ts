// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getUserId } from '@/core/service/api/context';

function resolveJsCtx(root: any): Record<string, unknown> {
  const getRequestContext = root?.getRequestContext;
  if (typeof getRequestContext === 'function') {
    try {
      const fromAccessor = getRequestContext();
      if (fromAccessor && typeof fromAccessor === 'object') return fromAccessor as Record<string, unknown>;
    } catch {
      // ignore and continue
    }
  }

  const getActiveRequest = root?.getActiveRequest;
  if (typeof getActiveRequest === 'function') {
    try {
      const req = getActiveRequest();
      const reqCtx = (req as any)?.context;
      if (reqCtx && typeof reqCtx === 'object') return reqCtx as Record<string, unknown>;
    } catch {
      // ignore and continue
    }
  }

  return (root?.request?.context ?? root?.context ?? root ?? {}) as Record<string, unknown>;
}

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
  // Read from the request-scoped context injected by Go/runtime accessors.
  const root: any = (globalThis as any)?.$choysum;
  const jsCtx = resolveJsCtx(root);
  const env = (jsCtx?.env as Record<string, unknown>) ?? {};
  return env && typeof env === 'object' ? env : {};
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
