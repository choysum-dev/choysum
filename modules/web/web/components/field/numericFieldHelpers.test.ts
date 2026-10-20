// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { nextTick, ref } from 'vue';
import {
  defaultListColumnAlignForFieldType,
  formatGroupedNumericString,
  isCenterAlignedListFieldType,
  isEndAlignedListFieldType,
  isFormEmptyPlaceholderMode,
  resolveNumericInputEl,
  resyncControlledNumericRaw,
} from './numericFieldHelpers';

describe('numericFieldHelpers', () => {
  test('resyncControlledNumericRaw restores DOM when browser mutated ahead of model', () => {
    const editingRaw = ref<string | null>('42');
    const input = document.createElement('input');
    input.value = '42abc';
    resyncControlledNumericRaw(editingRaw, input);
    expect(input.value).toBe('42');
    expect(editingRaw.value).toBe('42');
  });

  test('resyncControlledNumericRaw bounces controlled prop when element is unavailable', async () => {
    const editingRaw = ref<string | null>('7');
    resyncControlledNumericRaw(editingRaw, null);
    expect(editingRaw.value).toBeNull();
    await nextTick();
    expect(editingRaw.value).toBe('7');
  });

  test('resolveNumericInputEl unwraps component host and native element', () => {
    const input = document.createElement('input');
    expect(resolveNumericInputEl(input)).toBe(input);
    expect(resolveNumericInputEl({ $el: input })).toBe(input);
    expect(resolveNumericInputEl(null)).toBeNull();
    expect(resolveNumericInputEl({ $el: document.createElement('div') })).toBeNull();
  });

  test('formatGroupedNumericString applies i18n separators', () => {
    expect(
      formatGroupedNumericString('1234567.89', {
        thousandsSeparator: ',',
        decimalSeparator: '.',
        grouping: [3, 0],
      })
    ).toBe('1,234,567.89');
    expect(formatGroupedNumericString('42', null)).toBe('42');
    expect(formatGroupedNumericString('', { thousandsSeparator: ',' })).toBe('');
  });

  test('isFormEmptyPlaceholderMode hides empty text in table/inline', () => {
    expect(isFormEmptyPlaceholderMode('form', false)).toBe(true);
    expect(isFormEmptyPlaceholderMode('table', true)).toBe(false);
    expect(isFormEmptyPlaceholderMode('inline', true)).toBe(false);
    expect(isFormEmptyPlaceholderMode('auto', true)).toBe(true);
    expect(isFormEmptyPlaceholderMode('auto', false)).toBe(false);
    expect(isFormEmptyPlaceholderMode(undefined, undefined)).toBe(true);
  });

  test('list column align defaults by field type family', () => {
    for (const t of ['int', 'integer', 'bigint', 'float', 'double', 'number', 'decimal', 'monetary', 'INT']) {
      expect(isEndAlignedListFieldType(t)).toBe(true);
      expect(defaultListColumnAlignForFieldType(t)).toBe('end');
    }
    for (const t of ['boolean', 'binary', 'image', 'BOOLEAN']) {
      expect(isCenterAlignedListFieldType(t)).toBe(true);
      expect(defaultListColumnAlignForFieldType(t)).toBe('center');
    }
    expect(isEndAlignedListFieldType('varchar')).toBe(false);
    expect(isCenterAlignedListFieldType('selection')).toBe(false);
    expect(defaultListColumnAlignForFieldType('char')).toBeUndefined();
    expect(defaultListColumnAlignForFieldType('selection')).toBeUndefined();
    expect(defaultListColumnAlignForFieldType('')).toBeUndefined();
  });
});
