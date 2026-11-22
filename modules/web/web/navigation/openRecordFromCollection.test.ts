// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import { applyIdToPathTemplate, canOpenDetailRow, resolveDetailPath } from './detailPath';
import { openRecordFromCollection } from './openRecordFromCollection';
import { snapshotFromListStore } from './snapshotFromListStore';
import { useRecordSetStore } from '@/web/web/stores/recordSetStore';

test('applyIdToPathTemplate replaces :id and appends bare prefixes', () => {
  expect(applyIdToPathTemplate('/base/countries/:id', 'c1')).toBe('/base/countries/c1');
  expect(applyIdToPathTemplate('/base/countries/{id}', 'c1')).toBe('/base/countries/c1');
  expect(applyIdToPathTemplate('/base/countries', 'c1')).toBe('/base/countries/c1');
  expect(applyIdToPathTemplate('/base/countries/c1', 'c1')).toBe('/base/countries/c1');
  expect(applyIdToPathTemplate('/base/countries?tab=details', 'c1')).toBe(
    '/base/countries/c1?tab=details'
  );
  expect(applyIdToPathTemplate('/base/countries#sec', 'c1')).toBe('/base/countries/c1#sec');
  expect(applyIdToPathTemplate('/base/countries/:id', 'a$b')).toBe('/base/countries/a%24b');
  expect(applyIdToPathTemplate('/base/countries/c1?tab=x', 'c1')).toBe('/base/countries/c1?tab=x');
});

test('resolveDetailPath supports templates and factories', () => {
  expect(resolveDetailPath('/base/countries/:id', { Id: 'x' })).toBe('/base/countries/x');
  expect(
    resolveDetailPath(row => `/web/app-settings/${row.App}/${row.Id}`, { Id: '1', App: 'web' })
  ).toBe('/web/app-settings/web/1');
  expect(resolveDetailPath(() => null, { Id: '1' })).toBeNull();
  expect(
    resolveDetailPath(() => ({ name: 'MetaModelDataDetail', params: { id: '1' } }), { Id: '1' })
  ).toEqual({ name: 'MetaModelDataDetail', params: { id: '1' } });
});

test('canOpenDetailRow gates navigation', () => {
  expect(canOpenDetailRow(undefined, { Id: '1' })).toBe(true);
  expect(canOpenDetailRow(false, { Id: '1' })).toBe(false);
  expect(canOpenDetailRow(row => String(row.Id) === 'ok', { Id: 'ok' })).toBe(true);
  expect(canOpenDetailRow(row => String(row.Id) === 'ok', { Id: 'no' })).toBe(false);
  // Parameterless ACL still evaluates when row is missing.
  expect(canOpenDetailRow(() => true, null)).toBe(true);
  expect(canOpenDetailRow(() => false, undefined)).toBe(false);
});

test('snapshotFromListStore builds global index and page ids', () => {
  const store = {
    modelName: 'base.Country',
    state: {
      queryState: {
        appliedFilters: [],
        pagination: { offset: 100, limit: 100 },
        orderBy: [{ field: 'Name', direction: 'asc' }],
        forcedCondition: { IsActive: true },
      },
      result: {
        total: 249,
        rows: [{ Id: 'a' }, { Id: 'b' }, { Id: 'c' }],
      },
    },
    metadata: { fields: {} },
  };
  const entry = snapshotFromListStore(store, {
    model: 'base.Country',
    key: 'base.Country|list',
    rowIndex: 1,
    recordId: 'b',
  });
  expect(entry?.total).toBe(249);
  expect(entry?.index).toBe(101);
  expect(entry?.ids).toEqual(['a', 'b', 'c']);
  expect(entry?.windowStart).toBe(100);
  expect(entry?.condition).toEqual({ IsActive: true });

  // Missing Search total must still span offset + page before clamping index.
  const noTotal = snapshotFromListStore(
    {
      ...store,
      state: {
        ...store.state,
        result: { rows: [{ Id: 'a' }, { Id: 'b' }, { Id: 'c' }] },
      },
    },
    { model: 'base.Country', key: 'k', rowIndex: 2, recordId: 'c' }
  );
  expect(noTotal?.total).toBe(103);
  expect(noTotal?.index).toBe(102);

  expect(snapshotFromListStore(null, { model: 'x', key: 'k', rowIndex: 0 })).toBeNull();
  expect(
    snapshotFromListStore({ state: {} }, { model: '', key: 'k', rowIndex: 0 })
  ).toBeNull();
  // type:record wrapper + empty total clamps index to 0.
  const wrapped = snapshotFromListStore(
    {
      modelName: 'base.Country',
      state: {
        queryState: { pagination: { offset: 0 } },
        result: {
          rows: [{ type: 'record', record: { Id: 'z' } }, { kind: 'more' }],
        },
      },
      metadata: { fields: {} },
    },
    { model: 'base.Country', key: 'k', rowIndex: 0, recordId: 'z' }
  );
  expect(wrapped?.ids).toEqual(['z']);
  expect(wrapped?.total).toBe(1);
  expect(wrapped?.index).toBe(0);
});

