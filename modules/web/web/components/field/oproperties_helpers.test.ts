// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ResolvedPropertyItem } from '@/core/service/orm/model/properties_types';
import {
  buildFullPropertiesMap,
  countSchemaMapIntersection,
  filterRenderablePropertyItems,
  normalizeSelectionOptions,
  propertiesFieldKey,
  propertyDatetimeFromPicker,
  propertyDatetimeToPicker,
  writePropertyValue,
} from './oproperties_helpers';

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

describe('oproperties_helpers', () => {
  test('resolves field keys for RPC and DOM ids', () => {
    expect(propertiesFieldKey('A', 'B')).toBe('A');
    expect(propertiesFieldKey('', 'B')).toBe('B');
    expect(propertiesFieldKey('', '', 'properties')).toBe('properties');
    expect(propertiesFieldKey(null, undefined, '')).toBe('');
  });

  test('filters unknown / empty items and normalizes selection options', () => {
    expect(filterRenderablePropertyItems(null as any)).toEqual({ renderable: [], skipped: [] });
    const { renderable, skipped } = filterRenderablePropertyItems([
      null as any,
      { name: '', type: 'char' },
      { name: 'ok', type: 'char' },
      { name: 'bad', type: 'html' },
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
    expect(skipped.map(i => i.name)).toEqual(['bad', 'html_x']);
    expect(normalizeSelectionOptions(undefined)).toEqual([]);
    expect(normalizeSelectionOptions([123, { label: 'x' }, ['only'], ['v'], { value: 'c' }])).toEqual([
      { value: 'only', label: 'only' },
      { value: 'v', label: 'v' },
      { value: 'c', label: 'c' },
    ]);
  });

  test('counts schema∩map and builds / writes full replace maps', () => {
    expect(countSchemaMapIntersection(['active', 'code'], { active: true, orphan: 1 })).toBe(1);
    expect(countSchemaMapIntersection([], { active: true })).toBe(0);
    const map = buildFullPropertiesMap(
      [
        null as any,
        { name: '', type: 'char' },
        { name: 'skip', type: 'html' },
        { name: 'fromPrev', type: 'char' },
        { name: 'fromValue', type: 'char', value: 'V' },
        { name: 'fromDefault', type: 'char', default: 'D' },
        { name: 'empty', type: 'char' },
      ],
      { fromPrev: 'P', orphan: 9 }
    );
    expect(map).toEqual({ fromPrev: 'P', fromValue: 'V', fromDefault: 'D' });
    expect(writePropertyValue([{ name: 'code', type: 'char' }], { code: 'A' }, 'code', 'B2')).toEqual({
      code: 'B2',
    });
    expect(
      writePropertyValue(
        [
          { name: 'fromPrev', type: 'char' },
          { name: 'html_x', type: 'html' },
        ],
        map,
        'html_x',
        '<x/>'
      )
    ).toEqual({ fromPrev: 'P' });

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
});
