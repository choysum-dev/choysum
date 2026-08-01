// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createTranslate } from '../../i18n';
import { mergeSelectionByValue } from './selection_merge';

test('mergeSelectionByValue appends and overrides by value (PR-P2-F4)', () => {
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

test('mergeSelectionByValue plain label clears inherited labelText (PR-P2-F4)', () => {
  const labelText = createTranslate('demo', { scope: 'demo.status' })._lt('Done');
  expect(
    mergeSelectionByValue([{ value: 'done', label: 'Done', labelText }], [{ value: 'done', label: 'Finished' }])
  ).toEqual([{ value: 'done', label: 'Finished' }]);
});
