// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Inlined from user/_authz_shared.ts and normalization.ts to avoid backend imports.
function normalizeRpcRequireKey(key: string): string {
  const k = String(key || '').trim();
  if (!k) return '';
  if (k.startsWith('rpc:/')) return k;
  if (k.startsWith('service:/')) return `rpc:/${k.slice('service:/'.length)}`;
  return '';
}

function rpcServiceWildcard(key: string): string {
  const k = String(key || '').trim();
  if (!k.startsWith('rpc:/')) return '';
  if (k.endsWith('/*')) return k;
  const i = k.lastIndexOf('/');
  if (i <= 'rpc:/'.length) return '';
  return `${k.slice(0, i)}/*`;
}

function uniqStrings(xs: unknown): string[] {
  return Array.from(new Set((Array.isArray(xs) ? xs : []).map(v => String(v ?? '').trim()).filter(Boolean)));
}

function hasRpcPermission(req: string, allowSet: Set<string>, denySet: Set<string>): boolean {
  const k = normalizeRpcRequireKey(req);
  if (!k) return false;
  const wildcard = rpcServiceWildcard(k);
  if (denySet.has(k)) return false;
  if (wildcard && denySet.has(wildcard)) return false;
  if (allowSet.has(k)) return true;
  if (wildcard && allowSet.has(wildcard)) return true;
  return false;
}

function isUiResourceAllowed(requires: string[], allowSet: Set<string>, denySet: Set<string>): boolean {
  const reqs = uniqStrings((requires || []).map(v => String(v || '').trim()).filter(Boolean));
  if (reqs.length === 0) return true;
  for (const req of reqs) {
    if (!hasRpcPermission(req, allowSet, denySet)) return false;
  }
  return true;
}

