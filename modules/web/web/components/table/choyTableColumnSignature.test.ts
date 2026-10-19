// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  onTableColumnSignatureChange,
  skipTableColumnReregister,
  tableColumnPropSignature,
} from './choyTableColumnSignature';

test('tableColumnPropSignature snapshots primitives', () => {
  expect(
    tableColumnPropSignature({
      type: null,
      prop: 'Name',
      vColumnProps: { align: 'left' },
    }),
  ).toBe(
    tableColumnPropSignature({
      type: null,
      prop: 'Name',
      vColumnProps: { align: 'left' },
    }),
  );
  expect(tableColumnPropSignature({ prop: 'A' })).not.toBe(tableColumnPropSignature({ prop: 'B' }));
});

test('skipTableColumnReregister requires a current column and an equal signature', () => {
  expect(skipTableColumnReregister(false, 'a', 'a')).toBe(false);
  expect(skipTableColumnReregister(true, 'a', 'b')).toBe(false);
  expect(skipTableColumnReregister(true, 'a', 'a')).toBe(true);
});

test('onTableColumnSignatureChange skips equal signatures', () => {
  let n = 0;
  onTableColumnSignatureChange('a', 'a', () => {
    n += 1;
  });
  onTableColumnSignatureChange('b', 'a', () => {
    n += 1;
  });
  expect(n).toBe(1);
});
