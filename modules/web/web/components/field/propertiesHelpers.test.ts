// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ResolvedPropertyItem } from '@/core/service/orm/model/properties_types';
import {
  buildFullPropertiesMap,
  countSchemaMapIntersection,
  filterRenderablePropertyItems,
  normalizeSelectionOptions,
  propertiesFieldKey,
  propertyDateFromInput,
  propertyDateToInput,
  propertyDatetimeFromPicker,
  propertyDatetimeToPicker,
  writePropertyValue,
} from './propertiesHelpers';

const schemaItems: ResolvedPropertyItem[] = [
  { name: 'active', type: 'boolean', string: 'Active', value: true },
  { name: 'code', type: 'char', string: 'Code', value: 'A1' },
  { name: 'qty', type: 'integer', string: 'Qty', value: 2 },
  { name: 'amount', type: 'float', string: 'Amount', value: 1.5 },
  { name: 'note', type: 'text', string: 'Note' },
  { name: 'day', type: 'date', string: 'Day', value: '2024-01-02T00:00:00Z' },
  {
    name: 'when',
    type: 'datetime',
    string: 'When',
    value: '2024-06-30T16:00:00.000Z',
  },
  {
    name: 'kind',
    type: 'selection',
    string: 'Kind',
    selection: [
      ['a', 'Alpha'],
      { value: 'b', label: 'Beta' },
    ],
  },
  { name: 'html_x', type: 'html', string: 'Bad' },
];

