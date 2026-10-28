// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Web-side imports of properties_label_i18n so FE coverage (and Codecov patch)
 * records branch lines that core BE instrumentation under-counts.
 */
import {
  mergePropertyLabelWrite,
  mergeSelectionLabelsWrite,
  normalizePropertyLabelForStorage,
  normalizeSelectionLabelsForStorage,
  parsePropertyLabelMap,
  selectionOptionLabel,
  selectionOptionValue,
  translatePropertyDefinitionItems,
  unwrapPropertyLabel,
} from '@/core/service/orm/model/properties_label_i18n';

describe('properties_label_i18n web coverage', () => {
  test('parsePropertyLabelMap null-value and skip branches', () => {
    expect(parsePropertyLabelMap({ en_US: 'A', zh_CN: null, 'zh-CN': 'skip', '': 'skip' })).toEqual({
      en_US: 'A',
      zh_CN: '',
    });
    expect(parsePropertyLabelMap('plain')).toEqual({ en_US: 'plain' });
    expect(parsePropertyLabelMap(null)).toBeNull();
  });

  test('normalizePropertyLabelForStorage string, throws, and null map values', () => {
    expect(normalizePropertyLabelForStorage('Hello')).toBe('Hello');
    expect(normalizePropertyLabelForStorage({ en_US: null, zh_CN: '你好' })).toEqual({
      en_US: '',
      zh_CN: '你好',
    });
    expect(() => normalizePropertyLabelForStorage(12 as any)).toThrow(/string or lang map/);
    expect(() => normalizePropertyLabelForStorage(['x'] as any)).toThrow(/string or lang map/);
    expect(() => normalizePropertyLabelForStorage({ en_US: { bad: 1 } as any })).toThrow(/must be a string/);
  });

  test('selection option helpers via normalize, merge, and direct exports', () => {
    expect(selectionOptionValue({ value: 'low' })).toBe('low');
    expect(selectionOptionValue(['high', 'High'])).toBe('high');
    expect(selectionOptionValue(42)).toBeNull();
    expect(selectionOptionLabel(['a', 'A'])).toBe('A');
    expect(selectionOptionLabel({ value: 'a', label: 'A' })).toBe('A');
    expect(selectionOptionLabel('scalar')).toBeUndefined();
    expect(selectionOptionLabel(null)).toBeUndefined();

    expect(
      normalizeSelectionLabelsForStorage(
        [
          { value: 'low', label: 'Low' },
          ['high', 'High'],
          ['solo'],
        ],
        'prio',
      ),
    ).toEqual([
      { value: 'low', label: 'Low' },
      ['high', 'High'],
      ['solo', 'solo'],
    ]);
    expect(() => normalizeSelectionLabelsForStorage(undefined as any, 'prio')).toThrow(/invalid/);
    expect(() => normalizeSelectionLabelsForStorage('nope' as any, 'prio')).toThrow(/invalid/);
    expect(() => normalizeSelectionLabelsForStorage([null], 'prio')).toThrow(/invalid/);
    expect(() => normalizeSelectionLabelsForStorage([42], 'prio')).toThrow(/invalid/);

    expect(
      mergeSelectionLabelsWrite(
        [{ value: 'a', label: 'A' }, { value: 1 as any, label: 'skip' }, 'scalar' as any],
        [{ value: 'a', label: '甲' }],
        'zh_CN',
      ),
    ).toEqual([['a', { en_US: 'A', zh_CN: '甲' }]]);
  });

  test('unwrap / merge / translate smoke for import graph', () => {
    expect(unwrapPropertyLabel({ en_US: 'X', zh_CN: null as any }, 'zh_CN')).toBe('');
    expect(mergePropertyLabelWrite(undefined, 'Y', 'zh_CN')).toEqual({ en_US: 'Y', zh_CN: 'Y' });
    const translated = translatePropertyDefinitionItems(
      [
        {
          name: 'n',
          type: 'selection',
          string: 'N',
          help: { en_US: 'Help', zh_CN: '帮助' },
          selection: [
            ['a', 'A'],
            { value: 'b', label: 'B' },
            'passthrough' as any,
            [1 as any, 'skip'],
          ],
        },
      ],
      'zh_CN',
    );
    expect(translated[0].help).toBe('帮助');
    expect(translated[0].selection).toEqual([
      ['a', 'A'],
      { value: 'b', label: 'B' },
      'passthrough',
      [1, 'skip'],
    ]);
  });
});
