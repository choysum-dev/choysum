// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  defaultListColumnMaxWidth,
  defaultListColumnMinWidth,
  estimateListHeaderMinWidth,
  isListCodeLikeLeaf,
  listColumnIsHard,
  listColumnPrefersStretch,
  snapListColumnWidthUp,
} from './listColumnSizeDefaults';

test('defaultListColumnMinWidth covers the locked 20px scale', () => {
  expect(defaultListColumnMinWidth('boolean')).toBe(80);
  expect(defaultListColumnMinWidth('int')).toBe(80);
  expect(defaultListColumnMinWidth('monetary')).toBe(80);
  expect(defaultListColumnMinWidth('selection')).toBe(100);
  expect(defaultListColumnMinWidth('varchar')).toBe(140);
  expect(defaultListColumnMinWidth('ManyToOne')).toBe(140);
  expect(defaultListColumnMinWidth('datetime')).toBe(160);
  expect(defaultListColumnMinWidth('date')).toBe(160);
  expect(defaultListColumnMinWidth(undefined)).toBeUndefined();
  expect(defaultListColumnMinWidth('OneToMany')).toBeUndefined();
});

test('listColumnPrefersStretch marks text-like and m2o types', () => {
  expect(listColumnPrefersStretch('varchar')).toBe(true);
  expect(listColumnPrefersStretch('char')).toBe(true);
  expect(listColumnPrefersStretch('ManyToOne')).toBe(true);
  expect(listColumnPrefersStretch('int')).toBe(false);
  expect(listColumnPrefersStretch('boolean')).toBe(false);
});

test('defaultListColumnMaxWidth caps soft columns and skips hard types', () => {
  expect(defaultListColumnMaxWidth('varchar', 140)).toBe(350);
  expect(defaultListColumnMaxWidth('ManyToOne', 140)).toBe(350);
  expect(defaultListColumnMaxWidth('text', 140)).toBe(420);
  expect(defaultListColumnMaxWidth('int', 80)).toBeUndefined();
  expect(defaultListColumnMaxWidth('boolean', 80)).toBeUndefined();
  expect(defaultListColumnMaxWidth('datetime', 160)).toBeUndefined();
});

test('code-like leaves are semi-hard (min 100 / max 160)', () => {
  expect(isListCodeLikeLeaf('Code')).toBe(true);
  expect(isListCodeLikeLeaf('Sku')).toBe(true);
  expect(isListCodeLikeLeaf('Ref')).toBe(true);
  expect(isListCodeLikeLeaf('product_code')).toBe(true);
  expect(isListCodeLikeLeaf('Title')).toBe(false);
  expect(defaultListColumnMinWidth('varchar', 'Code')).toBe(100);
  expect(defaultListColumnMinWidth('varchar', 'Lines.Code')).toBe(100);
  expect(defaultListColumnMaxWidth('varchar', 100, 'Code')).toBe(160);
  expect(defaultListColumnMaxWidth('varchar', 140, 'Title')).toBe(350);
  // Author-raised min is not crushed below itself.
  expect(defaultListColumnMaxWidth('varchar', 180, 'Code')).toBe(180);
});

test('listColumnIsHard covers numeric boolean selection date', () => {
  expect(listColumnIsHard('int')).toBe(true);
  expect(listColumnIsHard('selection')).toBe(true);
  expect(listColumnIsHard('datetime')).toBe(true);
  expect(listColumnIsHard('varchar')).toBe(false);
});

test('estimateListHeaderMinWidth raises hard mins on a 20px scale', () => {
  // Force char-budget path (no canvas) for deterministic FE unit runs.
  expect(estimateListHeaderMinWidth('En', 80, { measure: null })).toBe(80);
  expect(estimateListHeaderMinWidth('启用', 80, { measure: null })).toBe(80);
  const long = estimateListHeaderMinWidth('非常长的状态标签名', 80, { measure: null });
  expect(long).toBeGreaterThan(80);
  expect(long % 20).toBe(0);
  expect(long).toBeLessThanOrEqual(80 + 80);
  expect(snapListColumnWidthUp(81)).toBe(100);
  expect(snapListColumnWidthUp(100)).toBe(100);
});