test('openRecordFromCollection seeds store and pushes route', async () => {
  setActivePinia(createPinia());
  const pushed: unknown[] = [];
  openRecordFromCollection({
    router: { push: to => pushed.push(to) },
    to: '/base/countries/b',
    entry: {
      key: 'k',
      model: 'base.Country',
      condition: {},
      total: 3,
      index: 1,
      ids: ['a', 'b', 'c'],
      windowStart: 0,
    },
  });
  expect(useRecordSetStore().active?.model).toBe('base.Country');
  expect(pushed).toEqual(['/base/countries/b']);

  // Named route object target.
  openRecordFromCollection({
    router: { push: to => pushed.push(to) },
    to: { name: 'CountryDetail', params: { id: 'c' } },
    entry: {
      key: 'k2',
      model: 'base.Country',
      condition: {},
      total: 1,
      index: 0,
      ids: ['c'],
      windowStart: 0,
    },
  });
  expect(pushed[pushed.length - 1]).toEqual({ name: 'CountryDetail', params: { id: 'c' } });

  // Failed push clears the seeded session.
  openRecordFromCollection({
    router: { push: () => Promise.reject(new Error('aborted')) },
    to: '/base/countries/x',
    entry: {
      key: 'k3',
      model: 'base.Country',
      condition: {},
      total: 1,
      index: 0,
      ids: ['x'],
      windowStart: 0,
    },
  });
  await Promise.resolve();
  await Promise.resolve();
  expect(useRecordSetStore().active).toBeNull();

  // Fulfilled NavigationFailure also clears the seed for that open.
  const VueRouter = await import('vue-router');
  const { router: stub } = (VueRouter as any).createFeStubRouter({
    route: { path: '/', fullPath: '/' },
  });
  stub.beforeEach(() => false);
  const navFailure = await stub.push('/blocked');
  expect((VueRouter as any).isNavigationFailure(navFailure)).toBe(true);
  openRecordFromCollection({
    router: { push: () => Promise.resolve(navFailure) },
    to: '/base/countries/fail',
    entry: {
      key: 'k-fail',
      model: 'base.Country',
      condition: {},
      total: 1,
      index: 0,
      ids: ['fail'],
      windowStart: 0,
    },
  });
  await Promise.resolve();
  await Promise.resolve();
  expect(useRecordSetStore().active).toBeNull();

  // Sync push throw clears the seed.
  openRecordFromCollection({
    router: {
      push: () => {
        throw new Error('sync boom');
      },
    },
    to: '/base/countries/sync',
    entry: {
      key: 'k-sync',
      model: 'base.Country',
      condition: {},
      total: 1,
      index: 0,
      ids: ['sync'],
      windowStart: 0,
    },
  });
  expect(useRecordSetStore().active).toBeNull();

  // Failed push must not clear a newer session (same collection key, newer generation).
  let rejectFirst: (err: Error) => void = () => undefined;
  const firstPush = new Promise((_, reject) => {
    rejectFirst = reject as (err: Error) => void;
  });
  openRecordFromCollection({
    router: { push: () => firstPush },
    to: '/base/countries/old',
    entry: {
      key: 'same',
      model: 'base.Country',
      condition: {},
      total: 1,
      index: 0,
      ids: ['old'],
      windowStart: 0,
    },
  });
  openRecordFromCollection({
    router: { push: to => pushed.push(to) },
    to: '/base/countries/new',
    entry: {
      key: 'same',
      model: 'base.Country',
      condition: {},
      total: 1,
      index: 0,
      ids: ['new'],
      windowStart: 0,
    },
  });
  expect(useRecordSetStore().active?.ids).toEqual(['new']);
  rejectFirst(new Error('aborted'));
  await Promise.resolve();
  await Promise.resolve();
  expect(useRecordSetStore().active?.ids).toEqual(['new']);
});

test('snapshotFromListStore pageIds override ignores group-only result.rows', () => {
  const entry = snapshotFromListStore(
    {
      modelName: 'base.Country',
      state: {
        queryState: { pagination: { offset: 100 }, orderBy: [{ field: 'Name', direction: 'asc' }] },
        result: {
          total: 40,
          rows: [{ kind: 'group', key: 'g1' }],
        },
      },
      metadata: { fields: {} },
    },
    {
      model: 'base.Country',
      key: 'k',
      rowIndex: 0,
      recordId: 'b',
      pageIds: ['a', 'b', 'c'],
    }
  );
  expect(entry?.ids).toEqual(['a', 'b', 'c']);
  expect(entry?.windowStart).toBe(0);
  expect(entry?.index).toBe(1);
  expect(entry?.total).toBe(40);
});

test('snapshotFromListStore empty total and unmatched recordId', () => {
  const empty = snapshotFromListStore(
    {
      modelName: 'base.Country',
      state: {
        queryState: { pagination: { offset: 5 } },
        result: { total: 0, rows: [] },
      },
      metadata: { fields: {} },
    },
    { model: 'base.Country', key: 'k', rowIndex: 2, recordId: 'missing' }
  );
  expect(empty?.total).toBe(5);
  // unmatched id keeps offset+rowIndex then clamps into [0, total-1]
  expect(empty?.index).toBe(4);

  const zero = snapshotFromListStore(
    {
      modelName: 'base.Country',
      state: {
        queryState: { pagination: { offset: 0 } },
        result: { rows: [] },
      },
      metadata: { fields: {} },
    },
    { model: 'base.Country', key: 'k', rowIndex: 0 }
  );
  expect(zero?.total).toBe(0);
  expect(zero?.index).toBe(0);

  // Non-record objects without Id/id are skipped (unwrap returns null).
  const skipped = snapshotFromListStore(
    {
      modelName: 'base.Country',
      state: {
        queryState: { pagination: { offset: 0 } },
        result: { rows: [{ Name: 'no-id' }, null, 'x', { Id: 'kept' }] },
      },
      metadata: { fields: {} },
    },
    { model: 'base.Country', key: 'k', rowIndex: 0, recordId: 'kept' }
  );
  expect(skipped?.ids).toEqual(['kept']);
});