function requireMatchesMethod(req: string, modelKey: string, methodLower: string): boolean {
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

test('hasRpcPermission: returns false for empty or invalid key', () => {
  const allowSet = new Set<string>(['rpc:/base.Partner/Read', 'rpc:/base.Partner/*']);
  const denySet = new Set<string>(['rpc:/base.Partner/Delete']);
  expect(hasRpcPermission('', allowSet, denySet)).toBe(false);
  expect(hasRpcPermission('invalid', allowSet, denySet)).toBe(false);
  expect(hasRpcPermission('x-rpc:/base.Partner/Read', allowSet, denySet)).toBe(false);
});

test('hasRpcPermission: deny takes precedence over allow', () => {
  const allowSet = new Set<string>(['rpc:/base.Partner/Read', 'rpc:/base.Partner/*']);
  expect(hasRpcPermission('rpc:/base.Partner/Delete', allowSet, new Set(['rpc:/base.Partner/Delete']))).toBe(false);
});

test('hasRpcPermission: wildcard deny blocks specific key', () => {
  const allowSet = new Set<string>(['rpc:/base.Partner/Read', 'rpc:/base.Partner/*']);
  const d = new Set(['rpc:/base.Partner/*']);
  expect(hasRpcPermission('rpc:/base.Partner/Read', allowSet, d)).toBe(false);
});

test('hasRpcPermission: exact allow returns true', () => {
  const allowSet = new Set<string>(['rpc:/base.Partner/Read', 'rpc:/base.Partner/*']);
  const denySet = new Set<string>(['rpc:/base.Partner/Delete']);
  expect(hasRpcPermission('rpc:/base.Partner/Read', allowSet, denySet)).toBe(true);
});

test('hasRpcPermission: wildcard allow returns true', () => {
  const allowSet = new Set<string>(['rpc:/base.Partner/Read', 'rpc:/base.Partner/*']);
  const denySet = new Set<string>(['rpc:/base.Partner/Delete']);
  expect(hasRpcPermission('rpc:/base.Partner/Write', allowSet, denySet)).toBe(true);
});

test('hasRpcPermission: no match returns false', () => {
  const allowSet = new Set<string>(['rpc:/base.Partner/Read', 'rpc:/base.Partner/*']);
  const denySet = new Set<string>(['rpc:/base.Partner/Delete']);
  expect(hasRpcPermission('rpc:/other.Model/Read', allowSet, denySet)).toBe(false);
});

test('hasRpcPermission: wildcard allow but exact deny returns false', () => {
  const a = new Set(['rpc:/base.Partner/*']);
  const d = new Set(['rpc:/base.Partner/Delete']);
  expect(hasRpcPermission('rpc:/base.Partner/Delete', a, d)).toBe(false);
});

test('isUiResourceAllowed: empty requires returns true', () => {
  const allowSet = new Set<string>(['rpc:/base.Partner/Read']);
  const denySet = new Set<string>();
  expect(isUiResourceAllowed([], allowSet, denySet)).toBe(true);
  expect(isUiResourceAllowed(undefined as any, allowSet, denySet)).toBe(true);
  expect(isUiResourceAllowed([''], allowSet, denySet)).toBe(true);
});

test('isUiResourceAllowed: single satisfied require returns true', () => {
  const allowSet = new Set<string>(['rpc:/base.Partner/Read']);
  const denySet = new Set<string>();
  expect(isUiResourceAllowed(['rpc:/base.Partner/Read'], allowSet, denySet)).toBe(true);
});

test('isUiResourceAllowed: single unsatisfied require returns false', () => {
  const allowSet = new Set<string>(['rpc:/base.Partner/Read']);
  const denySet = new Set<string>();
  expect(isUiResourceAllowed(['rpc:/base.Partner/Write'], allowSet, denySet)).toBe(false);
});

test('isUiResourceAllowed: all requires must be satisfied', () => {
  const a = new Set(['rpc:/base.Partner/Read', 'rpc:/base.Partner/Write']);
  const denySet = new Set<string>();
  expect(isUiResourceAllowed(['rpc:/base.Partner/Read', 'rpc:/base.Partner/Write'], a, denySet)).toBe(true);
});

test('isUiResourceAllowed: one deny blocks all', () => {
  const a = new Set(['rpc:/base.Partner/Read', 'rpc:/base.Partner/Write']);
  const d = new Set(['rpc:/base.Partner/Read']);
  expect(isUiResourceAllowed(['rpc:/base.Partner/Read', 'rpc:/base.Partner/Write'], a, d)).toBe(false);
});

test('requireMatchesMethod: returns false for empty or non-rpc key', () => {
  expect(requireMatchesMethod('', 'base.Partner', 'read')).toBe(false);
  expect(requireMatchesMethod('invalid', 'base.Partner', 'read')).toBe(false);
  expect(requireMatchesMethod('rpc:/base.Partner/Read/extra', 'base.Partner', 'read')).toBe(false);
});

test('requireMatchesMethod: returns false for malformed rpc key', () => {
  expect(requireMatchesMethod('rpc:/onlymodel', 'base.Partner', 'read')).toBe(false);
  expect(requireMatchesMethod('rpc:/a/b/c', 'base.Partner', 'read')).toBe(false);
});

test('requireMatchesMethod: wildcard method matches any method', () => {
  expect(requireMatchesMethod('rpc:/base.Partner/*', 'base.Partner', 'read')).toBe(true);
  expect(requireMatchesMethod('rpc:/base.Partner/*', 'base.Partner', 'write')).toBe(true);
});

test('requireMatchesMethod: exact method match is case-insensitive (caller passes lowercased method)', () => {
  expect(requireMatchesMethod('rpc:/base.Partner/Read', 'base.Partner', 'read')).toBe(true);
  expect(requireMatchesMethod('rpc:/base.partner/read', 'base.partner', 'read')).toBe(true);
});

test('requireMatchesMethod: model mismatch returns false', () => {
  expect(requireMatchesMethod('rpc:/base.Partner/Read', 'base.Company', 'read')).toBe(false);
});

test('requireMatchesMethod: method mismatch returns false', () => {
  expect(requireMatchesMethod('rpc:/base.Partner/Read', 'base.Partner', 'write')).toBe(false);
});
