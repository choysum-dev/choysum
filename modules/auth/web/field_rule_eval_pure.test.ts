// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

function assertFieldPerm(v: any): 'allow' | 'deny' | null {
  if (v == null) return null;
  if (typeof v === 'object') {
    const raw = (v as any)?.value ?? (v as any)?.Value ?? (v as any)?.id ?? (v as any)?.Id;
    if (raw != null && raw !== v) return assertFieldPerm(raw);
  }
  const s = String(v ?? '')
    .trim()
    .toLowerCase();
  if (!s) return null;
  if (s === 'allow' || s === 'deny') return s;
  throw new Error("invalid field rule permission: must be 'allow' or 'deny'");
}

function pickField(obj: any, keys: string[]): any {
  if (!obj || (typeof obj !== 'object' && typeof obj !== 'function')) return undefined;

  for (const k of keys) {
    if (k in obj) return obj[k];
  }

  const norm = (s: string): string => s.toLowerCase().replace(/[^a-z0-9]/g, '');
  const normalizedWants = keys.map(norm);

  for (const k in obj) {
    if (normalizedWants.includes(norm(k))) {
      return obj[k];
    }
  }

  return undefined;
}

test('assertFieldPerm: returns null for null/undefined', () => {
  expect(assertFieldPerm(null)).toBeNull();
  expect(assertFieldPerm(undefined)).toBeNull();
});

test('assertFieldPerm: returns allow/deny for string input', () => {
  expect(assertFieldPerm('allow')).toBe('allow');
  expect(assertFieldPerm('deny')).toBe('deny');
  expect(assertFieldPerm('  ALLOW  ')).toBe('allow');
});

test('assertFieldPerm: throws for unrecognized string; blank stays null', () => {
  expect(() => assertFieldPerm('maybe')).toThrow(/allow|deny/);
  expect(assertFieldPerm('')).toBeNull();
});

test('assertFieldPerm: unwraps objects with value/Value/id/Id', () => {
  expect(assertFieldPerm({ value: 'allow' })).toBe('allow');
  expect(assertFieldPerm({ Value: 'deny' })).toBe('deny');
  expect(assertFieldPerm({ id: 'allow' })).toBe('allow');
});

test('assertFieldPerm: throws for unrecognized object', () => {
  expect(() => assertFieldPerm({})).toThrow(/allow|deny/);
});

test('pickField: returns undefined for non-object', () => {
  expect(pickField(null, ['a'])).toBeUndefined();
  expect(pickField(42, ['a'])).toBeUndefined();
});

test('pickField: returns value by exact key match', () => {
  expect(pickField({ a: 1 }, ['a'])).toBe(1);
  expect(pickField({ b: 2 }, ['a', 'b'])).toBe(2);
});

test('pickField: prefers first matching key', () => {
  expect(pickField({ a: 1, b: 2 }, ['b', 'a'])).toBe(2);
});

test('pickField: falls back to normalized case-insensitive match', () => {
  expect(pickField({ FieldName: 'value' }, ['fieldname'])).toBe('value');
  expect(pickField({ 'field-name': 'v' }, ['fieldname'])).toBe('v');
});

test('pickField: in check beats for-in normalized fallback', () => {
  expect(pickField({ a: 'exact', A: 'upper' }, ['a'])).toBe('exact');
});

test('pickField: returns undefined when no match', () => {
  expect(pickField({ a: 1 }, ['b'])).toBeUndefined();
});
