// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getServiceFactory, registerServiceFactory, unregisterServiceFactory } from '@/core/service/rpc';
import { resolveEffectiveModelId, resolveEffectiveModelRow } from './_resolve_effective_model';

function withMockedMetaSearch(search: (cond: any, opts: any) => Promise<any[]>): () => void {
  const original = getServiceFactory('meta.MetaModel');
  registerServiceFactory('meta.MetaModel', () => ({ Search: search }));
  return () => {
    if (original) registerServiceFactory('meta.MetaModel', original);
    else unregisterServiceFactory('meta.MetaModel');
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

test('ModuleID key and object ModuleId with null/missing id count as empty', async () => {
  const restore = withMockedMetaSearch(async () => [
    { Id: 'mm_shell', ModuleID: 'mod1', UpdatedAt: 100 },
    { Id: 'mm_obj_null', ModuleId: { id: null }, UpdatedAt: 10 },
    { Id: 'mm_obj_empty', ModuleId: {}, UpdatedAt: 20 },
  ]);
  try {
    // Newest empty ModuleId wins among empty candidates (mm_obj_empty @20).
    expect(await resolveEffectiveModelId('demo', 'Widget')).toBe('mm_obj_empty');
  } finally {
    restore();
  }
});

test('pickEffectiveAmong keeps best empty when later shell is newer', async () => {
  const restore = withMockedMetaSearch(async () => [
    { Id: 'mm_e2', ModuleId: '', UpdatedAt: 1 },
    { Id: 'mm_shell', ModuleId: 'mod1', UpdatedAt: 999 },
  ]);
  try {
    expect(await resolveEffectiveModelId('demo', 'Widget')).toBe('mm_e2');
  } finally {
    restore();
  }
});

test('pickEffectiveAmong keeps newer UpdatedAt and larger Id on ties', async () => {
  const restore = withMockedMetaSearch(async () => [
    { Id: 'mm_new', ModuleId: '', UpdatedAt: 100 },
    { Id: 'mm_old', ModuleId: '', UpdatedAt: 50 },
    { Id: 'mm_a', ModuleId: '', UpdatedAt: 100 },
    { Id: 'mm_z', ModuleId: '', UpdatedAt: 100 },
  ]);
  try {
    // Same ts → larger Id wins (mm_z > mm_new > mm_a).
    expect(await resolveEffectiveModelId('demo', 'Widget')).toBe('mm_z');
  } finally {
    restore();
  }
});

test('resolveEffectiveModelRow merges custom fields into Search selection', async () => {
  let seenFields: string[] | undefined;
  const restore = withMockedMetaSearch(async (_c, opts) => {
    seenFields = opts?.fields as string[];
    return [{ Id: 'mm_custom', ModuleId: '', UpdatedAt: 1, Name: 'Widget' }];
  });
  try {
    const row = await resolveEffectiveModelRow('demo', 'Widget', ['Name']);
    expect(row?.Name).toBe('Widget');
    for (const f of ['Id', 'ModuleId', 'UpdatedAt', 'Name']) {
      if (!seenFields || !seenFields.includes(f)) {
        throw new Error(`expected Search fields to include ${f}, got ${JSON.stringify(seenFields)}`);
      }
    }
  } finally {
    restore();
  }
});

test('rowUpdatedAt treats missing UpdatedAt/updated_at as zero during pick', async () => {
  const restore = withMockedMetaSearch(async () => [
    { Id: 'mm_nots', ModuleId: '' },
    { Id: 'mm_ts', ModuleId: '', UpdatedAt: 10 },
  ]);
  try {
    expect(await resolveEffectiveModelId('demo', 'Widget')).toBe('mm_ts');
  } finally {
    restore();
  }
});

test('resolveEffectiveModelRow tolerates null Search pages', async () => {
  const restore = withMockedMetaSearch(async () => null as any);
  try {
    expect(await resolveEffectiveModelRow('demo', 'Widget')).toBeUndefined();
    expect(await resolveEffectiveModelId('demo', 'Widget')).toBe('');
  } finally {
    restore();
  }
});
