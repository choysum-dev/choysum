// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, test, expect } from 'vitest';

function normalizeFieldPerm(v: any): 'allow' | 'deny' | null {
  if (v == null) return null;
  if (typeof v === 'object') {
    const raw = (v as any)?.value ?? (v as any)?.Value ?? (v as any)?.id ?? (v as any)?.Id;
    if (raw != null && raw !== v) return normalizeFieldPerm(raw);
  }
  const s = String(v ?? '')
    .trim()
    .toLowerCase();
  if (!s) return null;
  if (s === 'allow' || s === 'deny') return s;
  return null;
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

describe('normalizeFieldPerm', () => {
  test('returns null for null/undefined', () => {
    expect(normalizeFieldPerm(null)).toBeNull();
    expect(normalizeFieldPerm(undefined)).toBeNull();
  });

  test('returns allow/deny for string input', () => {
    expect(normalizeFieldPerm('allow')).toBe('allow');
    expect(normalizeFieldPerm('deny')).toBe('deny');
    expect(normalizeFieldPerm('  ALLOW  ')).toBe('allow');
  });

  test('returns null for unrecognized string', () => {
    expect(normalizeFieldPerm('maybe')).toBeNull();
    expect(normalizeFieldPerm('')).toBeNull();
  });

  test('unwraps objects with value/Value/id/Id', () => {
    expect(normalizeFieldPerm({ value: 'allow' })).toBe('allow');
    expect(normalizeFieldPerm({ Value: 'deny' })).toBe('deny');
    expect(normalizeFieldPerm({ id: 'allow' })).toBe('allow');
  });

  test('returns null for unrecognized object', () => {
    expect(normalizeFieldPerm({})).toBeNull();
  });
});

describe('pickField', () => {
  test('returns undefined for non-object', () => {
    expect(pickField(null, ['a'])).toBeUndefined();
    expect(pickField(42, ['a'])).toBeUndefined();
  });

  test('returns value by exact key match', () => {
    expect(pickField({ a: 1 }, ['a'])).toBe(1);
    expect(pickField({ b: 2 }, ['a', 'b'])).toBe(2);
  });

  test('prefers first matching key', () => {
    expect(pickField({ a: 1, b: 2 }, ['b', 'a'])).toBe(2);
  });

  test('falls back to normalized case-insensitive match', () => {
    expect(pickField({ FieldName: 'value' }, ['fieldname'])).toBe('value');
    expect(pickField({ 'field-name': 'v' }, ['fieldname'])).toBe('v');
  });

  test('in check beats for-in normalized fallback', () => {
    expect(pickField({ a: 'exact', A: 'upper' }, ['a'])).toBe('exact');
  });

  test('returns undefined when no match', () => {
    expect(pickField({ a: 1 }, ['b'])).toBeUndefined();
  });
});