describe('propertiesHelpers', () => {
  test('propertiesFieldKey falls back', () => {
    expect(propertiesFieldKey('A', 'B')).toBe('A');
    expect(propertiesFieldKey('', 'B')).toBe('B');
    expect(propertiesFieldKey('', '', 'properties')).toBe('properties');
    expect(propertiesFieldKey(null, undefined, '')).toBe('');
  });

  test('filterRenderablePropertyItems skips unknown types', () => {
    expect(filterRenderablePropertyItems(null as any)).toEqual({ renderable: [], skipped: [] });
    const { renderable, skipped } = filterRenderablePropertyItems([
      null as any,
      { name: '', type: 'char' },
      { name: 'ok', type: 'char' },
      { name: 'bad', type: 'html' },
      { name: 'b', type: 'binary' },
      ...schemaItems,
    ]);
    expect(renderable.map(i => i.name)).toEqual([
      'ok',
      'active',
      'code',
      'qty',
      'amount',
      'note',
      'day',
      'when',
      'kind',
    ]);
    expect(skipped.map(i => i.name)).toEqual(['bad', 'b', 'html_x']);
  });

  test('normalizeSelectionOptions accepts tuple and object forms', () => {
    expect(normalizeSelectionOptions(null)).toEqual([]);
    expect(normalizeSelectionOptions(undefined)).toEqual([]);
    expect(normalizeSelectionOptions([['x', 'X'], { value: 'y', label: 'Y' }, 1, { value: 2 }])).toEqual([
      { value: 'x', label: 'X' },
      { value: 'y', label: 'Y' },
    ]);
    expect(normalizeSelectionOptions([['only']])).toEqual([{ value: 'only', label: 'only' }]);
  });

  test('buildFullPropertiesMap and writePropertyValue', () => {
    const items = [
      { name: 'color', type: 'char', default: 'red' },
      { name: 'qty', type: 'integer', value: 3 },
      { name: 'tier', type: 'char', default: 'bronze' },
      { name: '', type: 'char', default: 'x' },
      { name: 'skip', type: 'binary', default: 'x' },
    ];
    // color is present in previous → prev wins over default.
    // qty has value and no previous → value wins.
    // tier has only default and no previous → default branch.
    const map = buildFullPropertiesMap(items, { color: 'blue' });
    expect(map.color).toBe('blue');
    expect(map.qty).toBe(3);
    expect(map.tier).toBe('bronze');
    // Non-V1 with no previous value is omitted.
    expect(Object.prototype.hasOwnProperty.call(map, 'skip')).toBe(false);
    const next = writePropertyValue(items, map, 'qty', 9);
    expect(next.qty).toBe(9);
    expect(writePropertyValue(items, map, 'missing', 1)).toEqual(map);
    expect(countSchemaMapIntersection(['color', 'qty', 'missing'], next)).toBe(2);
    expect(countSchemaMapIntersection([], next)).toBe(0);
  });

  test('buildFullPropertiesMap preserves previous values for unrenderable types', () => {
    const items = [
      { name: 'color', type: 'char', default: 'red' },
      { name: 'blob', type: 'binary' },
    ];
    const map = buildFullPropertiesMap(items, { color: 'blue', blob: 'deadbeef' });
    expect(map.color).toBe('blue');
    expect(map.blob).toBe('deadbeef');
    const afterWrite = writePropertyValue(items, map, 'color', 'green');
    expect(afterWrite.color).toBe('green');
    expect(afterWrite.blob).toBe('deadbeef');
    // Writes to non-V1 keys are ignored; previous value stays.
    expect(writePropertyValue(items, map, 'blob', 'new')).toEqual(map);
  });

  test('buildFullPropertiesMap uses null-prototype maps for __proto__ schema names', () => {
    const protoPrev = JSON.parse('{"__proto__":"from-prev"}');
    const protoMap = buildFullPropertiesMap([{ name: '__proto__', type: 'char' }], protoPrev);
    expect(Object.prototype.hasOwnProperty.call(protoMap, '__proto__')).toBe(true);
    expect(protoMap['__proto__']).toBe('from-prev');
    expect(Object.getPrototypeOf(protoMap)).toBe(null);
    const written = writePropertyValue([{ name: '__proto__', type: 'char' }], {}, '__proto__', 'safe');
    expect(Object.prototype.hasOwnProperty.call(written, '__proto__')).toBe(true);
    expect(written['__proto__']).toBe('safe');
  });

  test('converts datetime through the UTC wall-clock codec', () => {
    expect(propertyDatetimeToPicker(null)).toBeNull();
    expect(propertyDatetimeToPicker('')).toBeNull();
    expect(propertyDatetimeToPicker({ x: 1 })).toBeNull();
    const fromDate = propertyDatetimeToPicker(new Date('2024-06-30T16:00:00.000Z'), 'UTC');
    expect(fromDate).toBeInstanceOf(Date);
    const fromMs = propertyDatetimeToPicker(Date.parse('2024-06-30T16:00:00.000Z'), 'UTC');
    expect(fromMs).toBeInstanceOf(Date);
    const wall = propertyDatetimeToPicker('2024-06-30T16:00:00.000Z', 'America/New_York');
    expect(wall!.getHours()).toBe(12);
    expect(propertyDatetimeFromPicker(wall, 'America/New_York')).toBe('2024-06-30T16:00:00.000Z');
    expect(propertyDatetimeFromPicker(null)).toBeNull();
    expect(propertyDatetimeFromPicker('')).toBeNull();
    expect(propertyDatetimeFromPicker('2024-06-30T12:00:00.000Z', 'UTC')).toMatch(/Z$/);
    expect(propertyDatetimeFromPicker(new Date('invalid'), 'UTC')).toBeNull();
  });

  test('propertyDateToInput / propertyDateFromInput round-trip ISO midnight', () => {
    expect(propertyDateToInput(null)).toBe('');
    expect(propertyDateToInput('')).toBe('');
    expect(propertyDateToInput('2024-01-01T00:00:00Z')).toBe('2024-01-01');
    expect(propertyDateToInput('2024-01-01')).toBe('2024-01-01');
    expect(propertyDateToInput(new Date(Date.UTC(2024, 0, 15)))).toBe('2024-01-15');
    expect(propertyDateToInput(new Date('invalid'))).toBe('');
    expect(propertyDateToInput({ x: 1 })).toBe('');
    expect(propertyDateFromInput('2024-01-01')).toBe('2024-01-01T00:00:00Z');
    expect(propertyDateFromInput('')).toBeNull();
    expect(propertyDateFromInput(null)).toBeNull();
    expect(propertyDateFromInput('not-a-date')).toBeNull();
  });
});
