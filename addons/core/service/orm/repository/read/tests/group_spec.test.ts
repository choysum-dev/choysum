// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { normalizeFieldAggregation, normalizeGroupBySpec, normalizeGroupBySpecs, rebuildCompositeGroupSpec } from '../group_spec';

test('repository group spec normalizes group and aggregation specs into stable aliases', () => {
  const month = normalizeGroupBySpec('CreatedAt:month' as any);
  expect(month).toEqual({
    field: 'CreatedAt',
    granularity: 'month',
    alias: 'CreatedAt__month',
    isTime: true,
  });

  const composite = normalizeGroupBySpecs(['CreatedAt:month' as any, 'Status' as any]) as any;
  expect(composite.composite).toBe(true);
  expect(composite.parts).toEqual([
    {
      field: 'CreatedAt',
      granularity: 'month',
      alias: 'CreatedAt__month',
      isTime: true,
    },
    {
      field: 'Status',
      alias: 'Status',
      isTime: false,
    },
  ]);

  const rebuilt = rebuildCompositeGroupSpec(composite);
  expect(rebuilt).toEqual([{ field: 'CreatedAt', granularity: 'month', alias: 'CreatedAt__month', range: undefined }, 'Status']);

  const agg = normalizeFieldAggregation({ field: 'Amount', agg: 'sum' } as any);
  expect(agg).toEqual({
    field: 'Amount',
    agg: 'sum',
    alias: 'Amount__sum',
    distinct: false,
  });
});

test('repository group spec covers object alias fallback and single-spec fast path', () => {
  expect(normalizeGroupBySpec({ field: 'CreatedAt', granularity: 'day' } as any)).toEqual({
    field: 'CreatedAt',
    granularity: 'day',
    alias: 'CreatedAt__day',
    isTime: true,
    range: undefined,
  });

  expect(normalizeGroupBySpecs([{ field: 'Status', alias: 'status_alias' } as any] as any)).toEqual({
    field: 'Status',
    granularity: undefined,
    alias: 'status_alias',
    isTime: false,
    range: undefined,
  });
});

test('repository group spec rejects invalid string aggregation format', () => {
  expect(() => normalizeFieldAggregation('invalid_agg' as any)).toThrow('Invalid aggregation: invalid_agg');
});

test('repository group spec object fallback uses field name alias when granularity is absent', () => {
  expect(normalizeGroupBySpec({ field: 'Category' } as any)).toEqual({
    field: 'Category',
    granularity: undefined,
    alias: 'Category',
    isTime: false,
    range: undefined,
  });
});
