// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getServiceFactory, registerServiceFactory } from '@/core/service/rpc';
import { resolveEffectiveModelId, resolveEffectiveModelRow } from './_resolve_effective_model';

function withMockedMetaSearch(search: (cond: any, opts: any) => Promise<any[]>): () => void {
  const original = getServiceFactory('meta.MetaModel');
  registerServiceFactory('meta.MetaModel', () => ({ Search: search }));
  return () => {
    if (original) registerServiceFactory('meta.MetaModel', original);
  };
}

test('resolveEffectiveModelId returns empty string when no rows', async () => {
  const restore = withMockedMetaSearch(async () => []);
  try {
    expect(await resolveEffectiveModelId('demo', 'Widget')).toBe('');
    expect(await resolveEffectiveModelRow('demo', 'Widget')).toBeUndefined();
  } finally {
    restore();
  }
});

test('resolveEffectiveModelRow returns the single matching row', async () => {
  const only = { Id: 'mm_only', ModuleId: 'mod1', UpdatedAt: '2026-01-01T00:00:00Z' };
  const restore = withMockedMetaSearch(async () => [only]);
  try {
    expect(await resolveEffectiveModelRow('demo', 'Widget')).toEqual(only);
    expect(await resolveEffectiveModelId('demo', 'Widget')).toBe('mm_only');
  } finally {
    restore();
  }
});

test('empty ModuleId wins over non-empty even with older UpdatedAt', async () => {
  const restore = withMockedMetaSearch(async () => [
    { Id: 'mm_shell', ModuleId: 'mod1', UpdatedAt: '2026-06-01T00:00:00Z' },
    { Id: 'mm_e2', ModuleId: '', UpdatedAt: '2020-01-01T00:00:00Z' },
  ]);
  try {
    expect(await resolveEffectiveModelId('demo', 'Widget')).toBe('mm_e2');
  } finally {
    restore();
  }
});

test('object ModuleId with empty Id counts as empty', async () => {
  const restore = withMockedMetaSearch(async () => [
    { Id: 'mm_shell', ModuleId: { Id: 'mod1' }, UpdatedAt: 100 },
    { Id: 'mm_e2', ModuleId: { Id: '  ' }, UpdatedAt: 1 },
  ]);
  try {
    expect(await resolveEffectiveModelId('demo', 'Widget')).toBe('mm_e2');
  } finally {
    restore();
  }
});

test('UpdatedAt string/number/invalid and Id tie-break', async () => {
  const restore = withMockedMetaSearch(async () => [
    { Id: 'mm_a', ModuleId: null, UpdatedAt: 'not-a-date' },
    { Id: 'mm_b', ModuleId: '', UpdatedAt: 50 },
    { Id: 'mm_c', module_id: '', updated_at: '2026-01-01T00:00:00.000Z' },
    { Id: 'mm_d', ModuleId: '', UpdatedAt: '' },
    { Id: 'mm_e', ModuleId: '', UpdatedAt: '2026-01-01T00:00:00.000Z' },
  ]);
  try {
    // Newest valid UpdatedAt wins; mm_c and mm_e share the same ts → larger Id (mm_e).
    expect(await resolveEffectiveModelId('demo', 'Widget')).toBe('mm_e');
  } finally {
    restore();
  }
});

test('pagination fetches beyond 500 rows before picking', async () => {
  const page1 = Array.from({ length: 500 }, (_, i) => ({
    Id: `mm_${String(i).padStart(4, '0')}`,
    ModuleId: 'shell',
    UpdatedAt: 1,
  }));
  const page2 = [
    { Id: 'mm_0500', ModuleId: 'shell', UpdatedAt: 1 },
    { Id: 'mm_winner', ModuleId: null, UpdatedAt: 1 },
  ];
  const calls: number[] = [];
  const restore = withMockedMetaSearch(async (_c, opts) => {
    const offset = Number(opts?.offset || 0);
    calls.push(offset);
    if (offset === 0) return page1;
    if (offset === 500) return page2;
    return [];
  });
  try {
    expect(await resolveEffectiveModelId('demo', 'Widget')).toBe('mm_winner');
    expect(calls).toEqual([0, 500]);
  } finally {
    restore();
  }
});

test('rows without Id are filtered out', async () => {
  const restore = withMockedMetaSearch(async () => [{ Name: 'no-id' }, { Id: '  ', ModuleId: '' }]);
  try {
    expect(await resolveEffectiveModelId('demo', 'Widget')).toBe('');
  } finally {
    restore();
  }
});
