// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  cloneFilter,
  countFilterLeaves,
  createCondition,
  createFilter,
  deepCloneFilter,
  genId,
  isCondition,
  isConditionComplete,
  isGroup,
  normalizeFilters,
  toFilters,
} from '@/web/web/query/utils/filter/structures';

test('filter structures helpers: genId returns unique local ids', () => {
  expect(genId()).not.toBe(genId());
});

test('filter structures helpers: isGroup / isCondition discriminate nodes', () => {
  const cond = createCondition('Name', '=', 'a');
  const group = createFilter('And', [cond]);
  expect(isCondition(cond)).toBe(true);
  expect(isGroup(cond)).toBe(false);
  expect(isGroup(group)).toBe(true);
  expect(isCondition(group)).toBe(false);
});

test('filter structures helpers: deepCloneFilter clones dates and nested values', () => {
  const d = new Date('2026-01-01T00:00:00Z');
  const original = createFilter('And', [
    createCondition('CreatedAt', '=', d),
    createCondition('Tags', 'in', [{ id: 1 }, 'x']),
  ]);
  const nested = createFilter('Or', [createCondition('Name', '=', 'n')]);
  original.children.push(nested);

  const cloned = deepCloneFilter(original);
  expect(cloned).not.toBe(original);
  expect((cloned.children[0] as any).value).toBeInstanceOf(Date);
  expect((cloned.children[0] as any).value).not.toBe(d);
  expect((cloned.children[0] as any).value.getTime()).toBe(d.getTime());
  expect((cloned.children[1] as any).value[0]).not.toBe((original.children[1] as any).value[0]);
  expect(cloned.children[2]).not.toBe(nested);
});

test('filter structures helpers: cloneFilter regenerates ids', () => {
  const original = createFilter('And', [createCondition('Name', '=', 'a')]);
  const cloned = cloneFilter(original);
  expect(cloned.id).not.toBe(original.id);
  expect((cloned.children[0] as any).id).not.toBe((original.children[0] as any).id);
  expect((cloned.children[0] as any).field).toBe('Name');
});

test('filter structures helpers: toFilters converts named filters and groups', () => {
  expect(toFilters(null)).toEqual([]);
  expect(toFilters(undefined)).toEqual([]);
  const named = toFilters({ name: 'Active', query: ['Active', '=', true] } as any);
  expect(named).toHaveLength(1);
  expect(named[0].name).toBe('Active');
  expect((named[0].children[0] as any).field).toBe('Active');

  const nested = toFilters({
    name: 'Multi',
    query: { And: [['Active', '=', true], { Or: [['Name', 'ilike', '%a%'], ['Code', '=', 'x']] }] },
  } as any);
  expect(nested).toHaveLength(1);
  expect(nested[0].name).toBe('Multi');
  expect(nested[0].logic).toBe('And');
  expect(nested[0].children).toHaveLength(2);
  expect(isGroup(nested[0].children[1])).toBe(true);

  const group = createFilter('Or', [createCondition('Code', '=', 'x')]);
  expect(toFilters(group)[0]).toBe(group);
  expect(toFilters({ name: '', query: ['A', '=', 1] } as any)).toEqual([]);

  // ConditionGroup with a spurious query property must still be kept as a group.
  const groupWithQuery = {
    ...createFilter('And', [createCondition('Name', '=', 'n')]),
    query: ['bogus', '=', 1],
  } as any;
  expect(toFilters(groupWithQuery)[0]).toBe(groupWithQuery);
});

test('filter structures helpers: isConditionComplete requires value when operator needs one', () => {
  expect(isConditionComplete({ field: 'Name', operator: '=', value: 'x' })).toBe(true);
  expect(isConditionComplete({ field: 'Name', operator: '=', value: '' })).toBe(false);
  expect(isConditionComplete({ field: 'Name', operator: '=', value: '   ' })).toBe(false);
  expect(isConditionComplete({ field: '', operator: '=', value: 'x' })).toBe(false);
  expect(isConditionComplete({ field: 'Active', operator: 'is', value: null })).toBe(true);
  expect(isConditionComplete({ field: 'PartnerId', operator: '=', value: { Id: 'p1' } })).toBe(true);
  expect(isConditionComplete({ field: 'PartnerId', operator: '=', value: { Id: '' } })).toBe(false);
  expect(isConditionComplete({ field: 'PartnerId', operator: 'in', value: [] })).toBe(false);
});

