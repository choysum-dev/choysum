// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  buildFullPropertiesMap,
  countSchemaMapIntersection,
  filterRenderablePropertyItems,
  normalizeSelectionOptions,
  propertiesFieldKey,
  writePropertyValue,
} from './propertiesHelpers';

describe('propertiesHelpers', () => {
  test('propertiesFieldKey falls back', () => {
    expect(propertiesFieldKey('', '', 'properties')).toBe('properties');
    expect(propertiesFieldKey('Props', '', 'properties')).toBe('Props');
  });

  test('filterRenderablePropertyItems skips unknown types', () => {
    const { renderable, skipped } = filterRenderablePropertyItems([
      { name: 'a', type: 'char' },
      { name: 'b', type: 'binary' },
      { name: '', type: 'char' },
      null as unknown as { name: string; type: string },
    ]);
    expect(renderable.map(i => i.name)).toEqual(['a']);
    expect(skipped.map(i => i.name)).toEqual(['b']);
  });

  test('normalizeSelectionOptions accepts tuple and object forms', () => {
    expect(normalizeSelectionOptions(null)).toEqual([]);
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
      { name: '', type: 'char', default: 'x' },
      { name: 'skip', type: 'binary', default: 'x' },
    ];
    const map = buildFullPropertiesMap(items, { color: 'blue' });
    expect(map.color).toBe('blue');
    expect(map.qty).toBe(3);
    expect(Object.prototype.hasOwnProperty.call(map, 'skip')).toBe(false);
    const next = writePropertyValue(items, map, 'qty', 9);
    expect(next.qty).toBe(9);
    // Unknown schema key → no write.
    expect(writePropertyValue(items, map, 'missing', 1)).toEqual(map);
    expect(countSchemaMapIntersection(['color', 'qty', 'missing'], next)).toBe(2);
    expect(countSchemaMapIntersection([], next)).toBe(0);
  });
});
