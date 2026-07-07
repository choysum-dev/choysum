// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getReadonlyCtx, getIdentity } from '@/core/service/api/context';
import { uniqStrings } from './_user_permission_requires';

/**
 * Resolve the current request object from the active Choysum runtime context.
 */
export function getCurrentReq(): any {
  const root: any = (globalThis as any)?.$choysum;
  const jsCtx: any = (root?.request?.context ?? root?.context ?? root) as any;
  return (jsCtx?.req ?? jsCtx?.request?.context?.req ?? jsCtx?.context?.req) as any;
}

/**
 * Return the request-scoped service cache, creating it when needed.
 */
export function getOrInitReqServiceState(req: any): any {
  if (!req) return undefined;
  if (!req.__choysumServiceState) req.__choysumServiceState = {};
  return req.__choysumServiceState;
}

/**
 * Read active and enabled company scope from request overrides or identity metadata.
 */
export function getCompanyScopeFromRequestContext(): { activeCompanyId: string; enabledCompanyIds: string[] } {
  // IMPORTANT: Use core runtime/context helpers to respect request-level overrides set by withContext().
  // Some tests (and potentially other flows) rely on Symbol.for('choysum.ctx.override') rather than
  // mutating jsCtx.ctx directly.
  const ctx: any = (getReadonlyCtx() ?? {}) as any;
  const identity: any = (getIdentity() ?? {}) as any;
  const meta: any = (identity?.metadata ?? identity?.Metadata ?? {}) as any;

  const activeCompanyId = String(ctx?.activeCompanyId ?? meta?.activeCompanyId ?? '').trim();
  const enabledCompanyIds = uniqStrings(ctx?.enabledCompanyIds ?? meta?.enabledCompanyIds ?? []);

  if (activeCompanyId && !enabledCompanyIds.includes(activeCompanyId)) {
    return { activeCompanyId, enabledCompanyIds: [activeCompanyId, ...enabledCompanyIds] };
  }
  return { activeCompanyId, enabledCompanyIds };
}
