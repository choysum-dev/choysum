// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, test, expect } from 'vitest';

// Inlined from _role_ui_projection.ts for pure-function testing.
function normalizeRefId(value: unknown): string | null {
  if (value == null) return null;
  const raw = typeof value === 'object' ? ((value as any).Id ?? (value as any).id ?? null) : value;
  const s = String(raw ?? '').trim();
  return s ? s : null;
}

function isAllowResourceScope(row: any): boolean {
  const mode = String((row as any)?.Mode ?? 'allow')
    .trim()
    .toLowerCase();
  const uiResourceId = normalizeRefId((row as any)?.IrUiResourceId);
  const appId = normalizeRefId((row as any)?.IrApplicationId);
  return mode === 'allow' && !!uiResourceId && appId == null;
}

function makeAllowResourceEntries(ids: string[]): Array<Record<string, any>> {
  return ids.map(id => ({
    Mode: 'allow',
    IrApplicationId: null,
    IrUiResourceId: id,
  }));
}

function extractUiResourcesArray(v: any): any[] | null {
  if (Array.isArray(v)) return v;
  if (v && typeof v === 'object' && Array.isArray((v as any).replace)) return (v as any).replace;
  return null;
}

function mergeAccessIntoUiResources(baseRows: Array<Record<string, any>>, accessIds: string[]): Array<Record<string, any>> {
  const preserved = (baseRows || []).filter(row => !isAllowResourceScope(row));
  const allowRows = makeAllowResourceEntries(accessIds);
  return [...preserved, ...allowRows];
}

function wantsAccessField(selection: any): boolean {
  if (selection == null) return false;
  if (typeof selection === 'string') return selection === 'AccessUiResourceIds';
  if (Array.isArray(selection)) return selection.some(it => wantsAccessField(it));
  if (typeof selection === 'object' && Array.isArray((selection as any).fields)) {
    return wantsAccessField((selection as any).fields);
  }
  return false;
}

describe('isAllowResourceScope', () => {
  test('returns true for allow mode with resource id and no app id', () => {
    expect(isAllowResourceScope({ Mode: 'allow', IrUiResourceId: 'R1' })).toBe(true);
    expect(isAllowResourceScope({ Mode: 'ALLOW', IrUiResourceId: 'R1', IrApplicationId: null })).toBe(true);
  });

  test('returns false for deny mode', () => {
    expect(isAllowResourceScope({ Mode: 'deny', IrUiResourceId: 'R1' })).toBe(false);
  });

  test('returns false when app id is present', () => {
    expect(isAllowResourceScope({ Mode: 'allow', IrUiResourceId: 'R1', IrApplicationId: 'A1' })).toBe(false);
    expect(isAllowResourceScope({ Mode: 'allow', IrUiResourceId: 'R1', IrApplicationId: { Id: 'A1' } })).toBe(false);
  });

  test('returns false when resource id is missing', () => {
    expect(isAllowResourceScope({ Mode: 'allow' })).toBe(false);
    expect(isAllowResourceScope({ Mode: 'allow', IrUiResourceId: null })).toBe(false);
  });

  test('defaults mode to allow', () => {
    expect(isAllowResourceScope({ IrUiResourceId: 'R1' })).toBe(true);
  });
});

describe('makeAllowResourceEntries', () => {
  test('returns array of allow entries', () => {
    const result = makeAllowResourceEntries(['R1', 'R2']);
    expect(result).toEqual([
      { Mode: 'allow', IrApplicationId: null, IrUiResourceId: 'R1' },
      { Mode: 'allow', IrApplicationId: null, IrUiResourceId: 'R2' },
    ]);
  });

  test('returns empty for empty ids', () => {
    expect(makeAllowResourceEntries([])).toEqual([]);
  });
});

describe('extractUiResourcesArray', () => {
  test('returns array as-is', () => {
    expect(extractUiResourcesArray(['a', 'b'])).toEqual(['a', 'b']);
  });

  test('returns replace property from object', () => {
    expect(extractUiResourcesArray({ replace: ['x'] })).toEqual(['x']);
  });

  test('returns null for non-array non-object', () => {
    expect(extractUiResourcesArray('foo')).toBeNull();
    expect(extractUiResourcesArray(null)).toBeNull();
    expect(extractUiResourcesArray(42)).toBeNull();
  });

  test('returns null for object without replace array', () => {
    expect(extractUiResourcesArray({})).toBeNull();
  });
});

describe('mergeAccessIntoUiResources', () => {
  test('preserves non-allow rows and appends allow entries', () => {
    const base = [
      { Mode: 'deny', IrUiResourceId: 'R1' },
      { Mode: 'allow', IrUiResourceId: 'R2' },
    ];
    const result = mergeAccessIntoUiResources(base, ['R3']);
    // Only the deny row is preserved; allow rows are replaced.
    expect(result).toHaveLength(2);
    expect(result[0].Mode).toBe('deny');
    expect(result[1].Mode).toBe('allow');
    expect(result[1].IrUiResourceId).toBe('R3');
  });

  test('handles empty base and empty access ids', () => {
    expect(mergeAccessIntoUiResources([], [])).toEqual([]);
    expect(mergeAccessIntoUiResources([], ['R1'])).toEqual([
      { Mode: 'allow', IrApplicationId: null, IrUiResourceId: 'R1' },
    ]);
  });
});

describe('wantsAccessField', () => {
  test('returns false for null/undefined', () => {
    expect(wantsAccessField(null)).toBe(false);
    expect(wantsAccessField(undefined)).toBe(false);
  });

  test('returns true for exact string match', () => {
    expect(wantsAccessField('AccessUiResourceIds')).toBe(true);
  });

  test('returns false for different string', () => {
    expect(wantsAccessField('OtherField')).toBe(false);
  });

  test('returns true if any element in array matches', () => {
    expect(wantsAccessField(['Id', 'AccessUiResourceIds'])).toBe(true);
  });

  test('returns false if no element matches', () => {
    expect(wantsAccessField(['Id', 'Name'])).toBe(false);
  });

  test('recurse into nested fields objects', () => {
    expect(wantsAccessField({ fields: ['AccessUiResourceIds'] })).toBe(true);
    expect(wantsAccessField({ fields: ['Id'] })).toBe(false);
    expect(wantsAccessField({ fields: [{ fields: ['AccessUiResourceIds'] }] })).toBe(true);
  });

  test('returns false for non-matching object', () => {
    expect(wantsAccessField({ x: 1 })).toBe(false);
  });
});
