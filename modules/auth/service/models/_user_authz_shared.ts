// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getIdentity, getReadonlyCtx, getJsCtxAndReq } from '@/core/service/api/context';
import { uniqStrings, normalizeRpcRequireKey, rpcServiceWildcard } from '@/core/service/utils/normalization';

/**
 * Execute fn with RecordRule and FieldRule bypass.
 */
export async function withPermissionGraphBypass<T>(fn: () => Promise<T>): Promise<T> {
  const { req } = getJsCtxAndReq();
  if (!req) return await fn();

  if (!req.__choysumServiceState) req.__choysumServiceState = {};
  const state: any = req.__choysumServiceState;

  const prevRR = typeof state.recordRuleBypassDepth === 'number' ? state.recordRuleBypassDepth : 0;
  const prevFR = typeof state.fieldRuleBypassDepth === 'number' ? state.fieldRuleBypassDepth : 0;
  const hadCompanyMode = Object.prototype.hasOwnProperty.call(req, 'companyMode');
  const prevCompanyMode = req.companyMode;

  state.recordRuleBypassDepth = prevRR + 1;
  state.fieldRuleBypassDepth = prevFR + 1;
  req.companyMode = 'skip';
  try {
    return await fn();
  } finally {
    const nextRR = prevRR;
    const nextFR = prevFR;
    if (nextRR > 0) state.recordRuleBypassDepth = nextRR;
    else delete state.recordRuleBypassDepth;
    if (nextFR > 0) state.fieldRuleBypassDepth = nextFR;
    else delete state.fieldRuleBypassDepth;

    if (hadCompanyMode) req.companyMode = prevCompanyMode;
    else delete req.companyMode;
  }
}

/**
 * Return a stable sorted copy of a string list.
 */
export function sortStrings(xs: string[]): string[] {
  return (xs || []).slice().sort((a, b) => (a < b ? -1 : a > b ? 1 : 0));
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
 * Parse a string or structured value into a normalized string array.
 */
export function parseJsonStringArray(raw: any): string[] {
  const normalize = (xs: any[]): string[] => uniqStrings((xs || []).map(v => String(v ?? '').trim()).filter(Boolean));
  const tryObjectSnapshot = (value: any): string[] | null => {
    if (!value || typeof value !== 'object') return null;
    try {
      const snap = JSON.parse(JSON.stringify(value));
      if (Array.isArray(snap)) return normalize(snap);
      if (!snap || typeof snap !== 'object') return null;

      for (const key of ['value', 'values', 'items']) {
        if (Array.isArray((snap as any)[key])) return normalize((snap as any)[key]);
      }

      const numericKeys = Object.keys(snap)
        .filter(key => /^\d+$/.test(key))
        .sort((a, b) => Number(a) - Number(b));
      if (numericKeys.length > 0) return normalize(numericKeys.map(key => (snap as any)[key]));
    } catch {
      // fallthrough
    }
    return null;
  };

  if (Array.isArray(raw)) return normalize(raw);
  if (raw == null) return [];

  if (typeof raw === 'string') {
    const s = raw.trim();
    if (!s) return [];
    try {
      const parsed = JSON.parse(s);
      if (Array.isArray(parsed)) return normalize(parsed);
    } catch {
      // fallthrough
    }
    return normalize([s]);
  }

  const snapResult = tryObjectSnapshot(raw);
  if (snapResult) return snapResult;

  try {
    if (typeof (raw as any)?.toString === 'function') {
      const s = String((raw as any).toString() || '').trim();
      if (!s) return [];
      const parsed = JSON.parse(s);
      if (Array.isArray(parsed)) return normalize(parsed);
    }
  } catch {
    // fallthrough
  }
  return [];
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
 * Normalize a UI resource reference into its string id.
 */
export function normalizeUiResourceId(raw: any): string {
  if (raw == null) return '';
  if (typeof raw === 'object' && raw !== null) {
    return String((raw as any).Id ?? (raw as any).id ?? '').trim();
  }
  return String(raw || '').trim();
}

/**
 * Normalize an application or scope reference into its string id.
 */
export function normalizeScopeRefId(raw: any): string {
  if (raw == null) return '';
  if (typeof raw === 'object' && raw !== null) {
    return String((raw as any).Id ?? (raw as any).id ?? '').trim();
  }
  return String(raw || '').trim();
}

/**
 * Normalize a model reference or scalar value into a string id.
 */
export function maybeId(value: any): string | undefined {
  if (!value) return undefined;
  if (typeof value === 'string') return value;
  if (typeof value === 'object') return String((value as any).Id ?? (value as any).id ?? '').trim() || undefined;
  return undefined;
}

/**
 * Hash a password using the Choysum crypto subsystem.
 */
export function hashPassword(password: string): string {
  const prefixMarker = '$CH$';
  if (password.startsWith(prefixMarker)) {
    const clientHashed = password.substring(prefixMarker.length);
    return (globalThis as any).$choysum.crypto.hashPassword(clientHashed);
  }
  return (globalThis as any).$choysum.crypto.hashPassword(password);
}

/**
 * Verify a plaintext or client-hashed password against the stored hash.
 */
export function verifyPassword(password: string, hashedPassword: string): boolean {
  const prefixMarker = '$CH$';
  if (password.startsWith(prefixMarker)) {
    password = password.substring(prefixMarker.length);
  }
  return (globalThis as any).$choysum.crypto.verifyPassword(password, hashedPassword);
}
