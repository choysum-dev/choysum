// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { withContext } from '../../../../runtime/context';
import {
  applyCompanyDependentFieldsForWrite,
  applyFieldCompanyValuesPatch,
  decodeCompanyDependentFieldValue,
  encodeCompanyDependentMapForDb,
  mergeCompanyDependentWrite,
  parseCompanyDependentStoredMap,
  unwrapCompanyDependentValue,
} from '../company_dependent_field_codec';

test('companyDependent codec unwrap is F0 (no fallback)', () => {
  const map = { C1: 12.5, C2: 11 };
  expect(unwrapCompanyDependentValue(map, 'C1')).toBe(12.5);
  expect(unwrapCompanyDependentValue(map, 'MISSING')).toBeNull();
  expect(unwrapCompanyDependentValue(null, 'C1')).toBeNull();
});

test('companyDependent merge: scalar write, null deletes current key only', () => {
  const current = { C1: 1, C2: 2 };
  expect(
    mergeCompanyDependentWrite({
      fieldName: 'Cost',
      value: 9,
      companyId: 'C1',
      currentMap: current,
      mode: 'update',
    })
  ).toEqual({ C1: 9, C2: 2 });

  expect(
    mergeCompanyDependentWrite({
      fieldName: 'Cost',
      value: null,
      companyId: 'C1',
      currentMap: current,
      mode: 'update',
    })
  ).toEqual({ C2: 2 });

  expect(
    mergeCompanyDependentWrite({
      fieldName: 'Cost',
      value: null,
      companyId: 'C2',
      currentMap: { C2: 2 },
      mode: 'update',
    })
  ).toBeNull();
});

test('companyDependent merge: replace null clears whole column', () => {
  expect(
    mergeCompanyDependentWrite({
      fieldName: 'Cost',
      value: null,
      companyId: 'C1',
      currentMap: { C1: 1, C2: 2 },
      mode: 'update',
      replace: true,
    })
  ).toBeNull();
});

test('companyDependent merge: create writes only active company key', () => {
  expect(
    mergeCompanyDependentWrite({
      fieldName: 'Cost',
      value: 3,
      companyId: 'C1',
      currentMap: null,
      mode: 'create',
    })
  ).toEqual({ C1: 3 });
});

test('companyDependent merge: map object merges; replace replaces', () => {
  expect(
    mergeCompanyDependentWrite({
      fieldName: 'Cost',
      value: { C2: 8 },
      companyId: 'C1',
      currentMap: { C1: 1 },
      mode: 'update',
    })
  ).toEqual({ C1: 1, C2: 8 });

  expect(
    mergeCompanyDependentWrite({
      fieldName: 'Cost',
      value: { C2: 8 },
      companyId: 'C1',
      currentMap: { C1: 1 },
      mode: 'update',
      replace: true,
    })
  ).toEqual({ C2: 8 });
});

test('companyDependent patch false deletes key; empty map null', () => {
  expect(
    applyFieldCompanyValuesPatch({
      fieldName: 'Cost',
      currentMap: { C1: 1, C2: 2 },
      values: { C1: false },
    })
  ).toEqual({ C2: 2 });

  expect(
    applyFieldCompanyValuesPatch({
      fieldName: 'Cost',
      currentMap: { C1: 1 },
      values: { C1: false },
    })
  ).toBeNull();
});

test('companyDependent decode respects prefetch_companies', async () => {
  const stored = JSON.stringify({ C1: 'a', C2: 'b' });
  await withContext({ activeCompanyId: 'C1', prefetch_companies: true }, async () => {
    expect(decodeCompanyDependentFieldValue(stored)).toEqual({ C1: 'a', C2: 'b' });
  });
  await withContext({ activeCompanyId: 'C1' }, async () => {
    expect(decodeCompanyDependentFieldValue(stored)).toBe('a');
  });
  await withContext({ activeCompanyId: 'MISSING' }, async () => {
    expect(decodeCompanyDependentFieldValue(stored)).toBeNull();
  });
});

test('companyDependent apply write rewrites payload field to map', () => {
  const meta = {
    fields: new Map([['Cost', { name: 'Cost', type: 'float', companyDependent: true, column: {} }]]),
  } as any;

  const created = applyCompanyDependentFieldsForWrite(meta, { Cost: 4 } as any, {
    mode: 'create',
    companyId: 'C1',
  });
  expect(created).toEqual({ Cost: { C1: 4 } });

  const updated = applyCompanyDependentFieldsForWrite(
    meta,
    { Cost: null } as any,
    { mode: 'update', companyId: 'C1', current: { Cost: { C1: 4, C2: 5 } } }
  );
  expect(updated).toEqual({ Cost: { C2: 5 } });

  expect(encodeCompanyDependentMapForDb({ C1: 1 })).toBe(JSON.stringify({ C1: 1 }));
  expect(encodeCompanyDependentMapForDb(null)).toBeNull();
  expect(parseCompanyDependentStoredMap('{"C1":1}')).toEqual({ C1: 1 });
});
