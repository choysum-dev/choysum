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
});
