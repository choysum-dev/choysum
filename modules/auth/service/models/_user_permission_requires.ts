// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { uniqStrings } from '@/core/service/utils/normalization';

/**
 * Normalize an RPC require key into its service-level wildcard form.
 */
export function rpcServiceWildcard(key: string): string {
  const k = String(key || '').trim();
  if (!k.startsWith('rpc:/')) return '';
  if (k.endsWith('/*')) return k;
  const i = k.lastIndexOf('/');
  if (i <= 'rpc:/'.length) return '';
  return `${k.slice(0, i)}/*`;
}

/**
 * Normalize supported require keys into the canonical rpc:/ form.
 */
export function normalizeRequireKey(key: string): string {
  const k = String(key || '').trim();
  if (!k) return '';
  if (k.startsWith('rpc:/')) return k;
  if (k.startsWith('service:/')) return `rpc:/${k.slice('service:/'.length)}`;
  return '';
}

/**
 * Evaluate a single RPC require key against allow and deny sets.
 */
export function hasRpcPermission(req: string, allowSet: Set<string>, denySet: Set<string>): boolean {
  const k = normalizeRequireKey(req);
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
  const k = normalizeRequireKey(req);
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
