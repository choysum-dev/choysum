// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createTranslate } from '../../i18n';
import { mergeSelectionByValue } from './selection_merge';

test('mergeSelectionByValue appends and overrides by value', () => {
  expect(
    mergeSelectionByValue(
      [
        { value: 'a', label: 'A' },
        { value: 'b', label: 'B' },
      ],
      [
        { value: 'c', label: 'C' },
        { value: 'b', label: 'B2' },
      ]
    )
  ).toEqual([
    { value: 'a', label: 'A' },
    { value: 'b', label: 'B2' },
    { value: 'c', label: 'C' },
  ]);
});

test('mergeSelectionByValue rejects leading or trailing whitespace in values', () => {
  expect(() =>
    mergeSelectionByValue([{ value: 'a', label: 'A' }], [{ value: ' vip ', label: 'VIP' } as any])
  ).toThrow(/leading or trailing whitespace/);
  expect(() =>
    mergeSelectionByValue([{ value: '  ', label: 'skip' } as any], [{ value: 'c', label: 'C' }])
  ).toThrow(/leading or trailing whitespace/);
  expect(() =>
    mergeSelectionByValue([{ value: 'a', label: 'A' }], [{ value: '', label: 'empty' } as any])
  ).toThrow(/non-empty/);
});

test('mergeSelectionByValue plain label clears inherited labelText', () => {
  const labelText = createTranslate('demo', { scope: 'demo.status' })._lt('Done');
  expect(
    mergeSelectionByValue([{ value: 'done', label: 'Done', labelText }], [{ value: 'done', label: 'Finished' }])
  ).toEqual([{ value: 'done', label: 'Finished' }]);
});

test('mergeSelectionByValue keeps addend labelText and tolerates empty base', () => {
  const labelText = createTranslate('demo', { scope: 'demo.status' })._lt('VIP');
  expect(mergeSelectionByValue(undefined, [{ value: 'vip', label: 'VIP', labelText }])).toEqual([
    { value: 'vip', label: 'VIP', labelText },
  ]);
  expect(mergeSelectionByValue(null as any, [])).toEqual([]);
  expect(
    mergeSelectionByValue(
      [null as any, { value: undefined as any, label: 'X' }, { value: 'a', label: 'A' }, { value: 'a', label: 'A2' }],
      [null as any, { value: undefined as any, label: 'Y' }, { value: 'b', label: 'B' }]
    )
  ).toEqual([
    { value: 'a', label: 'A2' },
    { value: 'b', label: 'B' },
  ]);
});