test('filter structures helpers: normalizeFilters drops incomplete nodes and empty groups', () => {
  const raw = [
    {
      id: 'g1',
      logic: 'OR' as any,
      name: 'N',
      children: [
        { id: 'c1', field: '', operator: '=', value: 1 },
        { id: 'c1b', field: 'Name', operator: '=', value: '' },
        { id: 'c2', field: 'Name', operator: '=', value: 'ok' },
        {
          id: 'g2',
          logic: 'And',
          children: [{ id: 'c3', field: 'Code', operator: 'like', value: '%x%' }],
        },
        null,
      ],
    },
    { id: 'empty', logic: 'And', children: [] },
    null,
  ] as any;
  const out = normalizeFilters(raw);
  expect(out).toHaveLength(1);
  expect(out[0].logic).toBe('Or');
  expect(out[0].name).toBe('N');
  // Empty value kept for external filters; incomplete field dropped.
  expect(out[0].children).toHaveLength(3);
  expect((out[0].children[0] as any).field).toBe('Name');
  expect((out[0].children[0] as any).value).toBe('');
  expect((out[0].children[1] as any).field).toBe('Name');
  expect(isGroup(out[0].children[2])).toBe(true);

  const leaves = countFilterLeaves({
    id: 'g',
    logic: 'And',
    children: [
      { id: 'a', field: 'Name', operator: '=', value: 'ok' },
      { id: 'b', field: 'Name', operator: '=', value: '' },
    ],
  } as any);
  expect(leaves).toEqual({ complete: 1, incomplete: 1 });
});

test('filter structures helpers: deepCloneFilter preserves name and handles missing children', () => {
  const named = { id: 'g', logic: 'And' as const, name: 'N', children: undefined as any };
  const cloned = deepCloneFilter(named as any);
  expect(cloned.name).toBe('N');
  expect(cloned.children).toEqual([]);

  const arr = toFilters([
    { name: 'A', query: ['X', '=', 1] } as any,
    { name: 'skip' } as any,
    createFilter('And', [createCondition('Y', '=', 2)]),
  ]);
  expect(arr).toHaveLength(2);
  expect(arr[0].name).toBe('A');
});

test('filter structures helpers: toFilters skips null query and blank name, throws on invalid query shapes', () => {
  expect(toFilters({ name: 'NullQ', query: null } as any)).toEqual([]);
  expect(toFilters({ name: '', query: ['A', '=', 1] } as any)).toEqual([]);
  // Empty queries (groupby-only favorites / filtersToQuery []) apply as no-op chips.
  expect(toFilters({ name: 'EmptyArr', query: [] } as any)).toEqual([]);
  expect(toFilters({ name: 'EmptyObj', query: {} } as any)).toEqual([]);
  expect(toFilters({ name: 'EmptyAnd', query: { And: [] } } as any)).toEqual([]);
  expect(toFilters({ name: 'EmptyOr', query: { Or: [] } } as any)).toEqual([]);

  for (const bad of [
    { name: 'BadArr', query: ['only', 'two'] },
    { name: 'Str', query: 'nope' },
    { name: 'JunkParts', query: { And: [null, 'x', { foo: 1 }] } },
    { name: 'NonStringLeaf', query: [1, '=', 2] },
    { name: 'NonStringOp', query: ['A', 1, 2] },
  ]) {
    expect(() => toFilters(bad as any)).toThrow(/invalid_named_filter_query/);
  }

  expect(() => toFilters(['primitive'] as any)).toThrow(/invalid_named_filter_query/);
  expect(() => toFilters([42] as any)).toThrow(/invalid_named_filter_query/);
  // Nested array entries must not be treated as incomplete NamedFilter objects.
  expect(() => toFilters([['nested']] as any)).toThrow(/invalid_named_filter_query/);
  // Partially invalid And/Or trees must fail the whole named filter.
  expect(() =>
    toFilters({
      name: 'Mixed',
      query: { And: [['Name', '=', 'x'], ['only', 'two']] },
    } as any)
  ).toThrow(/invalid_named_filter_query/);

  const groupShaped = toFilters({
    name: 'AlreadyGroup',
    query: { id: 'g', logic: 'Or', children: [{ id: 'c', field: 'Name', operator: '=', value: 'a' }] },
  } as any);
  expect(groupShaped).toHaveLength(1);
  expect(groupShaped[0].name).toBe('AlreadyGroup');
  expect(groupShaped[0].logic).toBe('Or');

  const orTree = toFilters({
    name: 'OrTree',
    query: { Or: [['A', '=', 1], ['B', '=', 2]] },
  } as any);
  expect(orTree[0].logic).toBe('Or');
  expect(orTree[0].children).toHaveLength(2);

  const nestedMulti = toFilters({
    name: 'KeepSub',
    query: { And: [{ And: [['X', '=', 1], ['Y', '=', 2]] }] },
  } as any);
  expect(isGroup(nestedMulti[0].children[0])).toBe(true);

  // Parts prefer And when both keys exist; logic still follows orParts presence.
  const bothKeys = toFilters({
    name: 'Both',
    query: { And: [['A', '=', 1]], Or: [['B', '=', 2]] },
  } as any);
  expect(bothKeys[0].children).toHaveLength(1);
  expect((bothKeys[0].children[0] as any).field).toBe('A');
  expect(bothKeys[0].logic).toBe('Or');

  // Named single-child subgroup is not flattened.
  const namedChild = toFilters({
    name: 'Outer',
    query: {
      And: [
        {
          id: 'g',
          logic: 'And',
          name: 'Keep',
          children: [{ id: 'c', field: 'X', operator: '=', value: 1 }],
        },
      ],
    },
  } as any);
  expect(isGroup(namedChild[0].children[0])).toBe(true);
  expect((namedChild[0].children[0] as any).name).toBe('Keep');
});

