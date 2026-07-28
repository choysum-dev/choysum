// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  getActiveCompanyId,
  getContextLang,
  getEnabledCompanyIds,
  normalizeWithCompanyPatch,
  withCompany,
  withContext,
} from './index';

test('normalizeWithCompanyPatch covers string, object, omit-enabled, trim, and dedupe', () => {
  expect(normalizeWithCompanyPatch(' C1 ')).toEqual({
    activeCompanyId: 'C1',
    enabledCompanyIds: ['C1'],
  });

  expect(normalizeWithCompanyPatch({ activeCompanyId: 'C1' })).toEqual({
    activeCompanyId: 'C1',
    enabledCompanyIds: ['C1'],
  });

  expect(
    normalizeWithCompanyPatch({
      activeCompanyId: ' C1 ',
      enabledCompanyIds: [' C1 ', 'C2', 'C1', '  '],
    })
  ).toEqual({
    activeCompanyId: 'C1',
    enabledCompanyIds: ['C1', 'C2'],
  });
});

test('normalizeWithCompanyPatch rejects blank activeCompanyId', () => {
  expect(() => normalizeWithCompanyPatch('')).toThrow('non-empty activeCompanyId');
  expect(() => normalizeWithCompanyPatch('   ')).toThrow('non-empty activeCompanyId');
  expect(() => normalizeWithCompanyPatch({ activeCompanyId: '  ' })).toThrow('non-empty activeCompanyId');
  expect(() => normalizeWithCompanyPatch(null as any)).toThrow('non-empty activeCompanyId');
});

test('normalizeWithCompanyPatch keeps explicit enabled that omit active (caller misuse)', () => {
  expect(
    normalizeWithCompanyPatch({
      activeCompanyId: 'C1',
      enabledCompanyIds: ['C2', 'C3'],
    })
  ).toEqual({
    activeCompanyId: 'C1',
    enabledCompanyIds: ['C2', 'C3'],
  });
});

test('normalizeWithCompanyPatch covers non-string primitives and non-array enabled', () => {
  // Non-string / non-object target (typeof !== 'object') and non-null trimNonEmpty arm.
  expect(normalizeWithCompanyPatch(42 as any)).toEqual({
    activeCompanyId: '42',
    enabledCompanyIds: ['42'],
  });
  expect(() => normalizeWithCompanyPatch(undefined as any)).toThrow('non-empty activeCompanyId');

  // Non-array enabledCompanyIds wraps into a one-element list.
  expect(
    normalizeWithCompanyPatch({
      activeCompanyId: 'C1',
      enabledCompanyIds: ' C2 ' as any,
    })
  ).toEqual({
    activeCompanyId: 'C1',
    enabledCompanyIds: ['C2'],
  });

  // Non-string enabled items exercise trimNonEmpty's String(value) path.
  expect(
    normalizeWithCompanyPatch({
      activeCompanyId: 'C1',
      enabledCompanyIds: [7, 'C1', null, 'C2'] as any,
    })
  ).toEqual({
    activeCompanyId: 'C1',
    enabledCompanyIds: ['7', 'C1', 'C2'],
  });
});

test('withCompany overrides company getters and restores outer view', async () => {
  const globalAny = globalThis as any;
  const hadPrev = Object.prototype.hasOwnProperty.call(globalAny, '$choysum');
  const prev = globalAny.$choysum;
  globalAny.$choysum = {
    request: {
      context: {
        ctx: {
          activeCompanyId: 'OUTER',
          enabledCompanyIds: ['OUTER', 'OTHER'],
          lang: 'en',
        },
      },
    },
  };

  try {
    expect(getActiveCompanyId()).toBe('OUTER');
    expect(getEnabledCompanyIds()).toEqual(['OUTER', 'OTHER']);

    const sync = withCompany(' C-IN ', () => ({
      companyId: getActiveCompanyId(),
      companyIds: getEnabledCompanyIds(),
      lang: getContextLang(),
    }));
    expect(sync).toEqual({ companyId: 'C-IN', companyIds: ['C-IN'], lang: 'en' });
    expect(getActiveCompanyId()).toBe('OUTER');
    expect(getEnabledCompanyIds()).toEqual(['OUTER', 'OTHER']);

    const multi = withCompany({ activeCompanyId: 'A', enabledCompanyIds: ['A', 'B'] }, () => ({
      companyId: getActiveCompanyId(),
      companyIds: getEnabledCompanyIds(),
    }));
    expect(multi).toEqual({ companyId: 'A', companyIds: ['A', 'B'] });

    const nested = await withContext({ lang: 'zh-CN' }, async () => {
      expect(getContextLang()).toBe('zh-CN');
      return withCompany('NEST', async () => {
        expect(getActiveCompanyId()).toBe('NEST');
        expect(getEnabledCompanyIds()).toEqual(['NEST']);
        expect(getContextLang()).toBe('zh-CN');
        await Promise.resolve();
        return getContextLang();
      });
    });
    expect(nested).toBe('zh-CN');
    expect(getActiveCompanyId()).toBe('OUTER');
    expect(getContextLang()).toBe('en');
  } finally {
    if (hadPrev) globalAny.$choysum = prev;
    else delete globalAny.$choysum;
  }
});

test('withCompany works on process stack without jsCtx', () => {
  const globalAny = globalThis as any;
  const hadPrev = Object.prototype.hasOwnProperty.call(globalAny, '$choysum');
  const prev = globalAny.$choysum;
  delete globalAny.$choysum;

  try {
    const value = withCompany('PROC', () => getActiveCompanyId());
    expect(value).toBe('PROC');
    expect(getActiveCompanyId()).toBeUndefined();
  } finally {
    if (hadPrev) globalAny.$choysum = prev;
    else delete globalAny.$choysum;
  }
});
