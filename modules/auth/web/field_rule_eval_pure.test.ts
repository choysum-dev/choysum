// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

function assertFieldPerm(v: any): 'allow' | 'deny' | null {
  if (v == null) return null;
  if (typeof v === 'object') {
    throw new Error("invalid field rule permission: must be 'allow' or 'deny'");
  }
  const s = String(v)
    .trim()
    .toLowerCase();
  if (!s) return null;
  if (s === 'allow' || s === 'deny') return s;
  throw new Error("invalid field rule permission: must be 'allow' or 'deny'");
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

test('assertFieldPerm: rejects object bags (Search returns plain strings)', () => {
  expect(() => assertFieldPerm({ value: 'allow' })).toThrow(/allow|deny/);
  expect(() => assertFieldPerm({ Value: 'deny' })).toThrow(/allow|deny/);
  expect(() => assertFieldPerm({ id: 'allow' })).toThrow(/allow|deny/);
});

test('assertFieldPerm: throws for unrecognized object', () => {
  expect(() => assertFieldPerm({})).toThrow(/allow|deny/);
});
