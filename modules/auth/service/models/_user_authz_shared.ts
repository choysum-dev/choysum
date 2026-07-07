// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getIdentity, getReadonlyCtx, getJsCtxAndReq, getOrInitReqServiceState, withBypassDepths } from '@/core/service/api/context';
import {
  uniqStrings,
  normalizeRpcRequireKey,
  rpcServiceWildcard,
  sortStrings,
  maybeRefId as maybeId,
  normalizeScopeRefId,
  normalizeUiResourceId,
  parseJsonStringArray,
} from '@/core/service/utils/normalization';

// Re-export core utilities for backward compat — other auth helpers import these from this module.
export { sortStrings, maybeId, normalizeScopeRefId, normalizeUiResourceId, parseJsonStringArray };

/**
 * Execute fn with RecordRule and FieldRule bypass.
 */
export async function withPermissionGraphBypass<T>(fn: () => Promise<T>): Promise<T> {
  const { req } = getJsCtxAndReq();
  if (!req) return await fn();

  const state = getOrInitReqServiceState(req);
  if (!state) return await fn();

  const hadCompanyMode = Object.prototype.hasOwnProperty.call(req, 'companyMode');
  const prevCompanyMode = req.companyMode;
  req.companyMode = 'skip';

  try {
    return await withBypassDepths(state, ['recordRuleBypassDepth', 'fieldRuleBypassDepth'], fn);
  } finally {
    if (hadCompanyMode) req.companyMode = prevCompanyMode;
    else delete req.companyMode;
  }
}

/**
 * Read active and enabled company scope from request overrides or identity metadata.
 */
export function getCompanyScopeFromRequestContext(): { activeCompanyId: string; enabledCompanyIds: string[] } {
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

/**
 * Evaluate a single RPC require key against allow and deny sets.
 */
export function hasRpcPermission(req: string, allowSet: Set<string>, denySet: Set<string>): boolean {
  const k = normalizeRpcRequireKey(req);
  if (!k) return false;
  const wildcard = rpcServiceWildcard(k);

  if (denySet.has(k)) return false;
  if (wildcard && denySet.has(wildcard)) return false;

  if (allowSet.has(k)) return true;
  if (wildcard && allowSet.has(wildcard)) return true;
  return false;
}

/**
 * Check whether all UI resource requires are satisfied by RPC permissions.
 */
export function isUiResourceAllowed(requires: string[], allowSet: Set<string>, denySet: Set<string>): boolean {
  const reqs = uniqStrings((requires || []).map(v => String(v || '').trim()).filter(Boolean));
  if (reqs.length === 0) return true;

  for (const req of reqs) {
    if (!hasRpcPermission(req, allowSet, denySet)) return false;
  }
  return true;
}

/**
 * Check whether a require key targets the specified model and method.
 */
export function requireMatchesMethod(req: string, modelKey: string, methodLower: string): boolean {
  const k = normalizeRpcRequireKey(req);
  if (!k || !k.startsWith('rpc:/')) return false;
  const body = k.slice('rpc:/'.length);
  const parts = body.split('/');
  if (parts.length !== 2) return false;
  const mk = String(parts[0] || '').trim();
  const mm = String(parts[1] || '')
    .trim()
    .toLowerCase();
  if (!mk || !mm) return false;
  if (
    mk.toLowerCase() !==
    String(modelKey || '')
      .trim()
      .toLowerCase()
  )
    return false;
  return mm === '*' || mm === methodLower;
}

/**
 * Hash a password using the Choysum crypto subsystem.
 */
export function hashPassword(password: string): string {
  const prefixMarker = '$CH$';
  const crypto = (globalThis as any)?.$choysum?.crypto;
  if (!crypto) {
    throw new Error('Choysum crypto subsystem is not initialized');
  }
  if (password.startsWith(prefixMarker)) {
    const clientHashed = password.substring(prefixMarker.length);
    return crypto.hashPassword(clientHashed);
  }
  return crypto.hashPassword(password);
}

/**
 * Verify a plaintext or client-hashed password against the stored hash.
 */
export function verifyPassword(password: string, hashedPassword: string): boolean {
  const prefixMarker = '$CH$';
  const crypto = (globalThis as any)?.$choysum?.crypto;
  if (!crypto) {
    throw new Error('Choysum crypto subsystem is not initialized');
  }
  if (password.startsWith(prefixMarker)) {
    password = password.substring(prefixMarker.length);
  }
  return crypto.verifyPassword(password, hashedPassword);
}
