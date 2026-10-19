// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { groupRowsToChartSeries } from './chartStoreHelpers';

describe('groupRowsToChartSeries', () => {
  test('maps top-level group rows to a count series', () => {
    const { categories, seriesMatrix } = groupRowsToChartSeries(
      [
        { kind: 'group', depth: 0, key: 'a', label: 'Alpha', count: 3 },
        { kind: 'group', depth: 0, key: 'b', label: 'Beta', count: 5 },
        { kind: 'group', depth: 1, key: 'nested', label: 'skip', count: 9 },
      ],
      { metricAlias: 'count', metricLabel: 'Count' },
    );
    expect(categories).toEqual(['Alpha', 'Beta']);
    expect(seriesMatrix).toEqual([{ name: 'Count', data: [3, 5] }]);
  });

  test('reads named metric aliases from metrics map', () => {
    const { categories, seriesMatrix } = groupRowsToChartSeries(
      [
        {
          kind: 'group',
          depth: 0,
          key: 'x',
          label: 'X',
          metrics: { amount_sum: 12.5 },
        },
      ],
      { metricAlias: 'amount_sum', metricLabel: 'Amount' },
    );
    expect(categories).toEqual(['X']);
    expect(seriesMatrix).toEqual([{ name: 'Amount', data: [12.5] }]);
  });

  test('returns empty series when no groups match depth', () => {
    const { categories, seriesMatrix } = groupRowsToChartSeries(
      [{ kind: 'group', depth: 1, key: 'only', label: 'Only', count: 1 }],
      { groupDepth: 0 },
    );
    expect(categories).toEqual([]);
    expect(seriesMatrix).toEqual([]);
  });

  test('maps nested children into multiple series for stack', () => {
    const { categories, seriesMatrix } = groupRowsToChartSeries(
      [
        {
          kind: 'group',
          depth: 0,
          key: 'west',
          label: 'West',
          children: [
            { kind: 'group', depth: 1, key: 'a', label: 'Open', count: 2 },
            { kind: 'group', depth: 1, key: 'b', label: 'Done', count: 4 },
          ],
        },
        {
          kind: 'group',
          depth: 0,
          key: 'east',
          label: 'East',
          children: [
            { kind: 'group', depth: 1, key: 'a', label: 'Open', count: 1 },
            { kind: 'group', depth: 1, key: 'c', label: 'Blocked', count: 3 },
          ],
        },
      ],
      { metricAlias: 'count', metricLabel: 'Count' },
    );
    expect(categories).toEqual(['West', 'East']);
    expect(seriesMatrix).toEqual([
      { name: 'Open', data: [2, 1] },
      { name: 'Done', data: [4, 0] },
      { name: 'Blocked', data: [0, 3] },
    ]);
  });

  test('reads nested named metrics for series values', () => {
    const { categories, seriesMatrix } = groupRowsToChartSeries(
      [
        {
          kind: 'group',
          depth: 0,
          key: 'q1',
          label: 'Q1',
          children: [
            {
              kind: 'group',
              depth: 1,
              key: 'a',
              label: 'A',
              metrics: { amount_sum: 10 },
            },
            {
              kind: 'group',
              depth: 1,
              key: 'b',
              label: 'B',
              metrics: { amount_sum: 20 },
            },
          ],
        },
      ],
      { metricAlias: 'amount_sum', metricLabel: 'Amount' },
    );
    expect(categories).toEqual(['Q1']);
    expect(seriesMatrix).toEqual([
      { name: 'A', data: [10] },
      { name: 'B', data: [20] },
    ]);
  });

  test('preserves snapshot order for temporal-looking keys (view sorts when axis is temporal)', () => {
    const { categories, seriesMatrix } = groupRowsToChartSeries(
      [
        { kind: 'group', depth: 0, key: '2025-10-01T00:00:00Z', label: '2025-Q4', count: 3 },
        { kind: 'group', depth: 0, key: '2025-01-01T00:00:00Z', label: '2025-Q1', count: 1 },
        { kind: 'group', depth: 0, key: '2025-04-01T00:00:00Z', label: '2025-Q2', count: 2 },
      ],
      { metricAlias: 'count', metricLabel: 'Count' },
    );
    expect(categories).toEqual(['2025-Q4', '2025-Q1', '2025-Q2']);
    expect(seriesMatrix).toEqual([{ name: 'Count', data: [3, 1, 2] }]);
  });

  test('coerces non-finite named metrics and blank alias to count', () => {
    const { seriesMatrix } = groupRowsToChartSeries(
      [
        {
          kind: 'group',
          depth: 0,
          key: 'x',
          label: 'X',
          metrics: { amount_sum: Number.NaN },
        },
      ],
      { metricAlias: 'amount_sum', metricLabel: 'Amount' },
    );
    expect(seriesMatrix).toEqual([{ name: 'Amount', data: [0] }]);
    const countFallback = groupRowsToChartSeries(
      [{ kind: 'group', depth: 0, key: 'y', label: 'Y', count: 2 }],
      { metricAlias: '   ', metricLabel: 'Count' },
    );
    expect(countFallback.seriesMatrix[0]!.data).toEqual([2]);
  });

  test('uses key when label missing and skips non-group children', () => {
    const { categories, seriesMatrix } = groupRowsToChartSeries(
      [
        {
          kind: 'group',
          depth: 0,
          key: 'only-key',
          children: [
            { kind: 'record', depth: 1, key: 'skip' } as any,
            { kind: 'group', depth: 1, key: 'child-key', count: 4 },
          ],
        },
      ],
      { metricAlias: 'count', metricLabel: 'Count' },
    );
    expect(categories).toEqual(['only-key']);
    expect(seriesMatrix).toEqual([{ name: 'child-key', data: [4] }]);
  });
});
