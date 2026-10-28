// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  mergePropertyLabelWrite,
  mergeSelectionLabelsWrite,
  normalizePropertyLabelForStorage,
  parsePropertyLabelMap,
  translatePropertyDefinitionItems,
  unwrapPropertyLabel,
} from './properties_label_i18n';
import { assertValidPropertyDefinitionItems } from './properties_types';

test('parsePropertyLabelMap treats plain string as en_US', () => {
  expect(parsePropertyLabelMap('Priority')).toEqual({ en_US: 'Priority' });
  expect(parsePropertyLabelMap({ en_US: 'Priority', zh_CN: '优先级' })).toEqual({
    en_US: 'Priority',
    zh_CN: '优先级',
  });
  expect(parsePropertyLabelMap(null)).toBeNull();
  expect(parsePropertyLabelMap(['x'])).toBeNull();
});

test('unwrapPropertyLabel falls back to en_US and keeps explicit empty', () => {
  expect(unwrapPropertyLabel({ en_US: 'Priority', zh_CN: '优先级' }, 'zh_CN')).toBe('优先级');
  expect(unwrapPropertyLabel({ en_US: 'Priority' }, 'zh_CN')).toBe('Priority');
  expect(unwrapPropertyLabel({ en_US: 'Priority', zh_CN: '' }, 'zh_CN')).toBe('');
  expect(unwrapPropertyLabel('Priority', 'zh_CN')).toBe('Priority');
  expect(unwrapPropertyLabel(undefined, 'zh_CN')).toBe('');
});

test('normalizePropertyLabelForStorage keeps string and cleans maps', () => {
  expect(normalizePropertyLabelForStorage('Priority')).toBe('Priority');
  expect(normalizePropertyLabelForStorage({ en_US: 'Priority' })).toBe('Priority');
  expect(normalizePropertyLabelForStorage({ en_US: 'Priority', zh_CN: '优先级' })).toEqual({
    en_US: 'Priority',
    zh_CN: '优先级',
  });
  expect(() => normalizePropertyLabelForStorage({ 'zh-CN': 'x' })).toThrow(/terminology code/);
  expect(() => normalizePropertyLabelForStorage({ en_US: { nested: 1 } as any })).toThrow(/must be a string/);
});

test('mergePropertyLabelWrite preserves other langs and seeds en_US', () => {
  expect(mergePropertyLabelWrite({ en_US: 'Priority', zh_CN: '旧' }, '优先级', 'zh_CN')).toEqual({
    en_US: 'Priority',
    zh_CN: '优先级',
  });
  expect(mergePropertyLabelWrite(undefined, '优先级', 'zh_CN')).toEqual({
    en_US: '优先级',
    zh_CN: '优先级',
  });
  expect(mergePropertyLabelWrite('Priority', '优先级', 'zh_CN')).toEqual({
    en_US: 'Priority',
    zh_CN: '优先级',
  });
});

test('mergeSelectionLabelsWrite merges by option value', () => {
  const merged = mergeSelectionLabelsWrite(
    [
      ['high', { en_US: 'High', zh_CN: '高' }],
      ['low', 'Low'],
    ],
    [
      { value: 'high', label: '很高' },
      { value: 'mid', label: '中' },
    ],
    'zh_CN'
  );
  expect(merged).toEqual([
    ['high', { en_US: 'High', zh_CN: '很高' }],
    ['mid', { en_US: '中', zh_CN: '中' }],
  ]);
});

test('translatePropertyDefinitionItems unwraps string, help, and selection labels', () => {
  const items = translatePropertyDefinitionItems(
    [
      {
        name: 'priority',
        type: 'selection',
        string: { en_US: 'Priority', zh_CN: '优先级' },
        help: { en_US: 'Record urgency', zh_CN: '记录紧急程度' },
        selection: [
          ['high', { en_US: 'High', zh_CN: '高' }],
          { value: 'low', label: { en_US: 'Low', zh_CN: '低' } },
        ],
        default: 'high',
      },
    ],
    'zh_CN'
  );
  expect(items[0].string).toBe('优先级');
  expect(items[0].help).toBe('记录紧急程度');
  expect(items[0].default).toBe('high');
  expect(items[0].selection).toEqual([
    ['high', '高'],
    { value: 'low', label: '低' },
  ]);
});

test('assertValidPropertyDefinitionItems accepts lang-map labels and help', () => {
  const out = assertValidPropertyDefinitionItems([
    {
      name: 'priority',
      type: 'selection',
      string: { en_US: 'Priority', zh_CN: '优先级' },
      help: { en_US: 'Record urgency', zh_CN: '记录紧急程度' },
      selection: [['high', { en_US: 'High', zh_CN: '高' }]],
      default: 'high',
    },
    {
      name: 'code',
      type: 'char',
      string: 'External Code',
      help: 'Short unique code used in references',
    },
  ]);
  expect(out[0].string).toEqual({ en_US: 'Priority', zh_CN: '优先级' });
  expect(out[0].help).toEqual({ en_US: 'Record urgency', zh_CN: '记录紧急程度' });
  expect(out[0].selection).toEqual([['high', { en_US: 'High', zh_CN: '高' }]]);
  expect(out[1].string).toBe('External Code');
  expect(out[1].help).toBe('Short unique code used in references');
});

test('assertValidPropertyDefinitionItems rejects invalid label maps', () => {
  expect(() =>
    assertValidPropertyDefinitionItems([{ name: 'x', type: 'char', string: { en_US: { bad: 1 } as any } }])
  ).toThrow(/must be a string/);
});

test('assertValidPropertyDefinitionItems rejects invalid help maps', () => {
  expect(() =>
    assertValidPropertyDefinitionItems([{ name: 'x', type: 'char', help: { en_US: { bad: 1 } as any } }])
  ).toThrow(/must be a string/);
});
